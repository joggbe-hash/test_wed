package main

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeEmail(t *testing.T) {
	got, err := normalizeEmail("  User.Name@Example.COM ")
	if err != nil || got != "user.name@example.com" {
		t.Fatalf("normalizeEmail = %q, %v", got, err)
	}

	for _, input := range []string{"", "not-an-email", "name <user@example.com>", strings.Repeat("a", 255)} {
		if _, err := normalizeEmail(input); err == nil {
			t.Errorf("normalizeEmail(%q) accepted invalid input", input)
		}
	}
}

func TestNormalizeUsername(t *testing.T) {
	if got, err := normalizeUsername("  測試者  "); err != nil || got != "測試者" {
		t.Fatalf("normalizeUsername = %q, %v", got, err)
	}

	for _, input := range []string{"a", "has space", "line\nbreak", strings.Repeat("名", 21)} {
		if _, err := normalizeUsername(input); err == nil {
			t.Errorf("normalizeUsername(%q) accepted invalid input", input)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	for _, input := range []string{"secure123", "密碼安全測試七8號"} {
		if err := validatePassword(input); err != nil {
			t.Fatalf("valid password %q rejected: %v", input, err)
		}
	}

	tests := []struct {
		input   string
		message string
	}{
		{"密碼a123", "密碼至少需要 8 個字元"},
		{strings.Repeat("密", 24) + "1", "密碼過長，請縮短後再試"},
		{"onlyletters", "密碼須包含至少一個字母與一個數字"},
		{"12345678", "密碼須包含至少一個字母與一個數字"},
	}
	for _, test := range tests {
		if err := validatePassword(test.input); err == nil || err.Error() != test.message {
			t.Errorf("validatePassword(%q) = %v, want %q", test.input, err, test.message)
		}
	}
}

func TestFeedCursorRoundTripPreservesTimestampAndID(t *testing.T) {
	wantTime := time.Date(2026, 7, 22, 9, 8, 7, 654321000, time.FixedZone("test", 8*60*60))
	wantID := 42
	got, err := decodeFeedCursor(encodeFeedCursor(wantTime, wantID))
	if err != nil {
		t.Fatalf("decodeFeedCursor: %v", err)
	}
	if !got.CreatedAt.Equal(wantTime) || got.ID != wantID {
		t.Fatalf("cursor = %#v, want time %s and id %d", got, wantTime, wantID)
	}
}

func TestDecodeFeedCursorRejectsMalformedValues(t *testing.T) {
	for _, raw := range []string{"", "not-base64", "MjAyNi0wNy0yMlQwMDowMDowMFo=", encodeFeedCursor(time.Now(), 0)} {
		if _, err := decodeFeedCursor(raw); err == nil {
			t.Errorf("decodeFeedCursor(%q) accepted malformed cursor", raw)
		}
	}
}

func TestPostLimits(t *testing.T) {
	if maxPostBodyBytes != 13<<20 {
		t.Fatalf("maxPostBodyBytes = %d, want 13 MiB", maxPostBodyBytes)
	}
	if validPostContent(strings.Repeat("文", maxPostContentRunes+1)) {
		t.Fatal("oversized post content was accepted")
	}
}
