-- Migration down: remove reading column from user_books
ALTER TABLE user_books DROP COLUMN reading;
