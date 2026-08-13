package main

import (
	"bufio"
	"context"
	"fmt"
	"mime"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestVerificationEmailMessageUsesCRLFAndRequiredHeaders(t *testing.T) {
	message := verificationEmailMessage("no-reply@type-wsp.local", "user@example.test", "123456")
	for _, expected := range []string{
		"From: no-reply@type-wsp.local\r\n",
		"To: user@example.test\r\n",
		"Subject: ",
		"Content-Type: text/plain; charset=UTF-8\r\n",
		"你的驗證碼是：123456",
		"此驗證碼將在 5 分鐘後失效。",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("message does not contain %q", expected)
		}
	}
}

func TestRegistrationVerificationEmailUsesRegistrationExpiry(t *testing.T) {
	message := verificationEmailMessage("no-reply@type-wsp.local", "user@example.test", "ABCDEFGHJKLMNPQR")
	if !strings.Contains(message, "24 小時") {
		t.Fatalf("registration email does not describe its 24-hour expiry: %q", message)
	}
}

func TestLoginOwnershipEmailMessageStatesPurposeAndExactExpiry(t *testing.T) {
	expiresAt := time.Date(2027, time.January, 2, 3, 4, 5, 0, time.UTC)
	message := loginOwnershipEmailMessage(
		"no-reply@type-wsp.local",
		"user@example.test",
		"ABCDEFGHJKLMNPQR",
		expiresAt,
	)
	for _, expected := range []string{
		mime.QEncoding.Encode("UTF-8", "Type-WSP 登入安全驗證"),
		"你的登入安全驗證碼是：ABCDEFGHJKLMNPQR",
		"此短效驗證碼將於 2027-01-02T03:04:05Z（UTC）失效。",
		"僅用於登入前確認此信箱由你持有。",
		"若非本人操作，請忽略這封信。",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("login ownership message does not contain %q: %q", expected, message)
		}
	}
	if strings.Contains(message, "註冊") || strings.Contains(strings.ToLower(message), "registration") {
		t.Fatalf("login ownership message is mislabeled as registration: %q", message)
	}
}

func TestMaskEmailDoesNotExposeFullAddress(t *testing.T) {
	if got := maskEmail("quality@example.test"); got != "qu***@example.test" {
		t.Fatalf("maskEmail() = %q", got)
	}
	if got := maskEmail("invalid"); got != "***" {
		t.Fatalf("maskEmail(invalid) = %q", got)
	}
}

func TestSMTPMailSenderDeliversVerificationMessage(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { listener.Close() })

	messages := make(chan string, 1)
	serverErrors := make(chan error, 1)
	go serveOneSMTPMessage(listener, messages, serverErrors)

	host, rawPort, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(rawPort)
	sender := &SMTPMailSender{
		host: host, port: port, from: "no-reply@type-wsp.local", timeout: 3 * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := sender.SendVerificationCode(ctx, "user@example.test", "123456"); err != nil {
		t.Fatalf("SendVerificationCode: %v", err)
	}

	select {
	case err := <-serverErrors:
		t.Fatalf("SMTP server: %v", err)
	case message := <-messages:
		if !strings.Contains(message, "你的驗證碼是：123456") {
			t.Fatalf("message does not contain verification code: %q", message)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for SMTP message")
	}
}

func TestSMTPMailSenderDeliversLoginOwnershipMessage(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { listener.Close() })

	messages := make(chan string, 1)
	serverErrors := make(chan error, 1)
	go serveOneSMTPMessage(listener, messages, serverErrors)

	host, rawPort, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(rawPort)
	sender := &SMTPMailSender{
		host: host, port: port, from: "no-reply@type-wsp.local", timeout: 3 * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	expiresAt := time.Date(2027, time.January, 2, 3, 4, 5, 0, time.UTC)
	if err := sender.SendLoginOwnershipCode(ctx, "user@example.test", "ABCDEFGHJKLMNPQR", expiresAt); err != nil {
		t.Fatalf("SendLoginOwnershipCode: %v", err)
	}

	select {
	case err := <-serverErrors:
		t.Fatalf("SMTP server: %v", err)
	case message := <-messages:
		if !strings.Contains(message, "你的登入安全驗證碼是：ABCDEFGHJKLMNPQR") ||
			!strings.Contains(message, "2027-01-02T03:04:05Z（UTC）") {
			t.Fatalf("message does not describe login ownership verification: %q", message)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for SMTP message")
	}
}

func TestVerificationCodeValidationAcceptsRegistrationAndLoginFormats(t *testing.T) {
	for _, code := range []string{"123456", "ABCDEFGHJKLMNPQR"} {
		if !validVerificationEmailCode(code) {
			t.Fatalf("valid verification code rejected: %q", code)
		}
	}
	for _, code := range []string{"12345", "abcdefghijklmnop", "ABCDEFGHILMNPQRO"} {
		if validVerificationEmailCode(code) {
			t.Fatalf("invalid verification code accepted: %q", code)
		}
	}
}

func TestLoginOwnershipCodeValidationRequiresHighEntropyFormat(t *testing.T) {
	if !validLoginOwnershipCode("ABCDEFGHJKLMNPQR") {
		t.Fatal("valid login ownership code rejected")
	}
	for _, code := range []string{"123456", "abcdefghjklmnpqr", "ABCDEFGHILMNPQRS", "ABCDEFGHJKLMNPQ"} {
		if validLoginOwnershipCode(code) {
			t.Fatalf("invalid login ownership code accepted: %q", code)
		}
	}
}

func serveOneSMTPMessage(listener net.Listener, messages chan<- string, serverErrors chan<- error) {
	connection, err := listener.Accept()
	if err != nil {
		serverErrors <- err
		return
	}
	defer connection.Close()
	reader := bufio.NewReader(connection)
	writer := bufio.NewWriter(connection)
	write := func(response string) error {
		if _, err := writer.WriteString(response + "\r\n"); err != nil {
			return err
		}
		return writer.Flush()
	}
	if err := write("220 localhost ESMTP"); err != nil {
		serverErrors <- err
		return
	}

	var message strings.Builder
	dataMode := false
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			serverErrors <- err
			return
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if dataMode {
			if trimmed == "." {
				dataMode = false
				messages <- message.String()
				if err := write("250 queued"); err != nil {
					serverErrors <- err
					return
				}
				continue
			}
			message.WriteString(line)
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "EHLO"):
			if _, err := writer.WriteString("250-localhost\r\n250 8BITMIME\r\n"); err != nil {
				serverErrors <- err
				return
			}
			if err := writer.Flush(); err != nil {
				serverErrors <- err
				return
			}
		case strings.HasPrefix(trimmed, "MAIL FROM:"), strings.HasPrefix(trimmed, "RCPT TO:"):
			if err := write("250 ok"); err != nil {
				serverErrors <- err
				return
			}
		case trimmed == "DATA":
			dataMode = true
			if err := write("354 end with dot"); err != nil {
				serverErrors <- err
				return
			}
		case trimmed == "QUIT":
			if err := write("221 bye"); err != nil {
				serverErrors <- err
			}
			return
		default:
			serverErrors <- fmt.Errorf("unexpected SMTP command %q", trimmed)
			return
		}
	}
}
