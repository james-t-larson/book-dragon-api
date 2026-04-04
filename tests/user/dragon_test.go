package usertest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"book-dragon/internal/auth"
	"book-dragon/internal/handlers"
	"book-dragon/internal/models"
)

func TestCreateDragon(t *testing.T) {
	st := setupTestStore(t)
	handler := &handlers.DragonHandler{Store: st}

	u := &models.User{
		Username: "dragontester",
		Email:    "dragon@example.com",
		Password: "password123",
	}
	_ = st.CreateUser(context.Background(), u)

	tests := []struct {
		name           string
		userID         int64
		payload        interface{}
		expectedStatus int
	}{
		{
			name:   "Valid Dragon",
			userID: u.ID,
			payload: models.CreateDragonRequest{
				Name:  "Smaug",
				Color: "Red",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:   "Duplicate Dragon",
			userID: u.ID,
			payload: models.CreateDragonRequest{
				Name:  "Balerion",
				Color: "Black",
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:   "Unauthorized",
			userID: 0,
			payload: models.CreateDragonRequest{
				Name:  "Drogon",
				Color: "Black",
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tc.payload)
			req := httptest.NewRequest(http.MethodPost, "/dragon", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			if tc.userID != 0 {
				ctx := context.WithValue(req.Context(), auth.UserContextKey, tc.userID)
				req = req.WithContext(ctx)
			}
			w := httptest.NewRecorder()

			handler.CreateDragon(w, req)

			if w.Result().StatusCode != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, w.Result().StatusCode)
			}
		})
	}
}

func TestGetDragon(t *testing.T) {
	st := setupTestStore(t)
	handler := &handlers.DragonHandler{Store: st}

	u := &models.User{
		Username: "getter",
		Email:    "getter@example.com",
		Password: "password123",
	}
	_ = st.CreateUser(context.Background(), u)

	u2 := &models.User{
		Username: "nodragon",
		Email:    "nodragon@example.com",
		Password: "password123",
	}
	_ = st.CreateUser(context.Background(), u2)

	dragon := &models.Dragon{
		Name:   "Puff",
		Color:  "Green",
		UserID: u.ID,
	}
	_ = st.CreateDragon(context.Background(), dragon)

	tests := []struct {
		name           string
		userID         int64
		expectedStatus int
	}{
		{
			name:           "Has Dragon",
			userID:         u.ID,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "No Dragon",
			userID:         u2.ID,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Unauthorized",
			userID:         0,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/dragon", nil)
			if tc.userID != 0 {
				ctx := context.WithValue(req.Context(), auth.UserContextKey, tc.userID)
				req = req.WithContext(ctx)
			}
			w := httptest.NewRecorder()

			handler.GetDragon(w, req)

			if w.Result().StatusCode != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, w.Result().StatusCode)
			}
		})
	}
}
