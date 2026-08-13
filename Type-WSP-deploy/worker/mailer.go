package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

const (
	verificationEmailSubject   = "Type-WSP 驗證碼"
	loginOwnershipEmailSubject = "Type-WSP 登入安全驗證"
)

type MailSender interface {
	SendVerificationCode(context.Context, string, string) error
	SendLoginOwnershipCode(context.Context, string, string, time.Time) error
}

type SMTPMailSender struct {
	host     string
	port     int
	from     string
	username string
	password string
	startTLS bool
	timeout  time.Duration
}

func NewSMTPMailSender(cfg *Config) *SMTPMailSender {
	return &SMTPMailSender{
		host: cfg.SMTPHost, port: cfg.SMTPPort, from: cfg.SMTPFrom,
		username: cfg.SMTPUsername, password: cfg.SMTPPassword,
		startTLS: cfg.SMTPSecure, timeout: 10 * time.Second,
	}
}

func (sender *SMTPMailSender) SendVerificationCode(ctx context.Context, recipient, code string) error {
	if !validVerificationEmailCode(code) {
		return fmt.Errorf("verification code has an invalid format")
	}
	return sender.sendMessage(ctx, recipient, verificationEmailMessage(sender.from, recipient, code))
}

func (sender *SMTPMailSender) SendLoginOwnershipCode(ctx context.Context, recipient, code string, expiresAt time.Time) error {
	if !validLoginOwnershipCode(code) {
		return fmt.Errorf("login ownership code has an invalid format")
	}
	if expiresAt.IsZero() {
		return fmt.Errorf("login ownership code expiry is required")
	}
	return sender.sendMessage(ctx, recipient, loginOwnershipEmailMessage(sender.from, recipient, code, expiresAt))
}

func (sender *SMTPMailSender) sendMessage(ctx context.Context, recipient, message string) error {
	recipientAddress, err := mail.ParseAddress(recipient)
	if err != nil {
		return fmt.Errorf("invalid recipient address: %w", err)
	}
	fromAddress, err := mail.ParseAddress(sender.from)
	if err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	address := net.JoinHostPort(sender.host, fmt.Sprintf("%d", sender.port))
	dialer := net.Dialer{Timeout: sender.timeout}
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("connect SMTP server: %w", err)
	}
	defer connection.Close()

	deadline := time.Now().Add(sender.timeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := connection.SetDeadline(deadline); err != nil {
		return fmt.Errorf("set SMTP deadline: %w", err)
	}

	client, err := smtp.NewClient(connection, sender.host)
	if err != nil {
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer client.Close()

	if sender.startTLS {
		if supported, _ := client.Extension("STARTTLS"); !supported {
			return fmt.Errorf("SMTP server does not support STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: sender.host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("start SMTP TLS: %w", err)
		}
	}

	if sender.username != "" {
		auth := smtp.PlainAuth("", sender.username, sender.password, sender.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authenticate SMTP client: %w", err)
		}
	}
	if err := client.Mail(fromAddress.Address); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := client.Rcpt(recipientAddress.Address); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}

	body, err := client.Data()
	if err != nil {
		return fmt.Errorf("open SMTP message body: %w", err)
	}
	writer := bufio.NewWriter(body)
	if _, err := writer.WriteString(message); err != nil {
		body.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := writer.Flush(); err != nil {
		body.Close()
		return fmt.Errorf("flush SMTP message: %w", err)
	}
	if err := body.Close(); err != nil {
		return fmt.Errorf("send SMTP message: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("close SMTP session: %w", err)
	}
	return nil
}

func validVerificationEmailCode(code string) bool {
	if len(code) == 6 {
		for _, char := range code {
			if char < '0' || char > '9' {
				return false
			}
		}
		return true
	}
	return validLoginOwnershipCode(code)
}

func validLoginOwnershipCode(code string) bool {
	if len(code) != 16 {
		return false
	}
	const highEntropyEmailCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	for _, char := range code {
		if !strings.ContainsRune(highEntropyEmailCodeAlphabet, char) {
			return false
		}
	}
	return true
}

func verificationEmailMessage(from, recipient, code string) string {
	subject := mime.QEncoding.Encode("UTF-8", verificationEmailSubject)
	expiryMessage := "此驗證碼將在 5 分鐘後失效。"
	if len(code) == 16 {
		expiryMessage = "此註冊驗證碼將在 24 小時後失效。"
	}
	lines := []string{
		"From: " + from,
		"To: " + recipient,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		"你的驗證碼是：" + code,
		"",
		expiryMessage,
		"如果不是你本人操作，請忽略這封信。",
		"",
	}
	return strings.Join(lines, "\r\n")
}

func loginOwnershipEmailMessage(from, recipient, code string, expiresAt time.Time) string {
	subject := mime.QEncoding.Encode("UTF-8", loginOwnershipEmailSubject)
	lines := []string{
		"From: " + from,
		"To: " + recipient,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		"你的登入安全驗證碼是：" + code,
		"",
		"此短效驗證碼將於 " + expiresAt.UTC().Format(time.RFC3339Nano) + "（UTC）失效。",
		"僅用於登入前確認此信箱由你持有。",
		"若非本人操作，請忽略這封信。",
		"",
	}
	return strings.Join(lines, "\r\n")
}

func maskEmail(value string) string {
	at := strings.LastIndex(value, "@")
	if at <= 0 {
		return "***"
	}
	local := []rune(value[:at])
	visible := 1
	if len(local) > 2 {
		visible = 2
	}
	return string(local[:visible]) + "***" + value[at:]
}
