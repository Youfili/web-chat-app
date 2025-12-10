package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Youfili/web-chat-app/service/database"
	"github.com/julienschmidt/httprouter"
)

// sendMessage invia un nuovo messaggio
func (rt *_router) sendMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	pathUsername := ps.ByName("username")
	if !rt.checkAuth(w, r, pathUsername) {
		return
	}

	// Decode
	var req database.NewMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validazione
	if req.ContentMess == "" {
		http.Error(w, "Message content cannot be empty", http.StatusBadRequest)
		return
	}

	requesterUser, _ := rt.db.GetUserByUsername(pathUsername)
	conversationID := ps.ByName("conversationId")

	// Verifico esistenza chat e se l'utente è o non è membro)
	if _, err := rt.db.GetConversationByID(conversationID, requesterUser.ID); err != nil {
		http.Error(w, "Conversation not found or access denied", http.StatusForbidden)
		return
	}

	// Creazione Messaggio
	msg := database.Message{
		ConversationID: conversationID,
		SenderUserID:   requesterUser.ID,
		SenderUsername: requesterUser.Username, // Popolo subito lo username, ho avuto problemi nel frontend nell'invio del messaggio
		ContentMess:    req.ContentMess,
		// Timestamp e ID vengono messi dal DB (vedere message_db.go)
		Forwarded: false,
	}

	// Salvataggio nel DB
	createdMsg, err := rt.db.CreateMessage(msg)
	if err != nil {
		rt.baseLogger.Errorf("Error sending message: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Mi assicuro che lo username ci sia nel ritorno (dopo aver avuto problemi con l'invio del messaggio)
	createdMsg.SenderUsername = requesterUser.Username

	// Ritorno il nuovo messaggio creato
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(createdMsg)
}

// getConversation recupera la lista messaggi con paginazione
func (rt *_router) getConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	pathUsername := ps.ByName("username")
	if !rt.checkAuth(w, r, pathUsername) {
		return
	}

	requesterUser, _ := rt.db.GetUserByUsername(pathUsername)
	conversationID := ps.ByName("conversationId")

	if _, err := rt.db.GetConversationByID(conversationID, requesterUser.ID); err != nil {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// --- GESTIONE PAGINAZIONE ---
	// limit (default metto 20)
	limit := 20
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// before (timestamp cursore)
	before := time.Now().UTC() // Default: messaggi più recenti di ADESSO
	if beforeParam := r.URL.Query().Get("before"); beforeParam != "" {
		if t, err := time.Parse(time.RFC3339, beforeParam); err == nil {
			before = t
		}
	}

	// Query DB
	messages, err := rt.db.GetMessages(conversationID, limit, before)
	if err != nil {
		rt.baseLogger.Errorf("Error fetching messages: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if messages == nil {
		messages = []database.Message{}
	}

	_ = json.NewEncoder(w).Encode(database.MessageList{Messages: messages})
}

// forwardMessage gestisce l'inoltro
func (rt *_router) forwardMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	pathUsername := ps.ByName("username")
	if !rt.checkAuth(w, r, pathUsername) {
		return
	}
	requesterUser, _ := rt.db.GetUserByUsername(pathUsername)

	// Decode Body
	var req database.ForwardMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validazione Input (Deve esserci O ConversationID O UserID)
	if req.TargetConversationID == nil && req.TargetUserID == nil {
		http.Error(w, "Must provide either targetConversationId or targetUserId", http.StatusBadRequest)
		return
	}

	// RECUPERO MESSAGGIO ORIGINALE
	originalMessageID := ps.ByName("messageId")
	originalMsg, err := rt.db.GetMessageByID(originalMessageID)
	if err != nil {
		if errors.Is(err, database.ErrMessageNotFound) {
			http.Error(w, "Original message not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	//
	// Controllo: L'utente non può inoltrare un messaggio da una chat di cui non fa parte!
	if _, err := rt.db.GetConversationByID(originalMsg.ConversationID, requesterUser.ID); err != nil {
		http.Error(w, "Forbidden: You cannot forward a message from a chat you are not in", http.StatusForbidden)
		return
	}

	// DETERMINO LA CHAT DI DESTINAZIONE
	var targetChatID string

	if req.TargetConversationID != nil {
		// CASO A: ID Conversazione fornito
		targetChatID = *req.TargetConversationID

		// Verifico l'esistenza e accesso alla chat di destinazione
		if _, err := rt.db.GetConversationByID(targetChatID, requesterUser.ID); err != nil {
			http.Error(w, "Target conversation not found or access denied", http.StatusBadRequest)
			return
		}

	} else {
		// CASO B: ID Utente fornito
		targetUserID := *req.TargetUserID

		// Non puoi inoltrare a te stesso in una chat privata (scelta implementativa personale)
		if targetUserID == requesterUser.ID {
			http.Error(w, "Cannot forward message to yourself as a new chat", http.StatusBadRequest)
			return
		}

		// Controllo se esiste già una chat privata
		existingChatID, exists, err := rt.db.CheckIfPrivateChatExists(requesterUser.ID, targetUserID)
		if err != nil {
			http.Error(w, "DB Error", http.StatusInternalServerError)
			return
		}

		if exists {
			targetChatID = existingChatID
		} else {
			// Se non esiste, la creo al volo
			newChat, err := rt.db.CreatePrivateChat(requesterUser.ID, targetUserID)
			if err != nil {
				http.Error(w, "Error creating new chat for forward", http.StatusInternalServerError)
				return
			}
			targetChatID = newChat.ID
		}
	}

	// CREAZIONE NUOVO MESSAGGIO
	// Copio il contenuto, resettiamo ID e Timestamp, settiamo Forwarded = true
	newMsg := database.Message{
		ConversationID: targetChatID,
		SenderUserID:   requesterUser.ID,
		ContentMess:    originalMsg.ContentMess, // Contenuto copiato dal DB
		Forwarded:      true,                    // Flag Forwarded impostato a true, proprio perché questo messaggio è stato inoltrato
		// Ricordo a me stesso che: Timestamp e ID verranno generati da rt.db.CreateMessage
	}

	createdMsg, err := rt.db.CreateMessage(newMsg)
	if err != nil {
		rt.baseLogger.Errorf("Error saving forwarded message: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Risposta
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(createdMsg)
}

func (rt *_router) markMessagesAsRead(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	pathUsername := ps.ByName("username")
	if !rt.checkAuth(w, r, pathUsername) {
		return
	}

	requesterUser, _ := rt.db.GetUserByUsername(pathUsername)
	conversationID := ps.ByName("conversationId")

	var req database.ReadStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	err := rt.db.MarkConversationAsRead(conversationID, requesterUser.ID, req.LastReadMessageID)
	if err != nil {
		http.Error(w, "Error updating read status", http.StatusInternalServerError)
		return
	}

	// Risposta JSON successo
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"success": true}`))
}

func (rt *_router) editMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	pathUsername := ps.ByName("username")
	if !rt.checkAuth(w, r, pathUsername) {
		return
	}
	requesterUser, _ := rt.db.GetUserByUsername(pathUsername)

	// Decode Body
	var req database.MessageUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validazione
	if req.ContentMess == "" {
		http.Error(w, "Message content cannot be empty", http.StatusBadRequest)
		return
	}

	messageID := ps.ByName("messageId")

	// Recupero il messaggio originale PRIMA di modificarlo
	originalMsg, err := rt.db.GetMessageByID(messageID)
	if err != nil {
		if errors.Is(err, database.ErrMessageNotFound) {
			http.Error(w, "Message not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Verifico che l'utente che fa la richiesta sia il MITTENTE
	if originalMsg.SenderUserID != requesterUser.ID {
		http.Error(w, "Forbidden: You can only edit your own messages", http.StatusForbidden)
		return
	}

	// 4. Esecuzione Update
	updatedMsg, err := rt.db.EditMessage(messageID, req.ContentMess)
	if err != nil {
		http.Error(w, "Error updating message", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(updatedMsg)
}

func (rt *_router) deleteMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	pathUsername := ps.ByName("username")
	if !rt.checkAuth(w, r, pathUsername) {
		return
	}
	requesterUser, _ := rt.db.GetUserByUsername(pathUsername)

	messageID := ps.ByName("messageId")

	// Recupero il messaggio per vedere di chi è (chi lo ha scritto)
	originalMsg, err := rt.db.GetMessageByID(messageID)
	if err != nil {
		if errors.Is(err, database.ErrMessageNotFound) {
			http.Error(w, "Message not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// SOLO il Mittente può Cancellare il messaggio
	if originalMsg.SenderUserID != requesterUser.ID {
		http.Error(w, "Forbidden: You can only delete your own messages", http.StatusForbidden)
		return
	}

	// 3. Esecuzione Delete
	err = rt.db.DeleteMessage(messageID)
	if err != nil {
		http.Error(w, "Error deleting message", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
