package usertest

import (
	"testing"
	"book-dragon/internal/store"
)

func setupTestStore(t *testing.T) *store.Store {
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	return st
}
