package tourneytest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"book-dragon/internal/auth"
	"book-dragon/internal/handlers"
	"book-dragon/internal/models"
	"book-dragon/internal/store"
)

func setupTestStore(t *testing.T) *store.Store {
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	return st
}

func createTestUser(t *testing.T, st *store.Store, username, email string) *models.User {
	hashedPassword, _ := auth.HashPassword("password123")
	u := &models.User{Username: username, Email: email, Password: hashedPassword}
	if err := st.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return u
}

func createTestBook(t *testing.T, st *store.Store, userID int64) *models.Book {
	b, err := st.GetOrCreateBook(context.Background(), "Dune", "Frank Herbert", "Sci-Fi", 412)
	if err != nil {
		t.Fatalf("failed to create test book: %v", err)
	}
	_ = st.AddUserBook(context.Background(), userID, b.ID)
	return b
}

func todayUTC() string {
	return time.Now().UTC().Format("2006-01-02")
}

func intPtr(i int) *int { return &i }

// --- GET /constants Tests ---

func TestGetConstants(t *testing.T) {
	t.Run("TC-CONST-01: Returns tourney config", func(t *testing.T) {
		st := setupTestStore(t)
		handler := &handlers.TourneyHandler{Store: st}

		req := httptest.NewRequest(http.MethodGet, "/constants", nil)
		w := httptest.NewRecorder()
		handler.GetConstants(w, req)

		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", w.Result().StatusCode, w.Body.String())
		}

		var resp models.TourneyConstantsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if len(resp.TourneyConfig.OverallGoalDays) != 4 {
			t.Errorf("expected 4 overall_goal_days options, got %d", len(resp.TourneyConfig.OverallGoalDays))
		}
		if len(resp.TourneyConfig.DailyGoalMinutes) != 4 {
			t.Errorf("expected 4 daily_goal_minutes options, got %d", len(resp.TourneyConfig.DailyGoalMinutes))
		}

		expectedDays := []int{3, 7, 14, 30}
		for i, opt := range resp.TourneyConfig.OverallGoalDays {
			if opt.Value != expectedDays[i] {
				t.Errorf("expected overall_goal_days[%d].value = %d, got %d", i, expectedDays[i], opt.Value)
			}
		}

		expectedMins := []int{5, 10, 15, 30}
		for i, opt := range resp.TourneyConfig.DailyGoalMinutes {
			if opt.Value != expectedMins[i] {
				t.Errorf("expected daily_goal_minutes[%d].value = %d, got %d", i, expectedMins[i], opt.Value)
			}
		}
	})
}

// --- POST /tourney Tests ---

