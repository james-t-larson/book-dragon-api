package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"book-dragon/internal/models"
)

var (
	ErrActiveChallenge     = errors.New("user already has an active challenge")
	ErrInviteCodeNotFound  = errors.New("invite code not found")
	ErrAlreadyEnrolled     = errors.New("user is already enrolled in this challenge")
	ErrInviteCodeCollision = errors.New("failed to generate unique invite code after retries")
)

// tauntMessages are randomly returned when the user hasn't completed their daily goal.
var tauntMessages = []string{
	"Only 5 minutes? The dragon slumbers!",
	"A true knight doesn't rest so soon!",
	"Read more, or the kingdom falls!",
	"Your dragon grows restless...",
	"The pages aren't going to turn themselves!",
}

// allowedDurations are the valid overall_goal_days values from the constants table.
var allowedDurations = map[int]bool{3: true, 7: true, 14: true, 30: true}

// allowedDailyMinutes are the valid daily_goal_minutes values from the constants table.
var allowedDailyMinutes = map[int]bool{5: true, 10: true, 15: true, 30: true}

// normalizeSQLiteDate converts SQLite date strings (which may include time components
// like "2006-01-02T00:00:00Z") into a consistent "2006-01-02" format.
func normalizeSQLiteDate(dateStr string) string {
	// Try full timestamp first
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return t.Format("2006-01-02")
	}
	// Try date-only format
	if t, err := time.Parse("2006-01-02", dateStr); err == nil {
		return t.Format("2006-01-02")
	}
	// Fallback: return first 10 chars if long enough
	if len(dateStr) >= 10 {
		return dateStr[:10]
	}
	return dateStr
}

// generateInviteCode produces a random 8-character alphanumeric code in XXXX-XXXX format.
func generateInviteCode() (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 8)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		code[i] = charset[n.Int64()]
	}
	return fmt.Sprintf("%s-%s", string(code[:4]), string(code[4:])), nil
}

// GetConstants fetches a constants row by name and returns the raw JSON content.
func (s *Store) GetConstants(ctx context.Context, name string) (string, error) {
	queryString := `SELECT content FROM constants WHERE name = ?`
	var content string
	err := s.queryRow(ctx, queryString, name).Scan(&content)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("constant %q not found", name)
		}
		return "", err
	}
	return content, nil
}

// GetTourneyConfig fetches and parses the tourney_config constant.
func (s *Store) GetTourneyConfig(ctx context.Context) (*models.TourneyConfig, error) {
	content, err := s.GetConstants(ctx, "tourney_config")
	if err != nil {
		return nil, err
	}
	var config models.TourneyConfig
	if err := json.Unmarshal([]byte(content), &config); err != nil {
		return nil, fmt.Errorf("failed to parse tourney_config: %w", err)
	}
	return &config, nil
}

