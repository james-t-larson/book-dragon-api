package models

// CreateChallengeRequest is used for POST /tourney to create a new challenge.
type CreateChallengeRequest struct {
	Name            string `json:"name"`
	OverallGoalDays int    `json:"overall_goal_days"`
	DailyGoalMins   int    `json:"daily_goal_minutes"`
	Ante            int    `json:"ante"`
}

// JoinChallengeRequest is used for POST /join_tourney to join a challenge by invite code.
type JoinChallengeRequest struct {
	InviteCode string `json:"invite_code"`
}

// DailyProgress represents the user's reading progress for the current day.
type DailyProgress struct {
	MinutesComplete int  `json:"minutes_complete"`
	MinuteGoal      int  `json:"minute_goal"`
	IsComplete      bool `json:"is_complete"`
}

// OverallProgress represents the user's overall progress in a challenge.
type OverallProgress struct {
	DayNumber    int  `json:"day_number"`
	DaysComplete int  `json:"days_complete"`
	DaysGoal     int  `json:"days_goal"`
	IsComplete   bool `json:"is_complete"`
}

// TourneyStatusResponse is the full payload describing a user's active challenge state.
// An empty/nil value means no active challenge.
type TourneyStatusResponse struct {
	ID              int64           `json:"id"`
	Name            string          `json:"name"`
	InviteCode      string          `json:"invite_code"`
	StartTime       string          `json:"starttime"` // RFC3339 string
	PotTotal        int             `json:"pot_total"`
	ChallengerCount int             `json:"challenger_count"`
	CompletedCount  int             `json:"completed_count"`
	TauntMessages   []string        `json:"taunt_messages"`
	DailyProgress   DailyProgress   `json:"daily_progress"`
	OverallProgress OverallProgress `json:"overall_progress"`
}

// ConstantOption represents a label/value pair for tourney configuration.
type ConstantOption struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

// TourneyConfig holds the options for creating a tourney.
type TourneyConfig struct {
	OverallGoalDays  []ConstantOption `json:"overall_goal_days"`
	DailyGoalMinutes []ConstantOption `json:"daily_goal_minutes"`
}

// TourneyConstantsResponse is the API response for GET /constants.
type TourneyConstantsResponse struct {
	TourneyConfig TourneyConfig `json:"tourney_config"`
}

// Challenge is the internal representation of a tourney row from the database.
type Challenge struct {
	ID               int64
	CreatorID        int64
	Name             string
	InviteCode       string
	DurationDays     int
	DailyMinutesGoal int
	MinAnte          int
	StartTime        string
	PotTotal         int
	ChallengerCount  int
	CompletedCount   int
}

// UserChallenge represents a row from the user_challenges table.
type UserChallenge struct {
	ID            int64
	UserID        int64
	ChallengeID   int64
	Status        string
	StartDate     string // YYYY-MM-DD
	PayoutClaimed bool
}
