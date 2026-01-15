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
		photo_url TEXT,
		reply_to TEXT,
		created_at DATETIME NOT NULL,		
		is_forwarded BOOLEAN DEFAULT 0,
		status TEXT DEFAULT 'sent',
		FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
		FOREIGN KEY (sender_id) REFERENCES users(id)
		FOREIGN KEY (reply_to) REFERENCES messages(id) ON DELETE SET NULL
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
	_, err = tx.Exec(`INSERT INTO messages (id, conversation_id, sender_id, content, photo_url, reply_to, created_at, is_forwarded, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.ConversationID, msg.SenderUserID, msg.ContentMess, msg.MessagePhoto, msg.ReplyTo, msg.Timestamp, msg.Forwarded, "sent")
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

	// Appena creato, il messaggio, è sicuramente "sent"
	msg.StatusInfo = "sent"
	return msg, nil
}

func (db *appdbimpl) GetMessages(username string, conversationID string, limit int, before time.Time) ([]Message, error) {

	// Recupero il MIO ID utente (per escludermi dal conteggio letture)
	user, err := db.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}

	// ---
	// Aggiorno a 'delivered' i messaggi che sto scaricando (se non li ho inviati io e sono ancora 'sent')
	_, _ = db.c.Exec(`
        UPDATE messages 
        SET status = 'delivered' 
        WHERE conversation_id = ? 
          AND sender_id != ? 
          AND status = 'sent'
    `, conversationID, user.ID)
	// ---

	// Logica per Spunte Blu di Gruppo

	// Conto quanti sono gli ALTRI partecipanti (escluso io)	--> Conto TUTTI i membri (escluso me)
	var totalOtherMembers int
	errCount := db.c.QueryRow(`
        SELECT COUNT(*) 
        FROM participants 
        WHERE conversation_id = ? AND user_id != ?
    `, conversationID, user.ID).Scan(&totalOtherMembers)

	if errCount != nil {
		return nil, errCount
	}

	// Cerco il MINIMO tempo di lettura e CONTO quanti utenti hanno una lettura valida
	// La JOIN esclude automaticamente chi ha last_read_message_id = NULL (es. un utente appena entrato)
	var minReadTimeStr sql.NullString
	var membersWhoHaveReadCount int

	timeQuery := `
        SELECT MIN(m.created_at), COUNT(p.user_id)
        FROM participants p
        JOIN messages m ON p.last_read_message_id = m.id
        WHERE p.conversation_id = ? AND p.user_id != ?
    `
	_ = db.c.QueryRow(timeQuery, conversationID, user.ID).Scan(&minReadTimeStr, &membersWhoHaveReadCount)

	// Valuto se TUTTI hanno letto
	// - totalOtherMembers > 0: La chat non è vuota
	// - totalOtherMembers == membersWhoHaveReadCount: Nessuno ha NULL (tutti hanno letto qualcosa)
	// - minReadTimeStr.Valid: Ho una data valida

	var groupReadTime time.Time
	var everyoneHasReadAtLeastSomething bool

	if totalOtherMembers > 0 && totalOtherMembers == membersWhoHaveReadCount && minReadTimeStr.Valid {
		// Parsing della data
		layout := "2006-01-02 15:04:05.999999999-07:00"
		t, errParse := time.Parse(layout, minReadTimeStr.String)
		if errParse != nil {
			t, _ = time.Parse("2006-01-02 15:04:05", minReadTimeStr.String) // Fallback
		}

		groupReadTime = t
		everyoneHasReadAtLeastSomething = true
	}

	// Recupero i Messaggi
	query := `
		SELECT m.id, m.content, m.photo_url, m.reply_to, m.created_at, m.is_forwarded, m.status, m.sender_id, u.username
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
		var photoUrl sql.NullString // Variabile temporanea per gestire il NULL della foto
		var replyTo sql.NullString  // Variabile per gestire il NULL del "reply message"
		var statusDb string         // Variabile per la gestione dello stato del messaggio

		err := rows.Scan(&m.ID, &m.ContentMess, &photoUrl, &replyTo, &m.Timestamp, &m.Forwarded, &statusDb, &m.SenderUserID, &m.SenderUsername)
		if err != nil {
			defer func() { _ = rows.Close() }()
			return nil, err
		}

		// Converto sql.NullString in *string per la struct
		if photoUrl.Valid {
			m.MessagePhoto = &photoUrl.String
		}

		// Gestione ReplyTo
		if replyTo.Valid {
			m.ReplyTo = &replyTo.String
		}

		m.ConversationID = conversationID

		// ----------------------------------------------------------------

		// Logica Assegnazione Stato
		m.StatusInfo = statusDb // Prendo quello che c'è nel DB ('sent' o 'delivered')

		// Sovrascrivo con 'read' SOLO SE:
		// 1° --> So che TUTTI i partecipanti hanno una ricevuta di lettura valida (nessuno è NULL)
		// 2° --> Il messaggio è più vecchio (o uguale) al momento in cui il "più lento" del gruppo ha letto.
		if everyoneHasReadAtLeastSomething {
			if !m.Timestamp.After(groupReadTime) {
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
	var photoUrl sql.NullString // Variabile per gestire il NULL della foto
	var replyTo sql.NullString  // Variabile per gestire il NULL della replyTo

	// Query con JOIN per avere anche lo username del mittente
	query := `
		SELECT m.id, m.conversation_id, m.sender_id, m.content, m.photo_url, m.reply_to, m.created_at, m.is_forwarded, m.status, u.username
        FROM messages m
        JOIN users u ON m.sender_id = u.id
        WHERE m.id = ?
	`

	var statusDb string

	// Scan
	err := db.c.QueryRow(query, messageID).Scan(
		&m.ID,
		&m.ConversationID,
		&m.SenderUserID,
		&m.ContentMess,
		&photoUrl, // Recupero la foto
		&replyTo,  // Recupero il messaggio a cui "Risponde"
		&m.Timestamp,
		&m.Forwarded,
		&statusDb,
		&m.SenderUsername,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return Message{}, ErrMessageNotFound
	}
	if err != nil {
		return Message{}, err
	}

	// Se c'è una foto valida nel DB, la assegno alla struct
	if photoUrl.Valid {
		m.MessagePhoto = &photoUrl.String
	}

	// Se c'è un reply_to, lo assegno
	if replyTo.Valid {
		m.ReplyTo = &replyTo.String
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

	// Assegno lo stato
	m.StatusInfo = statusDb

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

	// Recupero l'oggetto "messaggio" completo per restituirlo (inclusa la Foto se c'è)
	// Devo fare una JOIN con users per ripopolare il campo SenderUsername --> che serve al frontend per visualizzare il messaggio correttamente.
	var msg Message
	var photoUrl sql.NullString // Variabile per gestire il possibile NULL nel DB
	var replyTo sql.NullString  // Variabile per gestire il possibile NULL nel DB

	query := `
		SELECT m.id, m.conversation_id, m.sender_id, m.content, m.photo_url, m.reply_to, m.created_at, m.is_forwarded, u.username
        FROM messages m
        JOIN users u ON m.sender_id = u.id
        WHERE m.id = ?
	`
	err = db.c.QueryRow(query, messageID).Scan(
		&msg.ID,
		&msg.ConversationID,
		&msg.SenderUserID,
		&msg.ContentMess, // Questo conterrà il nuovo testo (messaggio modificato)
		&photoUrl,        // Aggiungo &photoUrl allo SCAN --> Recupero la foto
		&replyTo,         // Aggiungo &replyTo allo SCAN --> Recupero il messaggio a cui rispondo
		&msg.Timestamp,
		&msg.Forwarded,
		&msg.SenderUsername,
	)
	if err != nil {
		return Message{}, err
	}

	// Se c'è una foto, la assegno alla struct
	if photoUrl.Valid {
		msg.MessagePhoto = &photoUrl.String
	}

	// Se NON è Null (quindi rispondo a un messaggio specifico), lo assegno alla struct
	if replyTo.Valid {
		msg.ReplyTo = &replyTo.String
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
