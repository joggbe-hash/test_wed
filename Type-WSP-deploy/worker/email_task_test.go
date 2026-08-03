package main

import (
	"context"
	"errors"
	"testing"
)

type fakeMailSender struct {
	recipient string
	code      string
	err       error
}

func (sender *fakeMailSender) SendVerificationCode(_ context.Context, recipient, code string) error {
	sender.recipient = recipient
	sender.code = code
	return sender.err
}

func TestHandleSendEmailUsesConfiguredSender(t *testing.T) {
	previous := mailSender
	t.Cleanup(func() { mailSender = previous })
	fake := &fakeMailSender{}
	mailSender = fake

	err := handleSendEmail(context.Background(), EmailPayload{Email: "user@example.test", Code: "123456"})
	if err != nil {
		t.Fatalf("handleSendEmail: %v", err)
	}
	if fake.recipient != "user@example.test" || fake.code != "123456" {
		t.Fatalf("sender received recipient=%q code=%q", fake.recipient, fake.code)
	}
}

func TestHandleSendEmailReturnsSenderFailureForQueueRetry(t *testing.T) {
	previous := mailSender
	t.Cleanup(func() { mailSender = previous })
	mailSender = &fakeMailSender{err: errors.New("SMTP unavailable")}

	err := handleSendEmail(context.Background(), EmailPayload{Email: "user@example.test", Code: "123456"})
	if err == nil {
		t.Fatal("expected SMTP error")
	}
}
