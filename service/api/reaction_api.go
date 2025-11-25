package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Youfili/web-chat-app/service/database"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

// commentMessage aggiunge una reazione a un messaggio
func (rt *_router) commentMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Check Auth
	pathUsername := ps.ByName("username")
	if !rt.checkAuth(w, r, pathUsername) {
		return
	}
	requesterUser, _ := rt.db.GetUserByUsername(pathUsername)

	// 2. Decode Body
	var req database.NewReactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validazione Emoji
	if req.Emoji == "" {
		http.Error(w, "Emoji cannot be empty", http.StatusBadRequest)
		return
	}

	conversationID := ps.ByName("conversationId")
	messageID := ps.ByName("messageId")

	// 3. SICUREZZA: L'utente deve essere membro della chat
	if _, err := rt.db.GetConversationByID(conversationID, requesterUser.ID); err != nil {
		http.Error(w, "Conversation not found or access denied", http.StatusForbidden)
		return
	}

	// 4. SICUREZZA: Il messaggio deve appartenere a QUELLA chat
	targetMsg, err := rt.db.GetMessageByID(messageID)
	if err != nil {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}
	if targetMsg.ConversationID != conversationID {
		http.Error(w, "Message does not belong to this conversation", http.StatusBadRequest)
		return
	}

	// 5. Costruzione oggetto Reazione
	reaction := database.Reaction{
		ReactionID:     uuid.New().String(),
		Emoji:          req.Emoji,
		Timestamp:      time.Now().UTC(),
		MessToReactID:  messageID,
		SenderUserID:   requesterUser.ID,
		SenderUsername: requesterUser.Username,
	}

	// 6. Salvataggio nel DB
	err = rt.db.AddReaction(reaction)
	if err != nil {
		if errors.Is(err, database.ErrDuplicateReaction) {
			http.Error(w, "You already reacted to this message", http.StatusBadRequest)
			return
		}
		rt.baseLogger.Errorf("Error adding reaction: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 7. Risposta (201 Created)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(reaction)
}

// uncommentMessage rimuove la reazione dell'utente corrente
func (rt *_router) uncommentMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// 1. Check Auth
	pathUsername := ps.ByName("username")
	if !rt.checkAuth(w, r, pathUsername) {
		return
	}
	requesterUser, _ := rt.db.GetUserByUsername(pathUsername)

	conversationID := ps.ByName("conversationId")
	messageID := ps.ByName("messageId")

	// 2. Verifica accesso Chat
	if _, err := rt.db.GetConversationByID(conversationID, requesterUser.ID); err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// 3. Rimozione dal DB
	// Nota: in questo caso NON serve controllare se il messaggio esiste, se non c'è reazione la delete non fa nulla o ritorna errore
	err := rt.db.RemoveReaction(messageID, requesterUser.ID)
	if err != nil {
		http.Error(w, "Error removing reaction (or reaction not found)", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// getAllMessageReactions recupera la lista delle reazioni
func (rt *_router) getAllMessageReactions(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Check Auth
	pathUsername := ps.ByName("username")
	if !rt.checkAuth(w, r, pathUsername) {
		return
	}
	requesterUser, _ := rt.db.GetUserByUsername(pathUsername)

	conversationID := ps.ByName("conversationId")
	messageID := ps.ByName("messageId")

	// 2. Verifica accesso Chat
	if _, err := rt.db.GetConversationByID(conversationID, requesterUser.ID); err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// 3. Recupero dal DB
	reactions, err := rt.db.GetReactions(messageID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Come da prassi, evito null nel JSON
	if reactions == nil {
		reactions = []database.Reaction{}
	}

	_ = json.NewEncoder(w).Encode(database.ReactionList{Reactions: reactions})
}
