package database

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

func createTableConversations(db *sql.DB) error {
	const query = `
	CREATE TABLE IF NOT EXISTS conversations (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL, 
		group_name TEXT,
		group_photo TEXT,
		group_description TEXT,
		last_message_at DATETIME
	);`
	_, err := db.Exec(query)
	return err
}

func createTableParticipants(db *sql.DB) error {
	const query = `
	CREATE TABLE IF NOT EXISTS participants (
		conversation_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		is_admin BOOLEAN DEFAULT 0,
		last_read_message_id TEXT,
		PRIMARY KEY (conversation_id, user_id),
		FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`
	_, err := db.Exec(query)
	return err
}

// GetConversations: Logica ibrida
func (db *appdbimpl) GetConversations(userID string) ([]Conversation, error) {
	query := `
		SELECT c.id, c.type, c.group_name, c.group_photo, c.group_description, c.last_message_at
		FROM conversations c
		JOIN participants p ON c.id = p.conversation_id
		WHERE p.user_id = ?
		ORDER BY c.last_message_at DESC
	`
	rows, err := db.c.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var conversations []Conversation

	for rows.Next() {
		var c Conversation
		var gName, gPhoto, gDesc sql.NullString
		var lastMsgAt sql.NullTime

		err := rows.Scan(&c.ID, &c.ConversationType, &gName, &gPhoto, &gDesc, &lastMsgAt)
		if err != nil {
			return nil, err
		}

		if lastMsgAt.Valid {
			c.DtLastMessage = lastMsgAt.Time
		} else {
			c.DtLastMessage = time.Now()
		}

		// Get snippet
		_ = db.c.QueryRow(`SELECT content FROM messages WHERE conversation_id = ? ORDER BY created_at DESC LIMIT 1`, c.ID).Scan(&c.Snippet)

		switch c.ConversationType {
		case "group":
			if gName.Valid {
				v := gName.String
				c.GroupName = &v
			}
			if gPhoto.Valid {
				v := gPhoto.String
				c.GroupPhoto = &v
			}
			if gDesc.Valid {
				v := gDesc.String
				c.GroupDescription = &v
			}

		case "private":
			// Trovo l'altro utente
			var otherName, otherPhoto, otherID string
			err := db.c.QueryRow(`
                SELECT u.username, u.profile_photo, u.id
                FROM participants p 
                JOIN users u ON p.user_id = u.id
                WHERE p.conversation_id = ? AND p.user_id != ?`, c.ID, userID).Scan(&otherName, &otherPhoto, &otherID)

			if err == nil {
				c.RecipientUsername = &otherName
				c.UserPhoto = &otherPhoto
				c.RecipientUser = &otherID
			}
		}

		conversations = append(conversations, c)
	}
	return conversations, nil
}

