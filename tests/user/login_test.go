package usertest

import (
"bytes"
"encoding/json"
"net/http"
"net/http/httptest"
"testing"

"book-dragon/internal/auth"
"book-dragon/internal/handlers"
"book-dragon/internal/models"
"book-dragon/internal/store"
)

func TestLogin(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(st *store.Store)
		payload        interface{}
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name: "Valid Login",
			setup: func(st *store.Store) {
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
			},
			payload: models.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp models.AuthResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to parse auth response: %v", err)
				}
				if resp.Token == "" {
					t.Error("expected token to be present")
				}
				if resp.User.Email != "test@example.com" {
					t.Errorf("expected email test@example.com, got %s", resp.User.Email)
				}
				if resp.User.DragonName == nil || *resp.User.DragonName != "Toothless" {
					if resp.User.DragonName == nil {
						t.Errorf("expected dragon Toothless, got nil")
					} else {
						t.Errorf("expected dragon Toothless, got %v", *resp.User.DragonName)
					}
				}
			},
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

			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Login(w, req)

			if w.Result().StatusCode != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, w.Result().StatusCode)
			}

			if tc.checkBody != nil {
				tc.checkBody(t, w.Body.Bytes())
			}
		})
	}
}
