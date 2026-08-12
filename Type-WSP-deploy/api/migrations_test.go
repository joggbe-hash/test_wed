package main

import (
	"strings"
	"testing"
)

func TestEmbeddedMigrationSetsAreVersioned(t *testing.T) {
	for _, directory := range []string{"migrations/user", "migrations/system"} {
		items, err := loadMigrations(directory)
		if err != nil {
			t.Fatalf("loadMigrations(%q): %v", directory, err)
		}
		for index := 1; index < len(items); index++ {
			if items[index-1].version >= items[index].version {
				t.Fatalf("migrations in %s are not strictly ordered", directory)
			}
		}
	}
}

func TestImageStorageQuotaMigrationAddsPersistentAccounting(t *testing.T) {
	items, err := loadMigrations("migrations/system")
	if err != nil {
		t.Fatalf("load system migrations: %v", err)
	}
	var allSQL strings.Builder
	for _, item := range items {
		allSQL.WriteString(item.sql)
	}
	for _, expected := range []string{"image_reserved_bytes", "image_storage_bytes"} {
		if !strings.Contains(allSQL.String(), expected) {
			t.Fatalf("system migrations do not add %s", expected)
		}
	}
}

func TestImageUploadReservationMigrationIsPresent(t *testing.T) {
	items, err := loadMigrations("migrations/system")
	if err != nil {
		t.Fatalf("load system migrations: %v", err)
	}
	var migrationSQL string
	for _, item := range items {
		if strings.Contains(item.sql, "image_upload_reservations") {
			migrationSQL = item.sql
			break
		}
	}
	for _, expected := range []string{"image_upload_reservations", "raw_keys", "reserved_bytes", "expires_at"} {
		if !strings.Contains(migrationSQL, expected) {
			t.Fatalf("image upload reservation migration does not add %s", expected)
		}
	}
}

func TestLatestMigrationAddsDurableImageDeletionOutbox(t *testing.T) {
	items, err := loadMigrations("migrations/system")
	if err != nil {
		t.Fatalf("load system migrations: %v", err)
	}
	latest := items[len(items)-1]
	for _, expected := range []string{"image_deletion_jobs", "object_keys", "reserved_bytes", "processing_token", "processing_started_at"} {
		if !strings.Contains(latest.sql, expected) {
			t.Fatalf("latest migration %s does not add %s", latest.name, expected)
		}
	}
}