func TestCreateTourney(t *testing.T) {
	t.Run("TC-CREATE-01: Happy Path", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "creator", "creator@test.com")
		handler := &handlers.TourneyHandler{Store: st}

		payload := models.CreateChallengeRequest{
			Name:            "Summer Reading Challenge",
			OverallGoalDays: 7,
			DailyGoalMins:   15,
		}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/tourney", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.CreateTourney(w, req)

		if w.Result().StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d. Body: %s", w.Result().StatusCode, w.Body.String())
		}

		var resp models.TourneyStatusResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Name != "Summer Reading Challenge" {
			t.Errorf("expected name 'Summer Reading Challenge', got %q", resp.Name)
		}
		if resp.InviteCode == "" {
			t.Error("expected non-empty invite code")
		}
		if len(resp.InviteCode) < 6 || len(resp.InviteCode) > 9 {
			t.Errorf("expected invite code length 6-9 chars (with dash), got %d: %q", len(resp.InviteCode), resp.InviteCode)
		}
		if resp.DailyProgress.MinuteGoal != 15 {
			t.Errorf("expected minute_goal 15, got %d", resp.DailyProgress.MinuteGoal)
		}
		if resp.OverallProgress.DaysGoal != 7 {
			t.Errorf("expected days_goal 7, got %d", resp.OverallProgress.DaysGoal)
		}
		if resp.OverallProgress.DayNumber != 1 {
			t.Errorf("expected day_number 1, got %d", resp.OverallProgress.DayNumber)
		}
		if resp.DailyProgress.IsComplete {
			t.Error("expected daily progress to not be complete")
		}
		if resp.TauntMessages == nil || len(resp.TauntMessages) == 0 {
			t.Error("expected taunt messages when daily goal not met")
		}
	})

	t.Run("TC-CREATE-02: Conflict - Already Active", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "active_user", "active@test.com")
		handler := &handlers.TourneyHandler{Store: st}

		// Create first challenge
		payload := models.CreateChallengeRequest{Name: "First", OverallGoalDays: 3, DailyGoalMins: 5}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/tourney", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.CreateTourney(w, req)
		if w.Result().StatusCode != http.StatusCreated {
			t.Fatalf("setup: expected 201, got %d", w.Result().StatusCode)
		}

		// Try to create second
		payload2 := models.CreateChallengeRequest{Name: "Second", OverallGoalDays: 7, DailyGoalMins: 10}
		bodyBytes2, _ := json.Marshal(payload2)
		req2 := httptest.NewRequest(http.MethodPost, "/tourney", bytes.NewBuffer(bodyBytes2))
		req2.Header.Set("Content-Type", "application/json")
		ctx2 := context.WithValue(req2.Context(), auth.UserContextKey, u.ID)
		req2 = req2.WithContext(ctx2)
		w2 := httptest.NewRecorder()
		handler.CreateTourney(w2, req2)

		if w2.Result().StatusCode != http.StatusConflict {
			t.Fatalf("expected 409, got %d. Body: %s", w2.Result().StatusCode, w2.Body.String())
		}
	})

	t.Run("TC-CREATE-03: Validation - Missing Name", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "val_user1", "val1@test.com")
		handler := &handlers.TourneyHandler{Store: st}

		payload := models.CreateChallengeRequest{OverallGoalDays: 7, DailyGoalMins: 15}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/tourney", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.CreateTourney(w, req)

		if w.Result().StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Result().StatusCode)
		}
	})

	t.Run("TC-CREATE-03b: Validation - Invalid Duration", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "val_user2", "val2@test.com")
		handler := &handlers.TourneyHandler{Store: st}

		payload := models.CreateChallengeRequest{Name: "Bad Duration", OverallGoalDays: 5, DailyGoalMins: 15}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/tourney", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.CreateTourney(w, req)

		if w.Result().StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d. Body: %s", w.Result().StatusCode, w.Body.String())
		}
	})

	t.Run("TC-CREATE-03c: Validation - Invalid Daily Minutes", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "val_user3", "val3@test.com")
		handler := &handlers.TourneyHandler{Store: st}

		payload := models.CreateChallengeRequest{Name: "Bad Minutes", OverallGoalDays: 7, DailyGoalMins: 20}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/tourney", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.CreateTourney(w, req)

		if w.Result().StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d. Body: %s", w.Result().StatusCode, w.Body.String())
		}
	})

	t.Run("TC-CREATE: Unauthorized", func(t *testing.T) {
		st := setupTestStore(t)
		handler := &handlers.TourneyHandler{Store: st}

		payload := models.CreateChallengeRequest{Name: "Unauth", OverallGoalDays: 7, DailyGoalMins: 15}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/tourney", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.CreateTourney(w, req)

		if w.Result().StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Result().StatusCode)
		}
	})
}

// --- POST /join_tourney Tests ---

