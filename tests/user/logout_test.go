package usertest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"book-dragon/internal/auth"
	"book-dragon/internal/handlers"
)

func TestLogout(t *testing.T) {
	tests := []struct {
		name           string
		setContext     bool
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name:           "Valid Logout",
			setContext:     true,
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp map[string]string
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to parse logout response: %v", err)
				}
				if msg, ok := resp["message"]; !ok || msg != "successfully logged out" {
					t.Errorf("unexpected message: %s", msg)
				}
			},
		},
		{
			name:           "Missing Context (Unauthorized)",
			setContext:     false,
			expectedStatus: http.StatusUnauthorized,
			checkBody:      nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := setupTestStore(t)
			handler := &handlers.UserHandler{Store: st}

			req := httptest.NewRequest(http.MethodPost, "/logout", nil)

			if tc.setContext {
				ctx := context.WithValue(req.Context(), auth.UserContextKey, int64(1))
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()

			handler.Logout(w, req)

			if w.Result().StatusCode != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, w.Result().StatusCode)
			}

			if tc.checkBody != nil {
				tc.checkBody(t, w.Body.Bytes())
			}
		})
	}
}
