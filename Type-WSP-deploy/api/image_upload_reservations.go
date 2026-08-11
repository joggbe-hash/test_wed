package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const imageUploadReservationTTL = 15 * time.Minute

type imageUploadReservation struct {
	ID      uuid.UUID
	RawKeys []string
	Bytes   int64
}

func reserveImageUpload(ctx context.Context, userID, imageCount int, rawKeys []string) (imageUploadReservation, error) {
	reservation := imageUploadReservation{
		ID:      uuid.New(),
		RawKeys: append([]string(nil), rawKeys...),
		Bytes:   imageStorageReservation(imageCount),
	}
	rawKeysJSON, err := json.Marshal(reservation.RawKeys)
	if err != nil {
		return imageUploadReservation{}, fmt.Errorf("encode image upload reservation keys: %w", err)
	}

	now := time.Now().UTC()
	err = WithTx(ctx, systemPool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1::integer, $2::integer)", contentQuotaAdvisoryLockNamespace, userID); err != nil {
			return err
		}

		var total int
		var pending int
		var storedAndReserved int64
		if err := tx.QueryRow(ctx,
			`SELECT
			    (SELECT COUNT(*) FROM posts WHERE user_id = $1)
			      + (SELECT COUNT(*) FROM image_upload_reservations WHERE user_id = $1 AND expires_at > $2),
			    (SELECT COUNT(*) FROM posts WHERE user_id = $1 AND image_status = 'processing')
			      + (SELECT COUNT(*) FROM image_upload_reservations WHERE user_id = $1 AND expires_at > $2),
			    COALESCE((SELECT SUM(image_reserved_bytes + image_storage_bytes) FROM posts WHERE user_id = $1), 0)
			      + COALESCE((SELECT SUM(reserved_bytes) FROM image_upload_reservations WHERE user_id = $1 AND expires_at > $2), 0)`,
			userID, now,
		).Scan(&total, &pending, &storedAndReserved); err != nil {
			return err
		}
		if !hasPostCapacity(total) {
			return errPostQuotaExceeded
		}
		if !hasImagePostCapacity(pending) {
			return errImagePostQuotaExceeded
		}
		if !hasImageStorageCapacity(storedAndReserved, reservation.Bytes) {
			return errImageStorageQuotaExceeded
		}

		_, err := tx.Exec(ctx,
			`INSERT INTO image_upload_reservations (id, user_id, raw_keys, reserved_bytes, expires_at)
			 VALUES ($1, $2, $3, $4, $5)`,
			reservation.ID, userID, rawKeysJSON, reservation.Bytes, now.Add(imageUploadReservationTTL),
		)
		return err
	})
	if err != nil {
		return imageUploadReservation{}, err
	}
	return reservation, nil
}

func deleteImageUploadReservation(ctx context.Context, reservationID uuid.UUID, userID int) error {
	_, err := systemPool.Exec(ctx,
		"DELETE FROM image_upload_reservations WHERE id = $1 AND user_id = $2",
		reservationID, userID,
	)
	return err
}

func finalizeImageUpload(
	ctx context.Context,
	reservationID uuid.UUID,
	user *User,
	content string,
	visibility postVisibility,
) (int, error) {
	var postID int
	err := WithTx(ctx, systemPool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1::integer, $2::integer)", contentQuotaAdvisoryLockNamespace, user.ID); err != nil {
			return err
		}

		var rawKeysJSON []byte
		var reservedBytes int64
		if err := tx.QueryRow(ctx,
			`DELETE FROM image_upload_reservations
			 WHERE id = $1 AND user_id = $2 AND expires_at > NOW()
			 RETURNING raw_keys, reserved_bytes`,
			reservationID, user.ID,
		).Scan(&rawKeysJSON, &reservedBytes); err != nil {
			return err
		}

		return tx.QueryRow(ctx,
			`INSERT INTO posts (user_id, username, visibility, content, image_url, image_status, image_reserved_bytes)
			 VALUES ($1, $2, $3, $4, $5, 'processing', $6) RETURNING id`,
			user.ID, user.Username, visibility, content, string(rawKeysJSON), reservedBytes,
		).Scan(&postID)
	})
	return postID, err
}
