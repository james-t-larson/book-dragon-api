package tourneytest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

func setCoins(t *testing.T, st *store.Store, userID int64, coins int64) {
	_, err := st.AddCoinsToUser(context.Background(), userID, coins)
	if err != nil {
		t.Fatalf("failed to set coins: %v", err)
	}
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

// --- POST /tourney Tests (TC-1.x) ---

func TestCreateTourney(t *testing.T) {
	t.Run("TC-1.1 Successful creation", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "creator", "creator@test.com")
		setCoins(t, st, u.ID, 100)
		handler := &handlers.TourneyHandler{Store: st}

		payload := models.CreateChallengeRequest{
			Name:            "Summer Reading Challenge",
			OverallGoalDays: 7,
			DailyGoalMins:   15,
			Ante:            intPtr(50),
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
		json.Unmarshal(w.Body.Bytes(), &resp)

		// Check starttime: should be tomorrow 00:01
		expectedStart := time.Now().UTC().AddDate(0, 0, 1).Truncate(24 * time.Hour).Add(1 * time.Minute)
		gotStart, _ := time.Parse(time.RFC3339, resp.StartTime)
		if !gotStart.Truncate(time.Minute).Equal(expectedStart.Truncate(time.Minute)) {
			t.Errorf("expected starttime %v, got %v", expectedStart, gotStart)
		}

		updatedUser, _ := st.GetUserByID(context.Background(), u.ID)
		if updatedUser.Coins != 50 {
			t.Errorf("expected 50 coins remaining, got %d", updatedUser.Coins)
		}
	})

	t.Run("TC-1.2 Missing required ante param", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "val_user1", "val1@test.com")
		handler := &handlers.TourneyHandler{Store: st}

		// Payload missing "ante"
		payload := map[string]interface{}{
			"name":               "No Ante",
			"overall_goal_days":  7,
			"daily_goal_minutes": 15,
		}
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

	t.Run("TC-1.3 Ante exceeds user balance", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "poor_user", "poor@test.com")
		setCoins(t, st, u.ID, 50)
		handler := &handlers.TourneyHandler{Store: st}

		payload := models.CreateChallengeRequest{Name: "Expensive", OverallGoalDays: 7, DailyGoalMins: 15, Ante: intPtr(100)}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/tourney", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler.CreateTourney(w, req)

		if w.Result().StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403, got %d. Body: %s", w.Result().StatusCode, w.Body.String())
		}
	})
}

func TestJoinTourney(t *testing.T) {
	t.Run("TC-2.1 Join successfully", func(t *testing.T) {
		st := setupTestStore(t)
		creator := createTestUser(t, st, "creator", "creator@test.com")
		setCoins(t, st, creator.ID, 100)
		joiner := createTestUser(t, st, "joiner", "joiner@test.com")
		setCoins(t, st, joiner.ID, 100)

		ch, _, err := st.CreateChallenge(context.Background(), creator.ID, "Join Test", 7, 15, 50)
		if err != nil {
			t.Fatalf("create failed: %v", err)
		}

		err = st.JoinChallenge(context.Background(), joiner.ID, ch.InviteCode)
		if err != nil {
			t.Fatalf("join failed: %v", err)
		}

		updatedJoiner, _ := st.GetUserByID(context.Background(), joiner.ID)
		if updatedJoiner.Coins != 50 {
			t.Errorf("expected 50 coins remaining for joiner, got %d", updatedJoiner.Coins)
		}

		status, _ := st.BuildTourneyStatus(context.Background(), creator.ID)
		if status.PotTotal != 100 {
			t.Errorf("expected pot 100, got %d", status.PotTotal)
		}
	})

	t.Run("TC-2.3 Join fails after start time", func(t *testing.T) {
		st := setupTestStore(t)
		creator := createTestUser(t, st, "creator", "creator@test.com")
		setCoins(t, st, creator.ID, 100)
		joiner := createTestUser(t, st, "late_joiner", "late@test.com")
		setCoins(t, st, joiner.ID, 100)

		ch, _, _ := st.CreateChallenge(context.Background(), creator.ID, "Late Test", 7, 15, 50)

		pastStart := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
		st.ExecForTest(context.Background(), "UPDATE tourneys SET starttime = ? WHERE id = ?", pastStart, ch.ID)

		err := st.JoinChallenge(context.Background(), joiner.ID, ch.InviteCode)
		if err != store.ErrChallengeStarted {
			t.Fatalf("expected ErrChallengeStarted, got %v", err)
		}
	})
}