func (db *appdbimpl) GetConversationByID(conversationID string, requestingUserID string) (Conversation, error) {
	var c Conversation
	var grName, grPhoto, grDesc sql.NullString
	var lastMsgAt sql.NullTime

	// 1. Recupero i dati base della conversazione E verifico che l'utente ne faccia parte
	// Se l'utente non è in 'participants' per questa chat, la query restituirà sql.ErrNoRows
	query := `
		SELECT c.id, c.type, c.group_name, c.group_photo, c.group_description, c.last_message_at
		FROM conversations c
		JOIN participants p ON c.id = p.conversation_id
		WHERE c.id = ? AND p.user_id = ?
	`
	err := db.c.QueryRow(query, conversationID, requestingUserID).Scan(
		&c.ID,
		&c.ConversationType,
		&grName,
		&grPhoto,
		&grDesc,
		&lastMsgAt,
	)

	if err == sql.ErrNoRows {
		// Se non trovo righe, o la chat non esiste o l'utente non è membro.
		return Conversation{}, ErrChatNotFound
	}
	if err != nil {
		return Conversation{}, err
	}

	// Gestione Timestamp
	if lastMsgAt.Valid {
		c.DtLastMessage = lastMsgAt.Time
	} else {
		c.DtLastMessage = time.Now()
	}

	// 2. Recupero lo Snippet (ultimo messaggio)
	// Ignoro l'errore --> se non ci sono messaggi, lo snippet resta stringa vuota ""
	_ = db.c.QueryRow(`SELECT content FROM messages WHERE conversation_id = ? ORDER BY created_at DESC LIMIT 1`, conversationID).Scan(&c.Snippet)

	// 3. Logica specifica per TIPO
	switch c.ConversationType {
	case "group":
		// Popolo i campi del gruppo
		if grName.Valid {
			val := grName.String
			c.GroupName = &val
		}
		if grPhoto.Valid {
			val := grPhoto.String
			c.GroupPhoto = &val
		}
		if grDesc.Valid {
			val := grDesc.String
			c.GroupDescription = &val
		}

		// 3a. Recupero TUTTI i membri e gli admin
		rows, err := db.c.Query(`SELECT user_id, is_admin FROM participants WHERE conversation_id = ?`, conversationID)
		if err != nil {
			return Conversation{}, err
		}

		defer func() { _ = rows.Close() }()

		var members []string
		var admins []string

		for rows.Next() {
			var uid string
			var isAdmin bool
			if err := rows.Scan(&uid, &isAdmin); err != nil {
				return Conversation{}, err
			}
			members = append(members, uid)
			if isAdmin {
				admins = append(admins, uid)
			}
		}
		c.Members = &members
		c.Admins = &admins

	case "private":
		// 3b. Popolo i campi della chat privata cercando l'altro utente
		var otherID, otherName, otherPhoto string
		err := db.c.QueryRow(`
            SELECT u.id, u.username, u.profile_photo 
            FROM participants p
            JOIN users u ON p.user_id = u.id
            WHERE p.conversation_id = ? AND p.user_id != ?
        `, conversationID, requestingUserID).Scan(&otherID, &otherName, &otherPhoto)

		switch err {
		case nil:
			c.RecipientUser = &otherID
			c.RecipientUsername = &otherName
			c.UserPhoto = &otherPhoto
		case sql.ErrNoRows:
			// Nessuna riga trovata
		default:
			return Conversation{}, err
		}
	}

	return c, nil
}

func (db *appdbimpl) CreatePrivateChat(userA string, userB string) (Conversation, error) {
	tx, err := db.c.Begin()
	if err != nil {
		return Conversation{}, err
	}
	defer func() { _ = tx.Rollback() }()

	chatID := uuid.New().String()
	now := time.Now()

	if _, err := tx.Exec(`INSERT INTO conversations (id, type, last_message_at) VALUES (?, 'private', ?)`, chatID, now); err != nil {
		return Conversation{}, err
	}
	if _, err := tx.Exec(`INSERT INTO participants (conversation_id, user_id) VALUES (?, ?), (?, ?)`, chatID, userA, chatID, userB); err != nil {
		return Conversation{}, err
	}

	if err := tx.Commit(); err != nil {
		return Conversation{}, err
	}
	return Conversation{ID: chatID, ConversationType: "private"}, nil
}

func (db *appdbimpl) CheckIfPrivateChatExists(userA string, userB string) (string, bool, error) {
	query := `
		SELECT p1.conversation_id 
		FROM participants p1
		JOIN participants p2 ON p1.conversation_id = p2.conversation_id
		JOIN conversations c ON p1.conversation_id = c.id
		WHERE p1.user_id = ? AND p2.user_id = ? AND c.type = 'private'
		LIMIT 1`
	var chatID string
	err := db.c.QueryRow(query, userA, userB).Scan(&chatID)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return chatID, true, nil
}

