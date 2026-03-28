package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"book-dragon/internal/handlers"
	"book-dragon/internal/models"
	"book-dragon/internal/store"
)

func setupTestStore(t *testing.T) *store.Store {
	// :memory: creates a new in-memory SQLite database
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	return st
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(st *store.Store)
		payload        interface{}
		expectedStatus int
	}{
		{
			name: "Valid Registration",
			setup: func(st *store.Store) {},
			payload: models.RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Duplicate Email",
			setup: func(st *store.Store) {
				// Insert a user first to cause conflict
				u := &models.User{
					Username: "existinguser",
					Email:    "test@example.com",
					Password: "hashedpassword",
				}
				_ = st.CreateUser(u)
			},
			payload: models.RegisterRequest{
				Username: "anotheruser",
				Email:    "test@example.com",
				Password: "password123",
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "Missing Username",
			setup: func(st *store.Store) {},
			payload: models.RegisterRequest{
				Email:    "test2@example.com",
				Password: "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing Email",
			setup: func(st *store.Store) {},
			payload: models.RegisterRequest{
				Username: "testuser3",
				Password: "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing Password",
			setup: func(st *store.Store) {},
			payload: models.RegisterRequest{
				Username: "testuser4",
				Email:    "test4@example.com",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			setup:          func(st *store.Store) {},
			payload:        "not-json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := setupTestStore(t)
			tc.setup(st)
			handler := &handlers.UserHandler{Store: st}

			var bodyBytes []byte
			if str, ok := tc.payload.(string); ok {
				bodyBytes = []byte(str)
			} else {
				var err error
				bodyBytes, err = json.Marshal(tc.payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
			}

			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Register(w, req)

			if w.Result().StatusCode != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, w.Result().StatusCode)
			}
		})
	}
}
