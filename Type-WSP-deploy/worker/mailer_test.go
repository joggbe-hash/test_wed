package main

import (
	"bufio"
	"context"
	"fmt"
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
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("message does not contain %q", expected)
		}
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
