-- Migration: add reading column to user_books
ALTER TABLE user_books ADD COLUMN reading BOOLEAN NOT NULL DEFAULT FALSE;
