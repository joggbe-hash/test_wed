package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"typewsp/shared/contracts"
)

const (
	rawImageLifecycleRuleID           = "type-wsp-expire-raw-images"
	rawImageExpirationDays            = 1
	uploadReservationCleanupInterval  = time.Minute
	uploadReservationCleanupBatchSize = 100
)

const expiredImageUploadReservationsQuery = `
WITH expired AS (
    SELECT id
      FROM image_upload_reservations
     WHERE expires_at <= NOW()
     ORDER BY expires_at
     FOR UPDATE SKIP LOCKED
     LIMIT $1
)
DELETE FROM image_upload_reservations AS reservation
 USING expired
 WHERE reservation.id = expired.id
 RETURNING reservation.raw_keys`

func rawImageLifecycleConfiguration(existing *lifecycle.Configuration) *lifecycle.Configuration {
	configured := lifecycle.NewConfiguration()
	if existing != nil {
		configured.Rules = append(configured.Rules, existing.Rules...)
	}

	rawRule := lifecycle.Rule{
		ID:         rawImageLifecycleRuleID,
		Status:     "Enabled",
		RuleFilter: lifecycle.Filter{Prefix: contracts.RawImagePrefix},
		Expiration: lifecycle.Expiration{Days: lifecycle.ExpirationDays(rawImageExpirationDays)},
	}
	for index := range configured.Rules {
		if configured.Rules[index].ID == rawImageLifecycleRuleID {
			configured.Rules[index] = rawRule
			return configured
		}
	}
	configured.Rules = append(configured.Rules, rawRule)
	return configured
}

func ensureRawImageLifecycle(ctx context.Context) error {
	configuration, err := minioClient.GetBucketLifecycle(ctx, minioBucket)
	if err != nil {
		response := minio.ToErrorResponse(err)
		if response.Code != "NoSuchLifecycleConfiguration" {
			return fmt.Errorf("get MinIO bucket lifecycle: %w", err)
		}
		configuration = lifecycle.NewConfiguration()
	}

	if err := minioClient.SetBucketLifecycle(ctx, minioBucket, rawImageLifecycleConfiguration(configuration)); err != nil {
		return fmt.Errorf("set MinIO raw image lifecycle: %w", err)
	}
	return nil
}

func runUploadReservationJanitor(ctx context.Context) {
	cleanup := func() {
		if err := cleanupExpiredImageUploadReservations(ctx); err != nil {
			log.Printf("clean expired image upload reservations failed: %v", err)
		}
	}
	cleanup()

	ticker := time.NewTicker(uploadReservationCleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanup()
		}
	}
}

func cleanupExpiredImageUploadReservations(ctx context.Context) error {
	rows, err := systemPool.Query(ctx, expiredImageUploadReservationsQuery, uploadReservationCleanupBatchSize)
	if err != nil {
		return fmt.Errorf("claim expired image upload reservations: %w", err)
	}

	var reservations [][]string
	var decodeErrors []error
	for rows.Next() {
		var rawKeysJSON []byte
		if err := rows.Scan(&rawKeysJSON); err != nil {
			decodeErrors = append(decodeErrors, fmt.Errorf("scan expired image upload reservation: %w", err))
			continue
		}
		var rawKeys []string
		if err := json.Unmarshal(rawKeysJSON, &rawKeys); err != nil {
			decodeErrors = append(decodeErrors, fmt.Errorf("decode expired image upload reservation keys: %w", err))
			continue
		}
		reservations = append(reservations, rawKeys)
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		return errors.Join(errors.Join(decodeErrors...), fmt.Errorf("iterate expired image upload reservations: %w", rowsErr))
	}

	for _, rawKeys := range reservations {
		if err := deleteImages(ctx, ImageDeletePayload{Keys: rawKeys}); err != nil {
			decodeErrors = append(decodeErrors, fmt.Errorf("delete expired reservation images: %w", err))
		}
	}
	return errors.Join(decodeErrors...)
}
