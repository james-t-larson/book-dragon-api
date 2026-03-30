package models

import "time"

type User struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	Password    string    `json:"-"` // Omit password hash in responses
	CreatedAt   time.Time `json:"created_at"`
	DragonID    *int64            `json:"dragon_id,omitempty"`
	DragonName  *string           `json:"dragon_name,omitempty"`
	DragonColor *string           `json:"dragon_color,omitempty"`
	Coins       int64             `json:"coins"`
	Books       []UserBookSummary `json:"books,omitempty"`
}

type Dragon struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	UserID    int64     `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateDragonRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type FocusTimerRequest struct {
	Minutes   int   `json:"minutes"`
	BookID    int64 `json:"book_id"`
	PagesRead *int  `json:"pages_read"`
}

type FocusTimerResponse struct {
	CoinsEarned int64 `json:"coins_earned"`
	TotalCoins  int64 `json:"total_coins"`
}
