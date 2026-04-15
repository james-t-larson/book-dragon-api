-- Stores global configuration and dynamic UI constants as JSON
CREATE TABLE constants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    content TEXT NOT NULL, -- Stored as a JSON string
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Seed the initial tourney configuration constants
INSERT INTO constants (name, content) VALUES (
    'tourney_config',
    '{
      "overall_goal_days":[
         {"label":"3 days", "value":3},
         {"label":"1 week", "value":7},
         {"label":"2 weeks", "value":14},
         {"label":"1 month", "value":30}
      ],
      "daily_goal_minutes":[
         {"label":"5 minutes", "value":5},
         {"label":"10 minutes", "value":10},
         {"label":"15 minutes", "value":15},
         {"label":"30 minutes", "value":30}
      ]
   }'
);

-- Represents a created challenge and its rules
CREATE TABLE tourneys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    creator_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    invite_code TEXT NOT NULL UNIQUE,
    duration_days INTEGER NOT NULL,
    daily_minutes_goal INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(creator_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Tracks a user's participation and overall status in a challenge
CREATE TABLE user_challenges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    challenge_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'active', -- Options: 'active', 'completed', 'failed'
    start_date DATE NOT NULL, -- Format: YYYY-MM-DD
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, challenge_id),
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(challenge_id) REFERENCES tourneys(id) ON DELETE CASCADE
);

-- Tracks user reading time per calendar day
CREATE TABLE daily_reading_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    reading_date DATE NOT NULL, -- Format: YYYY-MM-DD
    minutes_read INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, reading_date),
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
