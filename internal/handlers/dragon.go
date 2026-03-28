package handlers

import (
	"encoding/json"
	"net/http"

	"book-dragon/internal/auth"
	"book-dragon/internal/models"
	"book-dragon/internal/store"
)

type DragonHandler struct {
	Store *store.Store
}

// @Summary Create a new dragon
// @Description Create a dragon for the authenticated user
// @Tags dragons
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body models.CreateDragonRequest true "Dragon Creation Info"
// @Success 201 {object} models.Dragon
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /dragon [post]
func (h *DragonHandler) CreateDragon(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserContextKey).(int64)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req models.CreateDragonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if req.Name == "" || req.Color == "" {
		writeError(w, http.StatusBadRequest, "name and color are required")
		return
	}

	dragon := &models.Dragon{
		Name:   req.Name,
		Color:  req.Color,
		UserID: userID,
	}

	if err := h.Store.CreateDragon(dragon); err != nil {
		if err == store.ErrDragonAlreadyExists {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create dragon")
		return
	}

	writeJSON(w, http.StatusCreated, dragon)
}

// @Summary Get user's dragon
// @Description Get the authenticated user's dragon
// @Tags dragons
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.Dragon
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /dragon [get]
func (h *DragonHandler) GetDragon(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserContextKey).(int64)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	dragon, err := h.Store.GetDragonByUserID(userID)
	if err != nil {
		if err == store.ErrDragonNotFound {
			writeError(w, http.StatusNotFound, "dragon not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch dragon")
		return
	}

	writeJSON(w, http.StatusOK, dragon)
}
