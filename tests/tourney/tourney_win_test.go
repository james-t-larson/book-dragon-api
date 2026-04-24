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
)

func TestFocusTimerTournamentWin(t *testing.T) {
	st := setupTestStore(t)
	u := createTestUser(t, st, "winner", "winner@test.com")
	setCoins(t, st, u.ID, 100)
	book := createTestBook(t, st, u.ID)

	// Create a 1-day tournament with 5 min daily goal
	ch, _, err := st.CreateChallenge(context.Background(), u.ID, "Win Test", 1, 5, 50)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Move start time to 1 hour ago
	pastStart := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	st.ExecForTest(context.Background(), "UPDATE tourneys SET starttime = ? WHERE id = ?", pastStart, ch.ID)

	// Setup handler
	userHandler := &handlers.UserHandler{Store: st}

	// Complete focus timer with 5 minutes (meets daily and overall goal)
	payload := models.FocusTimerRequest{
		Minutes:     5,
		BookID:      book.ID,
		CurrentPage: intPtr(10),
	}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/focus_timer_complete", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), auth.UserContextKey, u.ID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	userHandler.FocusTimerComplete(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", w.Result().StatusCode, w.Body.String())
	}

	var resp models.FocusTimerResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify base coins: (5/5)*3 = 3
	if resp.CoinsEarned != 3 {
		t.Errorf("expected 3 coins earned, got %d", resp.CoinsEarned)
	}

	// Verify tournament winnings: should be 50 (100% of pot since only 1 challenger)
	if resp.TourneyWinnings != 50 {
		t.Errorf("expected 50 tournament winnings, got %d", resp.TourneyWinnings)
	}

	// Verify total coins: 100 (initial) - 50 (ante) + 3 (read) + 50 (win) = 103
	if resp.TotalCoins != 103 {
		t.Errorf("expected 103 total coins, got %d", resp.TotalCoins)
	}

	// Verify tourney status
	if resp.TourneyStatus == nil {
		t.Fatal("expected tourney status in response")
	}
	if !resp.TourneyCompleted {
		t.Errorf("expected tourney_completed to be true")
	}
	if !resp.TourneyStatus.OverallProgress.IsComplete {
		t.Errorf("expected tourney status to show complete")
	}
}
