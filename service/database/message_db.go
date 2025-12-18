package database

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

func createTableMessages(db *sql.DB) error {
	const query = `
	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		conversation_id TEXT NOT NULL,
		sender_id TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME NOT NULL,		
		is_forwarded BOOLEAN DEFAULT 0,
		FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
		FOREIGN KEY (sender_id) REFERENCES users(id)
	);`
	_, err := db.Exec(query)
	return err
}

func (db *appdbimpl) CreateMessage(msg Message) (Message, error) {
	tx, err := db.c.Begin()
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback() }()

	// Se l'ID non c'è, viene generato
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}

	// Se il Timestamp è "zero" (non settato), metto ADESSO.
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now().UTC() // Uso UTC.
	}

	// Insert nel DB
	_, err = tx.Exec(`INSERT INTO messages (id, conversation_id, sender_id, content, created_at, is_forwarded) VALUES (?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.ConversationID, msg.SenderUserID, msg.ContentMess, msg.Timestamp, msg.Forwarded)
	if err != nil {
		return Message{}, err
	}

	// Visto che ho creato un nuovo messaggio in questo orario, aggiorno l'orario dell'ultima attività della conversazione
	_, err = tx.Exec(`UPDATE conversations SET last_message_at = ? WHERE id = ?`, msg.Timestamp, msg.ConversationID)
	if err != nil {
		return Message{}, err
	}

	if err := tx.Commit(); err != nil {
		return Message{}, err
	}

	// Appena creato, il messaggio, è sicuramente delivered (e non 'read')
	msg.StatusInfo = "delivered"
	return msg, nil
}

func (db *appdbimpl) GetMessages(username string, conversationID string, limit int, before time.Time) ([]Message, error) {

	// Recupero il MIO ID utente (per escludermi dal conteggio letture)
	user, err := db.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}

	// Recupero il timestamp dell'ultimo messaggio letto DALL'ALTRO utente (o dagli altri nel caso di una conversazione di gruppo)
	var lastReadStr sql.NullString

	// Questa query trova il timestamp del messaggio letto più "vecchio" tra gli altri partecipanti.
	timeQuery := `
		SELECT MAX(m.created_at)
        FROM participants p
        JOIN messages m ON p.last_read_message_id = m.id
        WHERE p.conversation_id = ? AND p.user_id != ?
	`

	// Eseguo la query scansionando in una STRINGA
	errQuery := db.c.QueryRow(timeQuery, conversationID, user.ID).Scan(&lastReadStr)

	// Variabile dove salverò la data convertita
	var otherLastReadTime time.Time
	var hasValidReadTime bool // di default è false

	if errQuery != nil {
		// Se c'è un errore SQL
	} else if lastReadStr.Valid {
		// Ho trovato una Data --> Devo convertirla in time.Time.
		// Faccio un Parsing

		// Layout standard che usa Go nel DB
		layout := "2006-01-02 15:04:05.999999999-07:00"
		t, errParse := time.Parse(layout, lastReadStr.String)

		if errParse != nil {
			// Fallback
			t, errParse = time.Parse("2006-01-02 15:04:05", lastReadStr.String)
		}

		if errParse == nil {
			otherLastReadTime = t
			hasValidReadTime = true
		}
	}

	// Recupero i Messaggi
	query := `
		SELECT m.id, m.content, m.created_at, m.is_forwarded, m.sender_id, u.username
        FROM messages m
        JOIN users u ON m.sender_id = u.id
        WHERE m.conversation_id = ? AND m.created_at < ?
        ORDER BY m.created_at DESC
        LIMIT ?
	`
	rows, err := db.c.Query(query, conversationID, before, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var msgs []Message

	for rows.Next() {
		var m Message
		err := rows.Scan(&m.ID, &m.ContentMess, &m.Timestamp, &m.Forwarded, &m.SenderUserID, &m.SenderUsername)
		if err != nil {
			defer func() { _ = rows.Close() }()
			return nil, err
		}
		m.ConversationID = conversationID

		// ----------------------------------------------------------------

		// Logica Assegnazione Stato
		m.StatusInfo = "delivered" // Default

		// Se ho trovato e convertito validamente la data di lettura dell'altro...
		if hasValidReadTime {
			// --> confronto le date
			if !m.Timestamp.After(otherLastReadTime) {
				m.StatusInfo = "read"
			}
		}

		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }() // Chiudo la connessione della query principale

	// ----------------------------------------------------------------------
	// Popolo le Reazioni per questo messaggio
	// Chiamo la funzione GetReactions che ho implementato in reaction_db.go
	// ------------------------------------------------------------------------
	for i := range msgs {
		// Uso l'indice per modificare direttamente l'elemento nell'array
		reactions, err := db.GetReactions(msgs[i].ID)
		if err != nil {
			msgs[i].Reactions = []Reaction{}
		} else {
			msgs[i].Reactions = reactions
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return msgs, nil
}

// GetMessageByID recupera un messaggio specifico (lo uso per Edit, Delete e Forward)
func (db *appdbimpl) GetMessageByID(messageID string) (Message, error) {
	var m Message

	// Query con JOIN per avere anche lo username del mittente
	query := `
		SELECT m.id, m.conversation_id, m.sender_id, m.content, m.created_at, m.is_forwarded, u.username
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		WHERE m.id = ?
	`
	err := db.c.QueryRow(query, messageID).Scan(
		&m.ID,
		&m.ConversationID,
		&m.SenderUserID,
		&m.ContentMess,
		&m.Timestamp,
		&m.Forwarded,
		&m.SenderUsername,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return Message{}, ErrMessageNotFound
	}
	if err != nil {
		return Message{}, err
	}

	// -------------------------------------
	// Popolo le reazioni
	reactions, err := db.GetReactions(m.ID)
	if err != nil {
		m.Reactions = []Reaction{}
	} else {
		m.Reactions = reactions
	}
	// --------------------------------------

	return m, nil
}

func (db *appdbimpl) EditMessage(messageID string, newContent string) (Message, error) {
	// Eseguo l'UPDATE del contenuto
	res, err := db.c.Exec(`UPDATE messages SET content = ? WHERE id = ?`, newContent, messageID)
	if err != nil {
		return Message{}, err
	}

	// Controllo se il messaggio esisteva
	affected, err := res.RowsAffected() // Se RowsAffected è 0, significa che l'ID non è stato trovato.
	if err != nil {
		return Message{}, err
	}
	if affected == 0 {
		return Message{}, ErrMessageNotFound
	}

	// Recupero l'oggetto "messaggio" completo per restituirlo
	// Devo fare una JOIN con users per ripopolare il campo SenderUsername --> che serve al frontend per visualizzare il messaggio correttamente.
	var msg Message
	query := `
		SELECT m.id, m.conversation_id, m.sender_id, m.content, m.created_at, m.is_forwarded, u.username
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		WHERE m.id = ?
	`
	err = db.c.QueryRow(query, messageID).Scan(
		&msg.ID,
		&msg.ConversationID,
		&msg.SenderUserID,
		&msg.ContentMess, // Questo conterrà il nuovo testo (messaggio modificato)
		&msg.Timestamp,
		&msg.Forwarded,
		&msg.SenderUsername,
	)
	if err != nil {
		return Message{}, err
	}

	// Imposto lo status di default (come in CreateMessage)
	msg.StatusInfo = "delivered"
	return msg, nil
}

func (db *appdbimpl) DeleteMessage(messageID string) error {
	res, err := db.c.Exec(`DELETE FROM messages WHERE id = ?`, messageID)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrMessageNotFound
	}
	return nil
}

// Qui aggiorno il "cursore" di lettura
func (db *appdbimpl) MarkConversationAsRead(conversationID string, userReaderID string, lastMessageID string) error {
	// Aggiorna l'ultimo messaggio letto per questo utente in questa chat
	_, err := db.c.Exec(`
        UPDATE participants 
        SET last_read_message_id = ? 
        WHERE conversation_id = ? AND user_id = ?`,
		lastMessageID, conversationID, userReaderID)
	return err
}