func TestJoinTourney(t *testing.T) {
	t.Run("TC-JOIN-01: Happy Path", func(t *testing.T) {
		st := setupTestStore(t)
		creator := createTestUser(t, st, "creator", "creator@test.com")
		joiner := createTestUser(t, st, "joiner", "joiner@test.com")
		handler := &handlers.TourneyHandler{Store: st}

		// Creator creates a tourney
		ch, _, err := st.CreateChallenge(context.Background(), creator.ID, "Join Test", 7, 15)
		if err != nil {
			t.Fatalf("failed to create challenge: %v", err)
		}

		// Joiner joins
		payload := models.JoinChallengeRequest{InviteCode: ch.InviteCode}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/join_tourney", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), auth.UserContextKey, joiner.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.JoinTourney(w, req)

		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", w.Result().StatusCode, w.Body.String())
		}

		// Verify enrollment
		uc, _, err := st.GetActiveUserChallenge(context.Background(), joiner.ID)
		if err != nil {
			t.Fatalf("failed to get active challenge: %v", err)
		}
		if uc == nil {
			t.Fatal("expected user to have an active challenge after joining")
		}
		if uc.ChallengeID != ch.ID {
			t.Errorf("expected challenge_id %d, got %d", ch.ID, uc.ChallengeID)
		}
	})

	t.Run("TC-JOIN-02: Invalid Code", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "joiner2", "joiner2@test.com")
		handler := &handlers.TourneyHandler{Store: st}

		payload := models.JoinChallengeRequest{InviteCode: "FAKE-CODE"}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/join_tourney", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.JoinTourney(w, req)

		if w.Result().StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404, got %d. Body: %s", w.Result().StatusCode, w.Body.String())
		}
	})

	t.Run("TC-JOIN-03: Already Active - Different Challenge", func(t *testing.T) {
		st := setupTestStore(t)
		creator1 := createTestUser(t, st, "c1", "c1@test.com")
		creator2 := createTestUser(t, st, "c2", "c2@test.com")
		joiner := createTestUser(t, st, "j3", "j3@test.com")
		handler := &handlers.TourneyHandler{Store: st}

		ch1, _, _ := st.CreateChallenge(context.Background(), creator1.ID, "Challenge 1", 7, 15)
		ch2, _, _ := st.CreateChallenge(context.Background(), creator2.ID, "Challenge 2", 3, 5)

		// Join first challenge
		_ = st.JoinChallenge(context.Background(), joiner.ID, ch1.InviteCode)

		// Try to join second while first is active
		payload := models.JoinChallengeRequest{InviteCode: ch2.InviteCode}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/join_tourney", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), auth.UserContextKey, joiner.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.JoinTourney(w, req)

		if w.Result().StatusCode != http.StatusConflict {
			t.Fatalf("expected 409, got %d. Body: %s", w.Result().StatusCode, w.Body.String())
		}
	})

	t.Run("TC-JOIN-04: Self-Join/Duplicate", func(t *testing.T) {
		st := setupTestStore(t)
		creator := createTestUser(t, st, "creator3", "creator3@test.com")
		handler := &handlers.TourneyHandler{Store: st}

		ch, _, _ := st.CreateChallenge(context.Background(), creator.ID, "Self Join Test", 7, 15)

		// Creator tries to join their own challenge (already enrolled as creator)
		payload := models.JoinChallengeRequest{InviteCode: ch.InviteCode}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/join_tourney", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), auth.UserContextKey, creator.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.JoinTourney(w, req)

		if w.Result().StatusCode != http.StatusConflict {
			t.Fatalf("expected 409, got %d. Body: %s", w.Result().StatusCode, w.Body.String())
		}
	})

	t.Run("TC-JOIN-05: Missing Invite Code", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "joiner5", "joiner5@test.com")
		handler := &handlers.TourneyHandler{Store: st}

		payload := models.JoinChallengeRequest{}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/join_tourney", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.JoinTourney(w, req)

		if w.Result().StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Result().StatusCode)
		}
	})
}

// --- GET /tourney Tests ---

