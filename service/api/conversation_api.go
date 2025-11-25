package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Youfili/web-chat-app/service/database"
	"github.com/julienschmidt/httprouter"
)

// ---------------------------------------------------------
// LISTA CONVERSAZIONI
// ---------------------------------------------------------

func (rt *_router) getMyConversations(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Check iniziale (funzione definia a fondo pagina)
	if !rt.checkAuth(w, r, ps.ByName("username")) {
		return // l'errore HTTP è gia gestito in checkAuth
	}

	// Recupero l'ID dell'utente (che ho già validato essere quello loggato)
	username := ps.ByName("username")
	user, _ := rt.db.GetUserByUsername(username)

	// 2. Query DB
	conversations, err := rt.db.GetConversations(user.ID)
	if err != nil {
		rt.baseLogger.Errorf("Error fetching conversations: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Evito un possibile valore NULL nel Json, ritorno la struct vuota
	if conversations == nil {
		conversations = []database.Conversation{}
	}

	_ = json.NewEncoder(w).Encode(database.ConversationList{Conversations: conversations}) // Creo una struct di tipo ConversationList e inizializzo il campo 'Conversations' con il valore della variabile "conversations"
}

// ---------------------------------------------------------
// CHAT PRIVATE
// ---------------------------------------------------------

func (rt *_router) newPrivateChat(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	if !rt.checkAuth(w, r, ps.ByName("username")) {
		return
	}
	pathUser, _ := rt.db.GetUserByUsername(ps.ByName("username")) // Mittente

	// Parsing
	var req database.PrivateChatPrototype
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Controllo Destinatario
	recipient, err := rt.db.GetUserByID(req.RecipientUser)
	if err != nil {
		http.Error(w, "Recipient user not found", http.StatusNotFound)
		return
	}

	// Logica: Controllo esistenza + Creazione o Recupero
	var chatID string
	var isNew bool

	existingID, exists, err := rt.db.CheckIfPrivateChatExists(pathUser.ID, recipient.ID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if exists {
		chatID = existingID
		isNew = false
	} else {
		// Creo la chat
		newChat, err := rt.db.CreatePrivateChat(pathUser.ID, recipient.ID)
		if err != nil {
			http.Error(w, "Error creating chat", http.StatusInternalServerError)
			return
		}
		chatID = newChat.ID
		isNew = true
	}

	// INVIO MESSAGGIO INIZIALE (Obbligatorio per come ho scelto di implementare la specifica)
	if req.InitialMessage != "" {
		_, err := rt.db.CreateMessage(database.Message{
			ConversationID: chatID,
			SenderUserID:   pathUser.ID,
			ContentMess:    req.InitialMessage,
			Timestamp:      time.Now().UTC(),
			Forwarded:      false,
		})
		if err != nil {
			rt.baseLogger.Errorf("Chat created but failed to send initial message: %v", err)
			// Non ritorno errore all'utente, la chat è valida
		}
	}

	// Recupero la chat completa per poi restituirla
	fullChat, err := rt.db.GetConversationByID(chatID, pathUser.ID)
	if err != nil {
		http.Error(w, "Error retrieving chat", http.StatusInternalServerError)
		return
	}

	if isNew {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	_ = json.NewEncoder(w).Encode(fullChat)
}

func (rt *_router) getPrivateChatById(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	if !rt.checkAuth(w, r, ps.ByName("username")) {
		return
	}

	user, _ := rt.db.GetUserByUsername(ps.ByName("username"))
	conversationID := ps.ByName("conversationId")

	chat, err := rt.db.GetConversationByID(conversationID, user.ID)
	if err != nil {
		if errors.Is(err, database.ErrChatNotFound) || errors.Is(err, database.ErrUserNotMember) {
			http.Error(w, "Conversation not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Verifica che la chat sia davvero privata (controllo di sicurezza aggiuntivo)
	if chat.ConversationType != "private" {
		http.Error(w, "This is not a private chat", http.StatusBadRequest) // O 404
		return
	}

	_ = json.NewEncoder(w).Encode(chat)
}

func (rt *_router) deletePrivateChat(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if !rt.checkAuth(w, r, ps.ByName("username")) {
		return
	}
	user, _ := rt.db.GetUserByUsername(ps.ByName("username"))

	err := rt.db.DeletePrivateChatForUser(ps.ByName("conversationId"), user.ID)
	if err != nil {
		if errors.Is(err, database.ErrChatNotFound) {
			http.Error(w, "Conversation not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------
// GRUPPI
// ---------------------------------------------------------

func (rt *_router) createNewGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	if !rt.checkAuth(w, r, ps.ByName("username")) {
		return
	}

	creator, _ := rt.db.GetUserByUsername(ps.ByName("username")) // L'utente loggato in sessione è ovviamente il creatore del gruppo

	var req database.GroupChatPrototype
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Aggiungo il creatore del gruppo (utente attualmente in sessione) alla lista membri se non c'è già
	members := req.Members
	// NOTA: La logica DB "CreateGroup" gestisce già l'aggiunta del creatore come admin,
	// qui passo la lista totale, il DB filtrerà i duplicati)

	newGroup, err := rt.db.CreateGroup(req.GroupName, req.GroupDescription, "", creator.ID, members)
	if err != nil {
		rt.baseLogger.Errorf("Error creating group: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Recupero l'oggetto completo (con tutti i membri popolati), per poi restituirlo come risposto
	fullGroup, _ := rt.db.GetConversationByID(newGroup.ID, creator.ID)

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(fullGroup)
}

func (rt *_router) getGroupChat(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	if !rt.checkAuth(w, r, ps.ByName("username")) {
		return
	}

	user, _ := rt.db.GetUserByUsername(ps.ByName("username"))

	chat, err := rt.db.GetConversationByID(ps.ByName("conversationId"), user.ID)
	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}
	if chat.ConversationType != "group" {
		http.Error(w, "This is not a group chat", http.StatusBadRequest)
		return
	}
	_ = json.NewEncoder(w).Encode(chat)
}

func (rt *_router) addToGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	pathUsername := ps.ByName("username")
	if !rt.checkAuth(w, r, pathUsername) {
		return
	}

	// Parsing del Body
	var req database.AddMembersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Recupero l'utente che sta facendo la richiesta (per avere il suo ID)
	requesterUser, err := rt.db.GetUserByUsername(pathUsername)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	groupID := ps.ByName("conversationId")

	// ------------------------------------------------------------------------------------------
	// Controllo se l'utente che fa la richiesta è Admin del gruppo a cui vuole aggiungere membri
	// ------------------------------------------------------------------------------------------
	if !rt.isUserAdminOfGroup(w, groupID, requesterUser.ID) {
		return // L'helper ha già scritto l'errore HTTP 403 o 404
	}

	for _, userIDToAdd := range req.UsersIDsToAdd {
		// Controllo se l'utente da aggiungere esiste davvero nel DB --> Non voglio aggiungere un ID fantasma alla tabella participants!
		if _, err := rt.db.GetUserByID(userIDToAdd); err != nil {
			// Se un utente non esiste, posso ignorarlo o fermarmi.
			http.Error(w, "One of the users to add does not exist: "+userIDToAdd, http.StatusBadRequest) // Restituisco Bad Request
			return
		}

		// Aggiungo al gruppo l'utente scelto per essere aggiunto
		err := rt.db.AddGroupMember(groupID, userIDToAdd)
		if err != nil {
			// Se l'utente è già nel gruppo (errore constraint), lo ignoro, Se è un errore più grave, loggo.
			rt.baseLogger.Errorf("Failed to add user %s to group %s: %v", userIDToAdd, groupID, err)
		}
	}

	// Ritorno il gruppo aggiornato
	updatedGroup, err := rt.db.GetConversationByID(groupID, requesterUser.ID)
	if err != nil {
		http.Error(w, "Error retrieving updated group", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(updatedGroup)
}

func (rt *_router) leaveGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if !rt.checkAuth(w, r, ps.ByName("username")) {
		return
	}
	user, _ := rt.db.GetUserByUsername(ps.ByName("username"))

	err := rt.db.RemoveGroupMember(ps.ByName("conversationId"), user.ID)
	if err != nil {
		http.Error(w, "Error leaving group", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) removeMember(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	pathUsername := ps.ByName("username")
	if !rt.checkAuth(w, r, pathUsername) {
		return
	}

	// Recupero l'utente che fa la richiesta (per avere il suo ID)
	requesterUser, err := rt.db.GetUserByUsername(pathUsername)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	groupID := ps.ByName("conversationId")
	targetUserID := ps.ByName("userId")

	// ---------------------------------------------------------
	// Faccio nuovamente il controllo per vedere se l'utente chiamante è un Admin (come fatto per la funzione addToGroup)
	// ---------------------------------------------------------

	if !rt.isUserAdminOfGroup(w, groupID, requesterUser.ID) {
		return // Se non è admin o il gruppo non esiste, esce qui.
	}

	// Devo recuperare di nuovo il gruppo per vedere la lista Admin e controllare il target
	// ----------------------------------------------------------------
	currentGroup, err := rt.db.GetConversationByID(groupID, requesterUser.ID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// VERIFICO CHE IL BERSAGLIO NON SIA ADMIN
	// (Ho scelto di implementarlo in questo modo, un Admin può rimuovere un altro admin, solo se prima gli rimuove lo stato di admin)
	targetIsAdmin := false
	if currentGroup.Admins != nil {
		for _, adminID := range *currentGroup.Admins {
			if adminID == targetUserID {
				targetIsAdmin = true
				break
			}
		}
	}
	if targetIsAdmin {
		http.Error(w, "Forbidden: Cannot remove an admin. Demote them first.", http.StatusBadRequest)
		return
	}

	// Rimuovo il membro dal gruppo
	err = rt.db.RemoveGroupMember(groupID, targetUserID)
	if err != nil {
		// Se l'utente non era nel gruppo
		if errors.Is(err, database.ErrUserNotMember) {
			http.Error(w, "User is not a member of this group", http.StatusNotFound)
			return
		}

		rt.baseLogger.Errorf("Error removing member %s from group %s: %v", targetUserID, groupID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) makeAdmin(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	pathUsername := ps.ByName("username")
	if !rt.checkAuth(w, r, pathUsername) {
		return
	}

	// Recupero ID richiedente
	requesterUser, _ := rt.db.GetUserByUsername(pathUsername)
	conversationID := ps.ByName("conversationId")

	// Controllo se il Richiedente è un Admin del gruppo)
	if !rt.isUserAdminOfGroup(w, conversationID, requesterUser.ID) {
		return
	}

	// Decode Body
	var req database.AdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Rendo l'utente scelto Admin
	err := rt.db.ToggleAdminStatus(conversationID, req.UserIDToPromote, true)
	if err != nil {
		if errors.Is(err, database.ErrUserNotMember) {
			http.Error(w, "User to promote is not a member of the group", http.StatusBadRequest)
			return
		}
		rt.baseLogger.Errorf("Error promoting user: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Ritorno il Gruppo modificato
	updatedGroup, _ := rt.db.GetConversationByID(conversationID, requesterUser.ID)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(updatedGroup)
}

func (rt *_router) removeAdminStatus(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	pathUsername := ps.ByName("username")
	if !rt.checkAuth(w, r, pathUsername) {
		return
	}

	requesterUser, _ := rt.db.GetUserByUsername(pathUsername)
	conversationID := ps.ByName("conversationId")
	targetUserID := ps.ByName("userId")

	// Contollo che Richiedente sia un Admin del gruppo
	if !rt.isUserAdminOfGroup(w, conversationID, requesterUser.ID) {
		return
	}

	// Rimuovo lo status di Admin dal gruppo
	err := rt.db.ToggleAdminStatus(conversationID, targetUserID, false)
	if err != nil {
		if errors.Is(err, database.ErrUserNotMember) {
			http.Error(w, "User is not a member of the group", http.StatusBadRequest)
			return
		}
		rt.baseLogger.Errorf("Error demoting user: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Ritorno il gruppo aggiornato (dopo aver "declassato" l'utente)
	updatedGroup, _ := rt.db.GetConversationByID(conversationID, requesterUser.ID)
	_ = json.NewEncoder(w).Encode(updatedGroup)
}

// Wrapper functions per il router --> gestisco la logia in "updateGroupHelper" in modo centralizzato
func (rt *_router) setGroupName(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	rt.updateGroupHelper(w, r, ps, "name")
}
func (rt *_router) setGroupDescription(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	rt.updateGroupHelper(w, r, ps, "desc")
}
func (rt *_router) setGroupPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	rt.updateGroupHelper(w, r, ps, "photo")
}

// updateGroupHelper centralizza la logica di aggiornamento
func (rt *_router) updateGroupHelper(w http.ResponseWriter, r *http.Request, ps httprouter.Params, field string) {
	w.Header().Set("Content-Type", "application/json")

	pathUsername := ps.ByName("username")
	if !rt.checkAuth(w, r, pathUsername) {
		return
	}

	requesterUser, _ := rt.db.GetUserByUsername(pathUsername)
	conversationID := ps.ByName("conversationId")

	// Solo ADMIN possono cambiare info gruppo)
	if !rt.isUserAdminOfGroup(w, conversationID, requesterUser.ID) {
		return
	}

	// Decode JSON in base al campo
	var name, desc, photo string

	dec := json.NewDecoder(r.Body)

	switch field {
	case "name":
		var req database.GroupNameUpdate
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "Invalid JSON for Group Name", http.StatusBadRequest)
			return
		}
		// Validazione lunghezza (La validazione si basa su parametri che ho scelto in fase di progettazione)
		if len(req.GroupName) < 1 || len(req.GroupName) > 200 {
			http.Error(w, "Group name must be between 1 and 200 chars", http.StatusBadRequest)
			return
		}
		name = req.GroupName

	case "desc":
		var req database.GroupDescriptionUpdate
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "Invalid JSON for Group Description", http.StatusBadRequest)
			return
		}
		desc = req.GroupDescription

	case "photo":
		var req database.GroupPhotoUpdate
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "Invalid JSON for Group Photo", http.StatusBadRequest)
			return
		}
		// Validazione URL
		if len(req.GroupPhoto) > 0 && !strings.HasPrefix(req.GroupPhoto, "http") {
			http.Error(w, "Invalid Photo URL", http.StatusBadRequest)
			return
		}
		photo = req.GroupPhoto

	default:
		// Se il campo non è uno di quelli previsti, restituisco un errore.
		http.Error(w, "Unknown field to update", http.StatusBadRequest)
		return
	}

	// Chiamo il Database
	err := rt.db.UpdateGroupInfo(conversationID, name, desc, photo)
	if err != nil {
		rt.baseLogger.Errorf("Error updating group info (%s): %v", field, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Ritorno il gruppo Modificato
	updatedGroup, _ := rt.db.GetConversationByID(conversationID, requesterUser.ID)
	_ = json.NewEncoder(w).Encode(updatedGroup)
}

// ---------------------------------------------------------
// Ho realizzato questa funzione di autenticazione per non dover ogni volta essere prolisso nel codice di ogni funzione
// ---------------------------------------------------------
// checkAuth verifica che l'utente nell'URL corrisponda al token Authorization
func (rt *_router) checkAuth(w http.ResponseWriter, r *http.Request, pathUsername string) bool {
	authHeader := r.Header.Get("Authorization")
	tokenID := strings.TrimPrefix(authHeader, "Bearer ")

	if tokenID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}

	// Recuperiamo l'utente del path per vedere il suo ID
	pathUser, err := rt.db.GetUserByUsername(pathUsername)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return false
	}

	if pathUser.ID != tokenID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return false
	}
	return true
}

// isUserAdminOfGroup verifica se l'utente è admin del gruppo.
// Restituisce true se è admin. Se false, gestisce già l'errore HTTP.
func (rt *_router) isUserAdminOfGroup(w http.ResponseWriter, conversationID string, userID string) bool {
	// 1. Recupero la chat
	chat, err := rt.db.GetConversationByID(conversationID, userID)
	if err != nil {
		// Se non la trova o l'utente non è membro
		http.Error(w, "Group not found or access denied", http.StatusNotFound)
		return false
	}

	// 2. Verifica tipo
	if chat.ConversationType != "group" {
		http.Error(w, "Operation allowed only on groups", http.StatusBadRequest)
		return false
	}

	// 3. Verifica Admin
	isAdmin := false
	if chat.Admins != nil {
		for _, adminID := range *chat.Admins {
			if adminID == userID {
				isAdmin = true
				break
			}
		}
	}

	if !isAdmin {
		http.Error(w, "Forbidden: Only admins can perform this action", http.StatusForbidden)
		return false
	}

	return true
}
