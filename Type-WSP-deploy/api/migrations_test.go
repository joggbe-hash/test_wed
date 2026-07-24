package main

import "testing"

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
