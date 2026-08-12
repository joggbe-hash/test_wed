package main

import (
	"context"
	"errors"
	"testing"
)

func TestRedisImageProcessingBudgetBoundsAndRollback(t *testing.T) {
	useIntegrationRedis(t)
	ctx := context.Background()
	const userID = 42
	pixels := maxImageProcessingPixelsPerUserWindow

	if err := reserveImageProcessingBudget(ctx, userID, pixels); err != nil {
		t.Fatalf("reserve full processing budget: %v", err)
	}
	if err := reserveImageProcessingBudget(ctx, userID, 1); !errors.Is(err, errImageProcessingQuotaExceeded) {
		t.Fatalf("reserve beyond processing budget = %v, want quota exceeded", err)
	}
	if err := releaseImageProcessingBudget(ctx, userID, pixels); err != nil {
		t.Fatalf("release processing budget: %v", err)
	}
	if err := reserveImageProcessingBudget(ctx, userID, pixels); err != nil {
		t.Fatalf("reserve processing budget after rollback: %v", err)
	}
}
