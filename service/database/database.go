/*
Package database is the middleware between the app database and the code. All data (de)serialization (save/load) from a
persistent database are handled here. Database specific logic should never escape this package.

To use this package you need to apply migrations to the database if needed/wanted, connect to it (using the database
data source name from config), and then initialize an instance of AppDatabase from the DB connection.

For example, this code adds a parameter in `webapi` executable for the database data source name (add it to the
main.WebAPIConfiguration structure):

	DB struct {
		Filename string `conf:""`
	}

This is an example on how to migrate the DB and connect to it:

	// Start Database
	logger.Println("initializing database support")
	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		logger.WithError(err).Error("error opening SQLite DB")
		return fmt.Errorf("opening SQLite: %w", err)
	}
	defer func() {
		logger.Debug("database stopping")
		_ = db.Close()
	}()

Then you can initialize the AppDatabase and pass it to the api package.
*/
package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// L'handler controllerà questi errori per decidere se mandare 404, 403 o 500.
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrChatNotFound      = errors.New("conversation not found")
	ErrMessageNotFound   = errors.New("message not found")
	ErrReactionNotFound  = errors.New("reaction not found")
	ErrUserNotMember     = errors.New("user is not a member of this conversation")
	ErrUserNotAdmin      = errors.New("user is not an admin of this group")
	ErrUsernameTaken     = errors.New("username already taken")
	ErrSelfOperation     = errors.New("operation on self not allowed") // Es. chat privata con se stessi o auto-rimozione admin errata
	ErrDuplicateReaction = errors.New("user already reacted to this message")
	ErrPrivateChatExists = errors.New("private chat already exists") // Utile per il check se una chat privata già esiste, quando l'user ne vuole creare una nuova con uno specifico utente (dimenticandosi che già ne aveva una)
	ErrConstraint        = errors.New("constraint violated")
)

