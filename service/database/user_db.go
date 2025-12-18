package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid" // Per gli UUID univoci
	"github.com/mattn/go-sqlite3"
)

func createTableUsers(db *sql.DB) error {
	const query = `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		profile_photo TEXT,
		status TEXT
	);`
	_, err := db.Exec(query)

	if !errors.Is(err, nil) {
		return fmt.Errorf("error while creating user table: %w", err)
	}
	return nil
}

func (db *appdbimpl) CreateUser(u User) (User, error) {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	query := `INSERT INTO users (id, username, profile_photo, status) VALUES (?, ?, ?, ?)`
	_, err := db.c.Exec(query, u.ID, u.Username, u.ProfilePhoto, u.Status)

	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
			return User{}, ErrUsernameTaken
		}
		return User{}, err
	}
	return u, nil
}

func (db *appdbimpl) GetUserByID(id string) (User, error) {
	var u User
	query := `SELECT id, username, profile_photo, status FROM users WHERE id = ?`
	err := db.c.QueryRow(query, id).Scan(&u.ID, &u.Username, &u.ProfilePhoto, &u.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	return u, err
}

func (db *appdbimpl) GetUserByUsername(username string) (User, error) {
	var u User
	query := `SELECT id, username, profile_photo, status FROM users WHERE username = ?`
	err := db.c.QueryRow(query, username).Scan(&u.ID, &u.Username, &u.ProfilePhoto, &u.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	return u, err
}

func (db *appdbimpl) SearchUsers(queryParam string) ([]User, error) {
	var users []User
	// Fuzzy Search --> invece di cercare solo corrispondenze perfette, cerca anche corrispondenze simili
	query := `SELECT id, username, profile_photo, status FROM users WHERE username LIKE ?`
	rows, err := db.c.Query(query, "%"+queryParam+"%")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.ProfilePhoto, &u.Status); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (db *appdbimpl) UpdateUsername(userID string, newUsername string) (User, error) {
	query := `UPDATE users SET username = ? WHERE id = ?`
	_, err := db.c.Exec(query, newUsername, userID)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
			return User{}, ErrUsernameTaken
		}
		return User{}, err
	}
	return db.GetUserByID(userID)
}

func (db *appdbimpl) SetUserPhoto(userID string, photoURL string) error {
	res, err := db.c.Exec(`UPDATE users SET profile_photo = ? WHERE id = ?`, photoURL, userID)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected() // Numero di righe modificate dalla query SQL
	if aff == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (db *appdbimpl) SetUserStatus(userID string, status string) error {
	res, err := db.c.Exec(`UPDATE users SET status = ? WHERE id = ?`, status, userID)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrUserNotFound
	}
	return nil
}