func TestProgressTracking(t *testing.T) {
	t.Run("TC-3.1 Reading sessions before/after start time", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "user", "user@test.com")
		setCoins(t, st, u.ID, 100)

		ch, _, _ := st.CreateChallenge(context.Background(), u.ID, "Time Test", 3, 5, 10)
		st.UpsertDailyReadingLog(context.Background(), u.ID, todayUTC(), 10)

		status, _ := st.BuildTourneyStatus(context.Background(), u.ID)
		if status.DailyProgress.MinutesComplete != 0 {
			t.Errorf("expected 0 progress before starttime, got %d", status.DailyProgress.MinutesComplete)
		}

		pastStart := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
		st.ExecForTest(context.Background(), "UPDATE tourneys SET starttime = ? WHERE id = ?", pastStart, ch.ID)

		status2, _ := st.BuildTourneyStatus(context.Background(), u.ID)
		if status2.DailyProgress.MinutesComplete != 10 {
			t.Errorf("expected 10 progress after starttime, got %d", status2.DailyProgress.MinutesComplete)
		}
	})

	t.Run("TC-3.4 & 3.5 Completion and count drop", func(t *testing.T) {
		st := setupTestStore(t)
		u := createTestUser(t, st, "comp", "comp@test.com")
		setCoins(t, st, u.ID, 100)

		ch, _, err := st.CreateChallenge(context.Background(), u.ID, "Comp Test", 1, 5, 50)
		if err != nil {
			t.Fatalf("create failed: %v", err)
		}

		pastStart := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
		st.ExecForTest(context.Background(), "UPDATE tourneys SET starttime = ? WHERE id = ?", pastStart, ch.ID)

		st.UpsertDailyReadingLog(context.Background(), u.ID, todayUTC(), 5)

		status, err := st.BuildTourneyStatus(context.Background(), u.ID)
		if err != nil {
			t.Fatalf("BuildTourneyStatus failed: %v", err)
		}
		if status == nil {
			t.Fatal("expected non-nil status after completion")
		}
		if status.CompletedCount != 1 {
			t.Errorf("expected 1 completed, got %d", status.CompletedCount)
		}
		if status.ChallengerCount != 0 {
			t.Errorf("expected 0 challengers, got %d", status.ChallengerCount)
		}
	})
}

func TestPayoutLogic(t *testing.T) {
	t.Run("TC-4.1 to 4.4 progressive payouts", func(t *testing.T) {
		st := setupTestStore(t)
		users := []*models.User{}
		for i := 1; i <= 4; i++ {
			u := createTestUser(t, st, fmt.Sprintf("u%d", i), fmt.Sprintf("u%d@test.com", i))
			setCoins(t, st, u.ID, 100)
			users = append(users, u)
		}

		ch, _, err := st.CreateChallenge(context.Background(), users[0].ID, "Pot Test", 1, 5, 50)
		if err != nil {
			t.Fatalf("create failed: %v", err)
		}
		st.JoinChallenge(context.Background(), users[1].ID, ch.InviteCode)
		st.JoinChallenge(context.Background(), users[2].ID, ch.InviteCode)
		st.JoinChallenge(context.Background(), users[3].ID, ch.InviteCode)

		pastStart := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
		st.ExecForTest(context.Background(), "UPDATE tourneys SET starttime = ? WHERE id = ?", pastStart, ch.ID)

		// U1 finishes: Gets 100 (50% of 200)
		st.UpsertDailyReadingLog(context.Background(), users[0].ID, todayUTC(), 5)
		status, err := st.BuildTourneyStatus(context.Background(), users[0].ID)
		if err != nil || status == nil {
			t.Fatalf("U1 failed: %v, %v", status, err)
		}
		u0, _ := st.GetUserByID(context.Background(), users[0].ID)
		if u0.Coins != 150 {
			t.Errorf("U1 expected 150, got %d", u0.Coins)
		}

		// U2 finishes: Gets 50 (50% of 100)
		st.UpsertDailyReadingLog(context.Background(), users[1].ID, todayUTC(), 5)
		status, err = st.BuildTourneyStatus(context.Background(), users[1].ID)
		if err != nil || status == nil {
			t.Fatalf("U2 failed: %v, %v", status, err)
		}
		u1, _ := st.GetUserByID(context.Background(), users[1].ID)
		if u1.Coins != 100 {
			t.Errorf("U2 expected 100, got %d", u1.Coins)
		}

		// U3 finishes: Gets 25 (50% of 50)
		st.UpsertDailyReadingLog(context.Background(), users[2].ID, todayUTC(), 5)
		status, err = st.BuildTourneyStatus(context.Background(), users[2].ID)
		if err != nil || status == nil {
			t.Fatalf("U3 failed: %v, %v", status, err)
		}
		u2, _ := st.GetUserByID(context.Background(), users[2].ID)
		if u2.Coins != 75 {
			t.Errorf("U3 expected 75, got %d", u2.Coins)
		}

		// U4 finishes: Gets 25 (100% of 25)
		st.UpsertDailyReadingLog(context.Background(), users[3].ID, todayUTC(), 5)
		status, err = st.BuildTourneyStatus(context.Background(), users[3].ID)
		if err != nil || status == nil {
			t.Fatalf("U4 failed: %v, %v", status, err)
		}
		u3, _ := st.GetUserByID(context.Background(), users[3].ID)
		if u3.Coins != 75 {
			t.Errorf("U4 expected 75, got %d", u3.Coins)
		}
	})
}
