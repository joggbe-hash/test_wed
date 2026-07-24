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
	if err := validatePassword("secure123"); err != nil {
		t.Fatalf("valid password rejected: %v", err)
	}

	for _, input := range []string{"a1", "onlyletters", "12345678", strings.Repeat("a", 72) + "1"} {
		if err := validatePassword(input); err == nil {
			t.Errorf("validatePassword(%q) accepted weak input", input)
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
	if maxPostBodyBytes != 25<<20 {
		t.Fatalf("maxPostBodyBytes = %d, want 25 MiB", maxPostBodyBytes)
	}
	if validPostContent(strings.Repeat("文", maxPostContentRunes+1)) {
		t.Fatal("oversized post content was accepted")
	}
}
