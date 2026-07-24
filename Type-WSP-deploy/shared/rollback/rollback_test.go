package rollback

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestExecuteUsesDetachedContextInReverseOrder(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()

	manager := New()
	var order []string
	manager.Add("first", func(ctx context.Context) error {
		if err := ctx.Err(); err != nil {
			t.Fatalf("cleanup context inherited cancellation: %v", err)
		}
		order = append(order, "first")
		return nil
	})
	manager.Add("second", func(ctx context.Context) error {
		if err := ctx.Err(); err != nil {
			t.Fatalf("cleanup context inherited cancellation: %v", err)
		}
		order = append(order, "second")
		return nil
	})

	if err := manager.Execute(parent); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !reflect.DeepEqual(order, []string{"second", "first"}) {
		t.Fatalf("cleanup order = %#v", order)
	}
}

func TestExecuteReturnsCleanupErrors(t *testing.T) {
	expected := errors.New("cleanup failed")
	manager := New()
	manager.Add("remove object", func(context.Context) error {
		return expected
	})

	err := manager.Execute(context.Background())
	if !errors.Is(err, expected) {
		t.Fatalf("Execute error = %v, want wrapped cleanup error", err)
	}
}