func (db *appdbimpl) CreateGroup(name string, desc string, photo string, creatorID string, membersIDs []string) (Conversation, error) {
	tx, err := db.c.Begin()
	if err != nil {
		return Conversation{}, err
	}
	defer func() { _ = tx.Rollback() }()

	groupID := uuid.New().String()
	now := time.Now()

	_, err = tx.Exec(`INSERT INTO conversations (id, type, group_name, group_description, group_photo, last_message_at) VALUES (?, 'group', ?, ?, ?, ?)`, groupID, name, desc, photo, now)
	if err != nil {
		return Conversation{}, err
	}

	_, err = tx.Exec(`INSERT INTO participants (conversation_id, user_id, is_admin) VALUES (?, ?, 1)`, groupID, creatorID)
	if err != nil {
		return Conversation{}, err
	}

	for _, memberID := range membersIDs {
		if memberID == creatorID {
			continue
		}
		_, err = tx.Exec(`INSERT INTO participants (conversation_id, user_id, is_admin) VALUES (?, ?, 0)`, groupID, memberID)
		if err != nil {
			return Conversation{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Conversation{}, err
	}
	return Conversation{ID: groupID, ConversationType: "group", GroupName: &name}, nil
}

// DeletePrivateChatForUser "nasconde" la chat rimuovendo l'utente dai partecipanti.
// SE però non rimangono più partecipanti (anche l'altro utente l'ha cancellata),
// allora elimina definitivamente la conversazione e tutti i messaggi dal DB.
func (db *appdbimpl) DeletePrivateChatForUser(conversationID string, userID string) error {
	// 1. Apro una transazione (fondamentale per garantire coerenza)
	tx, err := db.c.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 2. Rimuovo l'utente corrente dalla lista dei partecipanti
	res, err := tx.Exec(`DELETE FROM participants WHERE conversation_id = ? AND user_id = ?`, conversationID, userID)
	if err != nil {
		return err
	}

	// Controllo se ho effettivamente cancellato qualcosa
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		// La chat non esiste o l'utente non ne faceva parte
		return ErrChatNotFound
	}

	// Controllo quanti partecipanti sono rimasti in questa chat
	var remainingParticipants int
	err = tx.QueryRow(`SELECT COUNT(*) FROM participants WHERE conversation_id = ?`, conversationID).Scan(&remainingParticipants)
	if err != nil {
		return err
	}

	// Se non c'è più nessuno (count == 0), significa che anche l'altro utente aveva già cancellato la chat. Posso pulire il DB.
	if remainingParticipants == 0 {
		// Elimino la conversazione.
		// ON DELETE CASCADE cancellerà automaticamente anche le righe nella tabella 'messages' e 'reactions'.
		_, err = tx.Exec(`DELETE FROM conversations WHERE id = ?`, conversationID)
		if err != nil {
			return err
		}
	}

	// Confermo le modifiche
	return tx.Commit()
}

// AddGroupMember aggiunge un utente a un gruppo esistente.
func (db *appdbimpl) AddGroupMember(groupID string, userIDToAdd string) error {
	// Nota: L'handler deve aver già verificato che chi fa la richiesta sia Admin.
	// Qui mi limito a inserire la riga.
	_, err := db.c.Exec(`INSERT INTO participants (conversation_id, user_id, is_admin) VALUES (?, ?, 0)`, groupID, userIDToAdd)
	if err != nil {
		// Gestisco il caso in cui l'utente è già nel gruppo
		return err
	}
	return nil
}

// rimuove un membro dal gruppo.
func (db *appdbimpl) RemoveGroupMember(groupID string, userIDToRemove string) error {
	res, err := db.c.Exec(`DELETE FROM participants WHERE conversation_id = ? AND user_id = ?`, groupID, userIDToRemove)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotMember // L'utente non era nel gruppo
	}
	return nil
}

// promuove o retrocede un utente (isAdmin = true/false).
func (db *appdbimpl) ToggleAdminStatus(groupID string, userID string, isAdmin bool) error {
	res, err := db.c.Exec(`UPDATE participants SET is_admin = ? WHERE conversation_id = ? AND user_id = ?`, isAdmin, groupID, userID)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotMember // Non si può cambiare status a chi non è nel gruppo
	}
	return nil
}

// Aggiorna dinamicamente i campi del gruppo. --> Aggiorna SOLO i campi che non sono stringhe vuote.
func (db *appdbimpl) UpdateGroupInfo(groupID string, name string, desc string, photo string) error {

	if name != "" {
		_, err := db.c.Exec(`UPDATE conversations SET group_name = ? WHERE id = ? AND type = 'group'`, name, groupID)
		if err != nil {
			return err
		}
	}

	if desc != "" {
		_, err := db.c.Exec(`UPDATE conversations SET group_description = ? WHERE id = ? AND type = 'group'`, desc, groupID)
		if err != nil {
			return err
		}
	}

	if photo != "" {
		_, err := db.c.Exec(`UPDATE conversations SET group_photo = ? WHERE id = ? AND type = 'group'`, photo, groupID)
		if err != nil {
			return err
		}
	}

	return nil
}