func TestGetTourney(t *testing.T) {
	t.Run("TC-STATE: No Active Challenge returns empty object", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "noactive", "noactive@test.com")
		handler := &handlers.TourneyHandler{Store: st}

		req := httptest.NewRequest(http.MethodGet, "/tourney", nil)
		ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.GetTourney(w, req)

		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Result().StatusCode)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if len(resp) != 0 {
			t.Errorf("expected empty object, got %v", resp)
		}
	})

	t.Run("TC-STATE-01: Incomplete Day shows taunt messages", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "incomplete", "incomplete@test.com")
		handler := &handlers.TourneyHandler{Store: st}

		_, _, err := st.CreateChallenge(context.Background(), u.ID, "Taunt Test", 7, 15)
		if err != nil {
			t.Fatalf("failed to create challenge: %v", err)
		}

		// Log some but not enough minutes
		_ = st.UpsertDailyReadingLog(context.Background(), u.ID, todayUTC(), 5)

		req := httptest.NewRequest(http.MethodGet, "/tourney", nil)
		ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.GetTourney(w, req)

		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Result().StatusCode)
		}

		var resp models.TourneyStatusResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.DailyProgress.IsComplete {
			t.Error("expected daily progress to NOT be complete")
		}
		if resp.DailyProgress.MinutesComplete != 5 {
			t.Errorf("expected 5 minutes, got %d", resp.DailyProgress.MinutesComplete)
		}
		if resp.TauntMessages == nil || len(resp.TauntMessages) == 0 {
			t.Error("expected taunt messages when daily goal not met")
		}
	})

	t.Run("TC-STATE-02: Exact Completion", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "exact", "exact@test.com")
		handler := &handlers.TourneyHandler{Store: st}

		_, _, err := st.CreateChallenge(context.Background(), u.ID, "Exact Test", 7, 15)
		if err != nil {
			t.Fatalf("failed to create challenge: %v", err)
		}

		_ = st.UpsertDailyReadingLog(context.Background(), u.ID, todayUTC(), 15)

		req := httptest.NewRequest(http.MethodGet, "/tourney", nil)
		ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.GetTourney(w, req)

		var resp models.TourneyStatusResponse
		json.Unmarshal(w.Body.Bytes(), &resp)

		if !resp.DailyProgress.IsComplete {
			t.Error("expected daily progress to be complete at exact goal")
		}
		if resp.TauntMessages != nil && len(resp.TauntMessages) > 0 {
			t.Error("expected no taunt messages when daily goal met")
		}
	})

	t.Run("TC-STATE-03: Overachiever", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "over", "over@test.com")
		handler := &handlers.TourneyHandler{Store: st}

		_, _, err := st.CreateChallenge(context.Background(), u.ID, "Over Test", 7, 15)
		if err != nil {
			t.Fatalf("failed to create challenge: %v", err)
		}

		_ = st.UpsertDailyReadingLog(context.Background(), u.ID, todayUTC(), 30)

		req := httptest.NewRequest(http.MethodGet, "/tourney", nil)
		ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.GetTourney(w, req)

		var resp models.TourneyStatusResponse
		json.Unmarshal(w.Body.Bytes(), &resp)

		if !resp.DailyProgress.IsComplete {
			t.Error("expected daily progress to be complete when exceeding goal")
		}
		if resp.DailyProgress.MinutesComplete != 30 {
			t.Errorf("expected 30 minutes, got %d", resp.DailyProgress.MinutesComplete)
		}
	})

	t.Run("TC-STATE: Unauthorized", func(t *testing.T) {
		st := setupTestStore(t)
		handler := &handlers.TourneyHandler{Store: st}

		req := httptest.NewRequest(http.MethodGet, "/tourney", nil)
		w := httptest.NewRecorder()
		handler.GetTourney(w, req)

		if w.Result().StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Result().StatusCode)
		}
	})
}

// --- Lazy Evaluation & Expiration Tests ---

func TestLazyEvaluation(t *testing.T) {
	t.Run("TC-STATE-05: Expiration via BuildTourneyStatus", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "expiry", "expiry@test.com")

		// Create challenge
		_, _, err := st.CreateChallenge(context.Background(), u.ID, "Expired Test", 3, 5)
		if err != nil {
			t.Fatalf("failed to create challenge: %v", err)
		}

		// Manually update start_date to 5 days ago to simulate expiration
		ctx := context.Background()
		st.SetChallengeStartDate(ctx, u.ID, 5)

		// Now call BuildTourneyStatus, which should lazily expire the challenge
		status, err := st.BuildTourneyStatus(ctx, u.ID)
		if err != nil {
			t.Fatalf("BuildTourneyStatus error: %v", err)
		}
		if status != nil {
			t.Errorf("expected nil status after expiration, got %+v", status)
		}
	})
}