// GetActiveUserChallenge fetches the user's active challenge and its tourney details.
// Returns nil, nil if no active challenge exists.
func (s *Store) GetActiveUserChallenge(ctx context.Context, userID int64) (*models.UserChallenge, *models.Challenge, error) {
	queryString := `
		SELECT uc.id, uc.user_id, uc.challenge_id, uc.status, uc.start_date,
		       t.id, t.creator_id, t.name, t.invite_code, t.duration_days, t.daily_minutes_goal
		FROM user_challenges uc
		JOIN tourneys t ON uc.challenge_id = t.id
		WHERE uc.user_id = ? AND uc.status = 'active'
		LIMIT 1
	`
	row := s.queryRow(ctx, queryString, userID)

	var uc models.UserChallenge
	var ch models.Challenge
	err := row.Scan(
		&uc.ID, &uc.UserID, &uc.ChallengeID, &uc.Status, &uc.StartDate,
		&ch.ID, &ch.CreatorID, &ch.Name, &ch.InviteCode, &ch.DurationDays, &ch.DailyMinutesGoal,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	// Normalize date format from SQLite (may include time component)
	uc.StartDate = normalizeSQLiteDate(uc.StartDate)
	return &uc, &ch, nil
}

// CompleteExpiredChallenges checks if the user's active challenge has expired
// and updates its status to 'completed' if so.
// Returns true if a challenge was expired.
func (s *Store) CompleteExpiredChallenges(ctx context.Context, userID int64) (bool, error) {
	uc, ch, err := s.GetActiveUserChallenge(ctx, userID)
	if err != nil {
		return false, err
	}
	if uc == nil {
		return false, nil
	}

	startDate, err := time.Parse("2006-01-02", uc.StartDate)
	if err != nil {
		return false, fmt.Errorf("failed to parse start_date: %w", err)
	}

	endDate := startDate.AddDate(0, 0, ch.DurationDays)
	now := time.Now().UTC().Truncate(24 * time.Hour)

	if now.After(endDate) || now.Equal(endDate) {
		queryString := `UPDATE user_challenges SET status = 'completed' WHERE id = ?`
		_, err := s.exec(ctx, queryString, uc.ID)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// GetDailyReadingLog fetches the reading log for a specific user and date.
func (s *Store) GetDailyReadingLog(ctx context.Context, userID int64, date string) (int, error) {
	queryString := `SELECT minutes_read FROM daily_reading_logs WHERE user_id = ? AND DATE(reading_date) = DATE(?)`
	var minutes int
	err := s.queryRow(ctx, queryString, userID, date).Scan(&minutes)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return minutes, nil
}

// GetDailyReadingLogsForRange fetches reading logs for a user within a date range.
func (s *Store) GetDailyReadingLogsForRange(ctx context.Context, userID int64, startDate, endDate string) (map[string]int, error) {
	queryString := `
		SELECT reading_date, minutes_read
		FROM daily_reading_logs
		WHERE user_id = ? AND DATE(reading_date) >= DATE(?) AND DATE(reading_date) <= DATE(?)
	`
	rows, err := s.query(ctx, queryString, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make(map[string]int)
	for rows.Next() {
		var date string
		var minutes int
		if err := rows.Scan(&date, &minutes); err != nil {
			return nil, err
		}
		logs[normalizeSQLiteDate(date)] = minutes
	}
	return logs, nil
}

// UpsertDailyReadingLog inserts or updates the daily reading log for a user.
func (s *Store) UpsertDailyReadingLog(ctx context.Context, userID int64, date string, minutes int) error {
	queryString := `
		INSERT INTO daily_reading_logs (user_id, reading_date, minutes_read)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id, reading_date)
		DO UPDATE SET
			minutes_read = minutes_read + excluded.minutes_read,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := s.exec(ctx, queryString, userID, date, minutes)
	return err
}

// CreateChallenge creates a new tourney and enrolls the creator.
// Returns the created challenge and user_challenge.
func (s *Store) CreateChallenge(ctx context.Context, creatorID int64, name string, durationDays, dailyGoalMins int) (*models.Challenge, *models.UserChallenge, error) {
	// Validate against allowed constants
	if !allowedDurations[durationDays] {
		return nil, nil, fmt.Errorf("invalid duration_days: %d", durationDays)
	}
	if !allowedDailyMinutes[dailyGoalMins] {
		return nil, nil, fmt.Errorf("invalid daily_goal_minutes: %d", dailyGoalMins)
	}

	// Check user doesn't already have an active challenge
	uc, _, err := s.GetActiveUserChallenge(ctx, creatorID)
	if err != nil {
		return nil, nil, err
	}
	if uc != nil {
		return nil, nil, ErrActiveChallenge
	}

	// Generate invite code with retry for uniqueness
	var inviteCode string
	var insertErr error
	maxRetries := 3
	var challengeID int64

	for i := 0; i < maxRetries; i++ {
		inviteCode, err = generateInviteCode()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate invite code: %w", err)
		}

		insertQuery := `INSERT INTO tourneys (creator_id, name, invite_code, duration_days, daily_minutes_goal) VALUES (?, ?, ?, ?, ?)`
		result, err := s.exec(ctx, insertQuery, creatorID, name, inviteCode, durationDays, dailyGoalMins)
		if err != nil {
			if err.Error() == "UNIQUE constraint failed: tourneys.invite_code" {
				insertErr = err
				continue
			}
			return nil, nil, err
		}
		challengeID, err = result.LastInsertId()
		if err != nil {
			return nil, nil, err
		}
		insertErr = nil
		break
	}

	if insertErr != nil {
		return nil, nil, ErrInviteCodeCollision
	}

	// Enroll the creator
	today := time.Now().UTC().Format("2006-01-02")
	enrollQuery := `INSERT INTO user_challenges (user_id, challenge_id, status, start_date) VALUES (?, ?, 'active', ?)`
	_, err = s.exec(ctx, enrollQuery, creatorID, challengeID, today)
	if err != nil {
		return nil, nil, err
	}

	challenge := &models.Challenge{
		ID:               challengeID,
		CreatorID:        creatorID,
		Name:             name,
		InviteCode:       inviteCode,
		DurationDays:     durationDays,
		DailyMinutesGoal: dailyGoalMins,
	}

	userChallenge := &models.UserChallenge{
		UserID:      creatorID,
		ChallengeID: challengeID,
		Status:      "active",
		StartDate:   today,
	}

	return challenge, userChallenge, nil
}

// JoinChallenge enrolls a user in an existing challenge by invite code.
func (s *Store) JoinChallenge(ctx context.Context, userID int64, inviteCode string) error {
	// Check user doesn't already have an active challenge
	uc, _, err := s.GetActiveUserChallenge(ctx, userID)
	if err != nil {
		return err
	}
	if uc != nil {
		return ErrActiveChallenge
	}

	// Look up the challenge by invite code
	queryString := `SELECT id FROM tourneys WHERE invite_code = ?`
	var challengeID int64
	err = s.queryRow(ctx, queryString, inviteCode).Scan(&challengeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInviteCodeNotFound
		}
		return err
	}

	// Check if already enrolled in this specific challenge (any status)
	checkQuery := `SELECT 1 FROM user_challenges WHERE user_id = ? AND challenge_id = ?`
	var exists int
	err = s.queryRow(ctx, checkQuery, userID, challengeID).Scan(&exists)
	if err == nil {
		return ErrAlreadyEnrolled
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	// Enroll the user
	today := time.Now().UTC().Format("2006-01-02")
	enrollQuery := `INSERT INTO user_challenges (user_id, challenge_id, status, start_date) VALUES (?, ?, 'active', ?)`
	_, err = s.exec(ctx, enrollQuery, userID, challengeID, today)
	return err
}

// BuildTourneyStatus constructs the full TourneyStatusResponse for a user.
// It performs lazy evaluation: checking expirations, calculating daily and overall progress.
// Returns nil if no active challenge.
func (s *Store) BuildTourneyStatus(ctx context.Context, userID int64) (*models.TourneyStatusResponse, error) {
	// First, expire any completed challenges
	_, err := s.CompleteExpiredChallenges(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get current active challenge
	uc, ch, err := s.GetActiveUserChallenge(ctx, userID)
	if err != nil {
		return nil, err
	}
	if uc == nil {
		return nil, nil
	}

	// Parse start date
	startDate, err := time.Parse("2006-01-02", uc.StartDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse start_date: %w", err)
	}

	now := time.Now().UTC().Truncate(24 * time.Hour)
	today := now.Format("2006-01-02")

	// Calculate day number (1-indexed)
	dayNumber := int(now.Sub(startDate).Hours()/24) + 1
	if dayNumber > ch.DurationDays {
		dayNumber = ch.DurationDays
	}

	// Get today's reading
	minutesToday, err := s.GetDailyReadingLog(ctx, userID, today)
	if err != nil {
		return nil, err
	}

	// Calculate days complete (days where goal was met)
	endDateStr := startDate.AddDate(0, 0, ch.DurationDays-1).Format("2006-01-02")
	logs, err := s.GetDailyReadingLogsForRange(ctx, userID, uc.StartDate, endDateStr)
	if err != nil {
		return nil, err
	}

	daysComplete := 0
	for _, minutes := range logs {
		if minutes >= ch.DailyMinutesGoal {
			daysComplete++
		}
	}

	dailyComplete := minutesToday >= ch.DailyMinutesGoal
	overallComplete := dayNumber >= ch.DurationDays && daysComplete >= ch.DurationDays

	// Build taunt messages only if daily goal not met
	var taunts []string
	if !dailyComplete {
		taunts = tauntMessages
	}

	return &models.TourneyStatusResponse{
		ID:            ch.ID,
		Name:          ch.Name,
		InviteCode:    ch.InviteCode,
		TauntMessages: taunts,
		DailyProgress: models.DailyProgress{
			MinutesComplete: minutesToday,
			MinuteGoal:      ch.DailyMinutesGoal,
			IsComplete:      dailyComplete,
		},
		OverallProgress: models.OverallProgress{
			DayNumber:    dayNumber,
			DaysComplete: daysComplete,
			DaysGoal:     ch.DurationDays,
			IsComplete:   overallComplete,
		},
	}, nil
}

// SetChallengeStartDate is a test helper that backdates a user's active challenge
// start_date by the given number of days. This allows expiration tests without
// mocking time.Now().
func (s *Store) SetChallengeStartDate(ctx context.Context, userID int64, daysAgo int) {
	pastDate := time.Now().UTC().AddDate(0, 0, -daysAgo).Format("2006-01-02")
	queryString := `UPDATE user_challenges SET start_date = ? WHERE user_id = ? AND status = 'active'`
	s.exec(ctx, queryString, pastDate, userID)
}
