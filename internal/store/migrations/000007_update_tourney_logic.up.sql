-- Add min_ante, starttime, pot_total, challenger_count, and completed_count to tourneys
ALTER TABLE tourneys ADD COLUMN min_ante INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tourneys ADD COLUMN starttime DATETIME;
ALTER TABLE tourneys ADD COLUMN pot_total INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tourneys ADD COLUMN challenger_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tourneys ADD COLUMN completed_count INTEGER NOT NULL DEFAULT 0;

-- Add payout_claimed to user_challenges
ALTER TABLE user_challenges ADD COLUMN payout_claimed BOOLEAN NOT NULL DEFAULT 0;
