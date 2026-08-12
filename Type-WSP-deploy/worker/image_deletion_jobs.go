package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	imageDeletionLease           = 10 * time.Minute
	imageDeletionCleanupInterval = time.Minute
	imageDeletionCleanupBatch    = 100
)

type imageDeletionClaimer func(context.Context, string, string) ([]string, bool, error)
type imageObjectRemover func(context.Context, ImageDeletePayload) error
type imageDeletionStateChange func(context.Context, string, string) error

func processImageDeletionJob(ctx context.Context, jobID string) error {
	if _, err := uuid.Parse(jobID); err != nil {
		return fmt.Errorf("invalid image deletion job id: %w", err)
	}
	return executeImageDeletionJob(
		ctx,
		jobID,
		uuid.NewString(),
		claimImageDeletionJob,
		deleteImages,
		completeImageDeletionJob,
		releaseImageDeletionJob,
	)
}

func executeImageDeletionJob(
	ctx context.Context,
	jobID string,
	token string,
	claim imageDeletionClaimer,
	remove imageObjectRemover,
	complete imageDeletionStateChange,
	release imageDeletionStateChange,
) error {
	keys, claimed, err := claim(ctx, jobID, token)
	if err != nil || !claimed {
		return err
	}
	if err := remove(ctx, ImageDeletePayload{Keys: keys}); err != nil {
		releaseErr := release(ctx, jobID, token)
		return errors.Join(err, releaseErr)
	}
	if err := complete(ctx, jobID, token); err != nil {
		releaseErr := release(ctx, jobID, token)
		return errors.Join(err, releaseErr)
	}
	return nil
}

func claimImageDeletionJob(ctx context.Context, jobID, token string) ([]string, bool, error) {
	expiredBefore := time.Now().UTC().Add(-imageDeletionLease)
	var encodedKeys []byte
	err := systemPool.QueryRow(ctx,
		`UPDATE image_deletion_jobs
		 SET processing_token = $1,
		     processing_started_at = NOW()
		 WHERE id = $2
		   AND (
		     processing_token IS NULL
		     OR processing_started_at IS NULL
		     OR processing_started_at < $3
		   )
		 RETURNING object_keys`,
		token, jobID, expiredBefore,
	).Scan(&encodedKeys)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("claim image deletion job: %w", err)
	}
	var keys []string
	if err := json.Unmarshal(encodedKeys, &keys); err != nil {
		return nil, false, fmt.Errorf("decode image deletion keys: %w", err)
	}
	return keys, true, nil
}

func completeImageDeletionJob(ctx context.Context, jobID, token string) error {
	tag, err := systemPool.Exec(ctx,
		"DELETE FROM image_deletion_jobs WHERE id = $1 AND processing_token = $2",
		jobID, token,
	)
	if err != nil {
		return fmt.Errorf("complete image deletion job: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("image deletion job lease was lost")
	}
	return nil
}

func releaseImageDeletionJob(ctx context.Context, jobID, token string) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	_, err := systemPool.Exec(cleanupCtx,
		`UPDATE image_deletion_jobs
		 SET processing_token = NULL,
		     processing_started_at = NULL
		 WHERE id = $1 AND processing_token = $2`,
		jobID, token,
	)
	if err != nil {
		return fmt.Errorf("release image deletion job: %w", err)
	}
	return nil
}

func runImageDeletionJanitor(ctx context.Context) {
	cleanup := func() {
		if err := processPendingImageDeletionJobs(ctx); err != nil {
			log.Printf("process pending image deletion jobs failed: %v", err)
		}
	}
	cleanup()
	ticker := time.NewTicker(imageDeletionCleanupInterval)
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

func processPendingImageDeletionJobs(ctx context.Context) error {
	rows, err := systemPool.Query(ctx,
		`SELECT id
		 FROM image_deletion_jobs
		 WHERE processing_token IS NULL
		    OR processing_started_at IS NULL
		    OR processing_started_at < $1
		 ORDER BY created_at
		 LIMIT $2`,
		time.Now().UTC().Add(-imageDeletionLease), imageDeletionCleanupBatch,
	)
	if err != nil {
		return fmt.Errorf("list pending image deletion jobs: %w", err)
	}
	defer rows.Close()
	var jobIDs []string
	for rows.Next() {
		var jobID uuid.UUID
		if err := rows.Scan(&jobID); err != nil {
			return fmt.Errorf("scan image deletion job: %w", err)
		}
		jobIDs = append(jobIDs, jobID.String())
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate image deletion jobs: %w", err)
	}
	var cleanupErrors []error
	for _, jobID := range jobIDs {
		if err := processImageDeletionJob(ctx, jobID); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("process image deletion job %s: %w", jobID, err))
		}
	}
	return errors.Join(cleanupErrors...)
}