// --- Auth Integration Tests ---

func TestAuthTourneyIntegration(t *testing.T) {
	t.Run("TC-AUTH-01: Register returns nil tourney", func(t *testing.T) {
		st := setupTestStore(t)
		handler := &handlers.UserHandler{Store: st}

		payload := models.RegisterRequest{
			Username: "newuser",
			Email:    "newuser@test.com",
			Password: "password123",
		}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Register(w, req)

		if w.Result().StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d. Body: %s", w.Result().StatusCode, w.Body.String())
		}

		var resp models.AuthResponse
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp.User.TourneyStatus != nil {
			t.Errorf("expected nil tourney for new user, got %+v", resp.User.TourneyStatus)
		}
	})

	t.Run("TC-AUTH-02: Login with no tourney returns nil tourney", func(t *testing.T) {
		st := setupTestStore(t)
		_ = createTestUser(t, st, "loginuser", "login@test.com")
		handler := &handlers.UserHandler{Store: st}

		payload := models.LoginRequest{Email: "login@test.com", Password: "password123"}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Login(w, req)

		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", w.Result().StatusCode, w.Body.String())
		}

		var resp models.AuthResponse
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp.User.TourneyStatus != nil {
			t.Errorf("expected nil tourney, got %+v", resp.User.TourneyStatus)
		}
	})

	t.Run("TC-AUTH-03: Login with active tourney returns populated tourney", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "activelogin", "activelogin@test.com")
		handler := &handlers.UserHandler{Store: st}

		// Create a challenge for the user
		_, _, err := st.CreateChallenge(context.Background(), u.ID, "Active Login Test", 7, 15)
		if err != nil {
			t.Fatalf("failed to create challenge: %v", err)
		}

		payload := models.LoginRequest{Email: "activelogin@test.com", Password: "password123"}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Login(w, req)

		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", w.Result().StatusCode, w.Body.String())
		}

		var resp models.AuthResponse
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp.User.TourneyStatus == nil {
			t.Fatal("expected non-nil tourney status for user with active challenge")
		}
		if resp.User.TourneyStatus.Name != "Active Login Test" {
			t.Errorf("expected tourney name 'Active Login Test', got %q", resp.User.TourneyStatus.Name)
		}
		if resp.User.TourneyStatus.OverallProgress.DaysGoal != 7 {
			t.Errorf("expected days_goal 7, got %d", resp.User.TourneyStatus.OverallProgress.DaysGoal)
		}
	})

	t.Run("TC-AUTH-04: Login with expired tourney returns nil tourney", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "expiredlogin", "expiredlogin@test.com")
		handler := &handlers.UserHandler{Store: st}

		// Create then expire a challenge
		_, _, err := st.CreateChallenge(context.Background(), u.ID, "Expired Login Test", 3, 5)
		if err != nil {
			t.Fatalf("failed to create challenge: %v", err)
		}
		st.SetChallengeStartDate(context.Background(), u.ID, 5)

		payload := models.LoginRequest{Email: "expiredlogin@test.com", Password: "password123"}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.Login(w, req)

		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", w.Result().StatusCode, w.Body.String())
		}

		var resp models.AuthResponse
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp.User.TourneyStatus != nil {
			t.Errorf("expected nil tourney for expired challenge, got %+v", resp.User.TourneyStatus)
		}
	})

	t.Run("TC-AUTH: Me returns tourney status", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "meuser", "meuser@test.com")
		handler := &handlers.UserHandler{Store: st}

		// Create a challenge
		_, _, err := st.CreateChallenge(context.Background(), u.ID, "Me Test", 14, 10)
		if err != nil {
			t.Fatalf("failed to create challenge: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
		ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.Me(w, req)

		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Result().StatusCode)
		}

		var resp models.User
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp.TourneyStatus == nil {
			t.Fatal("expected tourney status in /auth/me response")
		}
		if resp.TourneyStatus.Name != "Me Test" {
			t.Errorf("expected name 'Me Test', got %q", resp.TourneyStatus.Name)
		}
	})
}

// --- Focus Timer Daily Log Tests ---

