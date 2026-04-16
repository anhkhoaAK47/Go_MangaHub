package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

func InitDB(filepath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE,
		password_hash TEXT,
		email TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS manga (
		id TEXT PRIMARY KEY,
		title TEXT,
		author TEXT,
		artist TEXT,
		genres TEXT,
		status TEXT,
		year INTEGER,
		total_chapters INTEGER,
		total_volumes INTEGER,
		serialization TEXT,
		publisher TEXT,
		description TEXT,
		my_anime_list TEXT,
		manga_dx TEXT
	);
	CREATE TABLE IF NOT EXISTS user_progress (
		user_id TEXT,
		manga_id TEXT,
		current_chapter INTEGER,
		status TEXT,
		rating INTEGER,
		started_reading TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, manga_id)
	);
	`
	_, err = db.Exec(schema)
	return db, err
	
}