// AppDatabase is the high level interface for the DB
type AppDatabase interface {
	GetName() (string, error)

	// ------------------------------------------------
	// USER MANAGEMENT
	// ------------------------------------------------

	// CreateUser crea un nuovo utente. Ritorna l'utente creato (con ID UUID).
	CreateUser(u User) (User, error)

	// GetUserByID ritorna il profilo utente dato l'UUID.
	GetUserByID(id string) (User, error)

	// GetUserByUsername ritorna il profilo utente dato lo username (utile per login e search).
	GetUserByUsername(username string) (User, error)

	// SearchUsers cerca utenti tramite "pattern matching" (LIKE %query%) sullo username.
	// Esclude l'utente che fa la richiesta (id dello user loggato in sessione) dai risultati se necessario.
	SearchUsers(query string) ([]User, error)

	// UpdateUsername aggiorna lo username. Deve ritornare ErrUsernameTaken se esiste già.
	UpdateUsername(userID string, newUsername string) (User, error)

	// SetUserPhoto aggiorna la foto profilo.
	SetUserPhoto(userID string, photoURL string) error

	// SetUserStatus aggiorna lo stato testuale dell'utente.
	SetUserStatus(userID string, status string) error

	// ------------------------------------------------
	// CONVERSATION LIST & GENERAL
	// ------------------------------------------------

	// GetConversations ritorna la lista delle chat (Private + Gruppi) per un utente.
	// La struct api.Conversation contiene campi nil/non-nil a seconda del tipo.
	// Deve essere ordinata per data dell'ultimo messaggio (desc).
	GetConversations(userID string) ([]Conversation, error)

	// GetConversationByID ritorna una specifica conversazione.
	// Deve popolare i campi corretti (es. GroupName se gruppo, Recipient se privata).
	GetConversationByID(conversationID string, requestingUserID string) (Conversation, error)

	// ------------------------------------------------
	// PRIVATE CHATS
	// ------------------------------------------------

	// CreatePrivateChat crea una chat tra due utenti.
	// Se la chat esiste già, dovrebbe ritornare quella esistente (o un errore specifico gestito dall'handler).
	CreatePrivateChat(userA string, userB string) (Conversation, error)

	// CheckIfPrivateChatExists controlla se esiste una chat tra due utenti e ne ritorna l'ID.
	// Fondamentale per la logica di "Forward" intelligente.
	CheckIfPrivateChatExists(userA string, userB string) (string, bool, error)

	// DeletePrivateChatForUser nasconde/elimina la chat per l'utente richiedente.
	DeletePrivateChatForUser(conversationID string, userID string) error

	// ------------------------------------------------
	// GROUP CHATS
	// ------------------------------------------------

	// CreateGroup crea un nuovo gruppo.
	// membersIDs è la lista iniziale dei partecipanti (incluso il creatore).
	CreateGroup(name string, desc string, photo string, creatorID string, membersIDs []string) (Conversation, error)

	// AddGroupMember aggiunge un utente al gruppo.
	AddGroupMember(groupID string, userIDToAdd string) error

	// RemoveGroupMember rimuove un utente dal gruppo.
	RemoveGroupMember(groupID string, userIDToRemove string) error

	// ToggleAdminStatus imposta o rimuove lo stato di admin per un membro.
	// isAdmin = true (promuovi), isAdmin = false (retrocedi).
	ToggleAdminStatus(groupID string, userID string, isAdmin bool) error

	// UpdateGroupInfo aggiorna nome, descrizione O foto (a seconda di cosa non è stringa vuota).
	UpdateGroupInfo(groupID string, name string, desc string, photo string) error

	// ------------------------------------------------
	// MESSAGES
	// ------------------------------------------------

	// CreateMessage salva un nuovo messaggio nel DB.
	CreateMessage(msg Message) (Message, error)

	// GetMessages recupera la cronologia.
	// 'before': timestamp per paginazione (messaggi più vecchi di...).
	// 'limit': numero messaggi.
	GetMessages(conversationID string, limit int, before time.Time) ([]Message, error)

	// Return Message by ID
	GetMessageByID(messageID string) (Message, error)

	// EditMessage modifica il contenuto testuale.
	EditMessage(messageID string, newContent string) (Message, error)

	// DeleteMessage elimina un messaggio.
	DeleteMessage(messageID string) error

	// MarkConversationAsRead aggiorna lo stato 'read' per i messaggi in una chat.
	// userReaderID: chi sta leggendo.
	// lastMessageID: fino a quale messaggio segnare come letto.
	MarkConversationAsRead(conversationID string, userReaderID string, lastMessageID string) error

	// ------------------------------------------------
	// REACTIONS
	// ------------------------------------------------

	// AddReaction aggiunge una reazione.
	AddReaction(reaction Reaction) error

	// RemoveReaction rimuove la reazione di un utente a un messaggio.
	RemoveReaction(messageID string, userID string) error

	// GetReactions recupera tutte le reazioni di un messaggio.
	GetReactions(messageID string) ([]Reaction, error)

	Ping() error
}

type appdbimpl struct {
	c *sql.DB
}

// New returns a new instance of AppDatabase based on the SQLite connection `db`.
// `db` is required - an error will be returned if `db` is `nil`.
func New(db *sql.DB) (AppDatabase, error) {
	if db == nil {
		return nil, errors.New("database is required when building a AppDatabase")
	}

	/// --- CREAZIONE TABELLE (Inizializzazione) ---

	// 1. Utenti
	if err := createTableUsers(db); err != nil {
		return nil, fmt.Errorf("error creating users table: %w", err)
	}

	// 2. Conversazioni (tabella ibrida - Private/Group)
	if err := createTableConversations(db); err != nil {
		return nil, fmt.Errorf("error creating conversations table: %w", err)
	}

	// 3. Partecipanti (Collega utenti e conversazioni)
	if err := createTableParticipants(db); err != nil {
		return nil, fmt.Errorf("error creating participants table: %w", err)
	}

	// 4. Messaggi
	if err := createTableMessages(db); err != nil {
		return nil, fmt.Errorf("error creating messages table: %w", err)
	}

	// 5. Reazioni
	if err := createTableReactions(db); err != nil {
		return nil, fmt.Errorf("error creating reactions table: %w", err)
	}

	return &appdbimpl{
		c: db,
	}, nil
}

func (db *appdbimpl) Ping() error {
	return db.c.Ping()
}

func (db *appdbimpl) GetName() (string, error) {
	// Restituisci il nome del progetto o quello che richiede la specifica
	return "Wasatext", nil
}