func TestFocusTimerDailyLog(t *testing.T) {
	t.Run("TC-FT-01: Initial Log", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "ftlog1", "ftlog1@test.com")
		b := createTestBook(t, st, u.ID)
		handler := &handlers.UserHandler{Store: st}

		payload := models.FocusTimerRequest{Minutes: 15, BookID: b.ID, CurrentPage: intPtr(50)}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/focus_timer_complete", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.FocusTimerComplete(w, req)

		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", w.Result().StatusCode, w.Body.String())
		}

		// Verify daily reading log
		minutes, err := st.GetDailyReadingLog(context.Background(), u.ID, todayUTC())
		if err != nil {
			t.Fatalf("failed to get daily log: %v", err)
		}
		if minutes != 15 {
			t.Errorf("expected 15 minutes in daily log, got %d", minutes)
		}
	})

	t.Run("TC-FT-02: Accumulation", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "ftlog2", "ftlog2@test.com")
		b := createTestBook(t, st, u.ID)
		handler := &handlers.UserHandler{Store: st}

		// First session: 15 minutes
		payload1 := models.FocusTimerRequest{Minutes: 15, BookID: b.ID, CurrentPage: intPtr(50)}
		bodyBytes1, _ := json.Marshal(payload1)
		req1 := httptest.NewRequest(http.MethodPost, "/focus_timer_complete", bytes.NewBuffer(bodyBytes1))
		req1.Header.Set("Content-Type", "application/json")
		ctx1 := context.WithValue(req1.Context(), auth.UserContextKey, u.ID)
		req1 = req1.WithContext(ctx1)
		w1 := httptest.NewRecorder()
		handler.FocusTimerComplete(w1, req1)

		// Second session: 10 minutes
		payload2 := models.FocusTimerRequest{Minutes: 10, BookID: b.ID, CurrentPage: intPtr(80)}
		bodyBytes2, _ := json.Marshal(payload2)
		req2 := httptest.NewRequest(http.MethodPost, "/focus_timer_complete", bytes.NewBuffer(bodyBytes2))
		req2.Header.Set("Content-Type", "application/json")
		ctx2 := context.WithValue(req2.Context(), auth.UserContextKey, u.ID)
		req2 = req2.WithContext(ctx2)
		w2 := httptest.NewRecorder()
		handler.FocusTimerComplete(w2, req2)

		if w2.Result().StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", w2.Result().StatusCode)
		}

		// Verify accumulation
		minutes, _ := st.GetDailyReadingLog(context.Background(), u.ID, todayUTC())
		if minutes != 25 {
			t.Errorf("expected 25 minutes accumulated, got %d", minutes)
		}
	})

	t.Run("TC-FT-05: Max Limit Validation", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "ftmax", "ftmax@test.com")
		b := createTestBook(t, st, u.ID)
		handler := &handlers.UserHandler{Store: st}

		payload := models.FocusTimerRequest{Minutes: 1441, BookID: b.ID, CurrentPage: intPtr(10)}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/focus_timer_complete", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.FocusTimerComplete(w, req)

		if w.Result().StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d. Body: %s", w.Result().StatusCode, w.Body.String())
		}
	})
}

// --- Days Complete counting ---

func TestDaysCompleteCount(t *testing.T) {
	t.Run("Days complete counts correctly", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "daycount", "daycount@test.com")
		handler := &handlers.TourneyHandler{Store: st}

		// Create a 7-day challenge with 10 min daily goal
		_, _, err := st.CreateChallenge(context.Background(), u.ID, "Days Count", 7, 10)
		if err != nil {
			t.Fatalf("failed to create challenge: %v", err)
		}

		// Log today with enough minutes
		_ = st.UpsertDailyReadingLog(context.Background(), u.ID, todayUTC(), 15)

		req := httptest.NewRequest(http.MethodGet, "/tourney", nil)
		ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.GetTourney(w, req)

		var resp models.TourneyStatusResponse
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp.OverallProgress.DaysComplete != 1 {
			t.Errorf("expected 1 day complete, got %d", resp.OverallProgress.DaysComplete)
		}
	})
}
