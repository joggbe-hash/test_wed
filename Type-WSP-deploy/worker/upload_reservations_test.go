package main

import (
	"strings"
	"testing"

	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"typewsp/shared/contracts"
)

func TestRawImageLifecycleConfigurationPreservesRulesAndIsIdempotent(t *testing.T) {
	config := &lifecycle.Configuration{Rules: []lifecycle.Rule{{
		ID:         "keep-existing-rule",
		Status:     "Enabled",
		Prefix:     "archive/",
		Expiration: lifecycle.Expiration{Days: 30},
	}}}

	configured := rawImageLifecycleConfiguration(config)
	configured = rawImageLifecycleConfiguration(configured)

	if len(configured.Rules) != 2 {
		t.Fatalf("lifecycle rules = %d, want 2", len(configured.Rules))
	}
	if configured.Rules[0].ID != "keep-existing-rule" {
		t.Fatal("existing lifecycle rule was not preserved")
	}
	rawRule := configured.Rules[1]
	if rawRule.ID != rawImageLifecycleRuleID || rawRule.Status != "Enabled" {
		t.Fatalf("raw lifecycle rule identity/status = %q/%q", rawRule.ID, rawRule.Status)
	}
	if rawRule.RuleFilter.Prefix != contracts.RawImagePrefix {
		t.Fatalf("raw lifecycle prefix = %q, want %q", rawRule.RuleFilter.Prefix, contracts.RawImagePrefix)
	}
	if rawRule.Expiration.Days != lifecycle.ExpirationDays(rawImageExpirationDays) {
		t.Fatalf("raw lifecycle expiration = %d days", rawRule.Expiration.Days)
	}
}

func TestExpiredReservationClaimDeletesRowsBeforeObjectCleanup(t *testing.T) {
	for _, expected := range []string{"FOR UPDATE SKIP LOCKED", "DELETE FROM image_upload_reservations", "RETURNING reservation.raw_keys"} {
		if !strings.Contains(expiredImageUploadReservationsQuery, expected) {
			t.Fatalf("expired reservation claim query does not contain %q", expected)
		}
	}
}
