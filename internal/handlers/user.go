package handlers

import (
	"encoding/json"
	"net/http"

	"book-dragon/internal/auth"
	"book-dragon/internal/models"
	"book-dragon/internal/store"
)

type UserHandler struct {
	Store *store.Store
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// @Summary Register a new user
// @Description Create a new user account and log the user in
// @Tags users
// @Accept json
// @Produce json
// @Param body body models.RegisterRequest true "User Registration Info"
// @Success 201 {object} models.AuthResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /register [post]
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username, email, and password are required")
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
	}

	if err := h.Store.CreateUser(r.Context(), user); err != nil {
		if err == store.ErrDuplicateEmail {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusCreated, models.AuthResponse{
		Token: token,
		User:  *user,
	})
}

// @Summary User login
// @Description Authenticate a user and return a JWT token
// @Tags users
// @Accept json
// @Produce json
// @Param body body models.LoginRequest true "User Login Info"
// @Success 200 {object} models.AuthResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /login [post]
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	user, err := h.Store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if err == store.ErrUserNotFound {
			writeError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}

	if !auth.CheckPasswordHash(req.Password, user.Password) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, models.AuthResponse{
		Token: token,
		User:  *user,
	})
}

// @Summary Get current user
// @Description Get the currently authenticated user's profile
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.User
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /auth/me [get]
func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserContextKey).(int64)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.Store.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// @Summary User logout
// @Description Logout a user (client-side token deletion required)
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string
// @Failure 401 {object} models.ErrorResponse
// @Router /logout [post]
func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value(auth.UserContextKey).(int64)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "successfully logged out"})
}

// @Summary Complete a focus timer session
// @Description Add coins based on minutes read
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body models.FocusTimerRequest true "Focus Timer Info"
// @Success 200 {object} models.FocusTimerResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /focus_timer_complete [post]
func (h *UserHandler) FocusTimerComplete(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserContextKey).(int64)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req models.FocusTimerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if req.Minutes <= 0 || req.BookID <= 0 {
		writeError(w, http.StatusBadRequest, "minutes and book_id are required and must be strictly positive")
		return
	}

	if req.PagesRead == nil {
		writeError(w, http.StatusBadRequest, "pages_read is required")
		return
	}

	if *req.PagesRead < 0 {
		writeError(w, http.StatusBadRequest, "pages_read must be zero or positive")
		return
	}

	hasBook, err := h.Store.HasUserBook(r.Context(), userID, req.BookID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify book ownership")
		return
	}
	if !hasBook {
		writeError(w, http.StatusNotFound, "book not found in user's library")
		return
	}

	coinsEarned := int64((req.Minutes / 5) * 3)

	totalCoins, err := h.Store.AddCoinsToUser(r.Context(), userID, coinsEarned)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update coins")
		return
	}

	if *req.PagesRead > 0 {
		if err := h.Store.AddPagesRead(r.Context(), userID, req.BookID, *req.PagesRead); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update reading progress")
			return
		}
	}

	writeJSON(w, http.StatusOK, models.FocusTimerResponse{
		CoinsEarned: coinsEarned,
		TotalCoins:  totalCoins,
	})
}

