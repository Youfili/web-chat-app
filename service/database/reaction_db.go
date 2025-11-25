package database

import (
	"database/sql"
	"errors"

	"github.com/mattn/go-sqlite3"
)

func createTableReactions(db *sql.DB) error {
	const query = `
	CREATE TABLE IF NOT EXISTS reactions (
		id TEXT PRIMARY KEY,
		message_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		emoji TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		UNIQUE(message_id, user_id),
		FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`
	_, err := db.Exec(query)
	return err
}

func (db *appdbimpl) AddReaction(r Reaction) error {
	query := `INSERT INTO reactions (id, message_id, user_id, emoji, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := db.c.Exec(query, r.ReactionID, r.MessToReactID, r.SenderUserID, r.Emoji, r.Timestamp)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
			return ErrDuplicateReaction
		}
		return err
	}
	return nil
}

func (db *appdbimpl) RemoveReaction(messageID string, userID string) error {
	res, err := db.c.Exec(`DELETE FROM reactions WHERE message_id = ? AND user_id = ?`, messageID, userID)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrConstraint
	}
	return nil
}

func (db *appdbimpl) GetReactions(messageID string) ([]Reaction, error) {
	// Query con JOIN per recuperare anche lo username di chi ha messo la reazione -->  Non l'ho passato prima cosi da avere l'username aggiornato (nel caso venga modificato dall'utente stesso)
	query := `
		SELECT r.id, r.emoji, r.created_at, r.message_id, r.user_id, u.username
		FROM reactions r
		JOIN users u ON r.user_id = u.id
		WHERE r.message_id = ?
		ORDER BY r.created_at ASC
	`

	rows, err := db.c.Query(query, messageID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }() //gestisco l'errore "buttandolo via" con _

	// Inizializzo la slice (se non ci sono risultati, ritornerà un array vuoto [] in JSON invece di null)
	reactions := make([]Reaction, 0)

	for rows.Next() {
		var r Reaction
		err := rows.Scan(
			&r.ReactionID,
			&r.Emoji,
			&r.Timestamp,
			&r.MessToReactID,
			&r.SenderUserID,
			&r.SenderUsername,
		)
		if err != nil {
			return nil, err
		}
		reactions = append(reactions, r)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return reactions, nil
}
