package usertest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"book-dragon/internal/auth"
	"book-dragon/internal/handlers"
	"book-dragon/internal/models"
	"book-dragon/internal/store"
)

func TestMe(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(st *store.Store) *models.User
		setContext     bool
		expectedStatus int
		checkBody      func(t *testing.T, body []byte, u *models.User)
	}{
		{
			name: "Valid Token",
			setup: func(st *store.Store) *models.User {
				hashedPassword, _ := auth.HashPassword("password123")
				u := &models.User{
					Username: "testuser",
					Email:    "test@example.com",
					Password: hashedPassword,
				}
				_ = st.CreateUser(u)
				d := &models.Dragon{
					Name:   "Toothless",
					Color:  "Black",
					UserID: u.ID,
				}
				_ = st.CreateDragon(d)
				return u
			},
			setContext:     true,
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte, u *models.User) {
				var resp models.User
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to parse user response: %v", err)
				}
				if resp.Email != "test@example.com" {
					t.Errorf("expected email test@example.com, got %s", resp.Email)
				}
				if resp.ID != u.ID {
					t.Errorf("expected ID %d, got %d", u.ID, resp.ID)
				}
				if resp.DragonName == nil || *resp.DragonName != "Toothless" {
					if resp.DragonName == nil {
						t.Errorf("expected dragon Toothless, got nil")
					} else {
						t.Errorf("expected dragon Toothless, got %v", *resp.DragonName)
					}
				}
			},
		},
		{
			name: "User Not Found in DB",
			setup: func(st *store.Store) *models.User {
				return &models.User{ID: 999} // Non-existent user
			},
			setContext:     true,
			expectedStatus: http.StatusNotFound,
			checkBody:      nil,
		},
		{
			name: "Missing Context (Unauthorized)",
			setup: func(st *store.Store) *models.User {
				return nil
			},
			setContext:     false,
			expectedStatus: http.StatusUnauthorized,
			checkBody:      nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := setupTestStore(t)
			u := tc.setup(st)
			handler := &handlers.UserHandler{Store: st}

			req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)

			if tc.setContext && u != nil {
				ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()

			handler.Me(w, req)

			if w.Result().StatusCode != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, w.Result().StatusCode)
			}

			if tc.checkBody != nil {
				tc.checkBody(t, w.Body.Bytes(), u)
			}
		})
	}
}
