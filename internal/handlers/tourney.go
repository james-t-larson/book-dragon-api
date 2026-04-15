package handlers

import (
	"encoding/json"
	"net/http"

	"book-dragon/internal/auth"
	"book-dragon/internal/models"
	"book-dragon/internal/store"
)

type TourneyHandler struct {
	Store *store.Store
}

// @Summary Get tourney configuration constants
// @Description Provides available configuration options for creating a new challenge
// @Tags tourney
// @Produce json
// @Success 200 {object} models.TourneyConstantsResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /constants [get]
func (h *TourneyHandler) GetConstants(w http.ResponseWriter, r *http.Request) {
	config, err := h.Store.GetTourneyConfig(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch tourney config")
		return
	}

	writeJSON(w, http.StatusOK, models.TourneyConstantsResponse{
		TourneyConfig: *config,
	})
}

// @Summary Get active tourney status
// @Description Fetches the exact UI state for the Tourney Hall
// @Tags tourney
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.TourneyStatusResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /tourney [get]
func (h *TourneyHandler) GetTourney(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserContextKey).(int64)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	status, err := h.Store.BuildTourneyStatus(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get tourney status")
		return
	}

	if status == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{})
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// @Summary Create a new tourney
// @Description Creates a new challenge and automatically enrolls the creator
// @Tags tourney
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body models.CreateChallengeRequest true "Challenge Info"
// @Success 201 {object} models.TourneyStatusResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /tourney [post]
func (h *TourneyHandler) CreateTourney(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserContextKey).(int64)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req models.CreateChallengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if req.OverallGoalDays <= 0 || req.DailyGoalMins <= 0 {
		writeError(w, http.StatusBadRequest, "overall_goal_days and daily_goal_minutes must be positive integers")
		return
	}

	_, _, err := h.Store.CreateChallenge(r.Context(), userID, req.Name, req.OverallGoalDays, req.DailyGoalMins)
	if err != nil {
		if err == store.ErrActiveChallenge {
			writeError(w, http.StatusConflict, "user already has an active challenge")
			return
		}
		if err == store.ErrInviteCodeCollision {
			writeError(w, http.StatusInternalServerError, "failed to generate unique invite code")
			return
		}
		// Check if validation error from store (invalid duration/minutes)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Build the full status response to return to frontend
	status, err := h.Store.BuildTourneyStatus(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build tourney status")
		return
	}

	writeJSON(w, http.StatusCreated, status)
}

// @Summary Join an existing tourney
// @Description Joins an existing challenge via invite code
// @Tags tourney
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body models.JoinChallengeRequest true "Join Info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /join_tourney [post]
func (h *TourneyHandler) JoinTourney(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserContextKey).(int64)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req models.JoinChallengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if req.InviteCode == "" {
		writeError(w, http.StatusBadRequest, "invite_code is required")
		return
	}

	err := h.Store.JoinChallenge(r.Context(), userID, req.InviteCode)
	if err != nil {
		switch err {
		case store.ErrActiveChallenge:
			writeError(w, http.StatusConflict, "user already has an active challenge")
		case store.ErrInviteCodeNotFound:
			writeError(w, http.StatusNotFound, "invite code not found")
		case store.ErrAlreadyEnrolled:
			writeError(w, http.StatusConflict, "user is already enrolled in this challenge")
		default:
			writeError(w, http.StatusInternalServerError, "failed to join challenge")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "successfully joined challenge"})
}
