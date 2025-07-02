package models

import (
	"database/sql"
	"errors"

	_ "github.com/mattn/go-sqlite3"
)

var (
	DB                *sql.DB
	ErrNotInitialized = errors.New("database is not initialized")
)

func Initialize(connString string) error {
	db, err := sql.Open("sqlite3", connString)
	if err != nil {
		return err
	}

	DB = db

	// Run table migrations
	if err := (DirectMessage{}).CreateTable(); err != nil {
		return err
	}

	return nil
}
