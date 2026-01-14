package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Youfili/web-chat-app/service/database"
	"github.com/julienschmidt/httprouter"
)

// ---------------------------------------------------------
// LETTURA DATI (GET)
// ---------------------------------------------------------

// myProfileDetails restituisce i dettagli dell'utente loggato (ossia quello che per scelta implementativa ho specificato nel path)
func (rt *_router) myProfileDetails(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	// Recupero lo username dal parametro nel path
	username := ps.ByName("username")

	// Recupero l'utente dal DB usando lo username nel path appena recuperato
	user, err := rt.db.GetUserByUsername(username)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		rt.baseLogger.Errorf("Error getting profile: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(user)
}

// getUserProfile restituisce il profilo di un ALTRO utente (specificato per username con il parametro usernameSearched)
func (rt *_router) getUserProfile(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	// Recupero il parametro "usernameSearched" nel path
	targetUsername := ps.ByName("usernameSearched")

	user, err := rt.db.GetUserByUsername(targetUsername)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(user)
}

// searchUser gestisce la ricerca fuzzy degli utenti
func (rt *_router) searchUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	// Leggo il parametro dalla Query String (?usernameSearched=...)
	queryParam := r.URL.Query().Get("usernameSearched")

	if len(queryParam) == 0 { // Username cercato, NON trovato
		// Se la query è vuota, restituiamo una lista vuota invece di errore (mia scelta implementativa)
		_ = json.NewEncoder(w).Encode(database.UserListResponse{Users: []database.User{}})
		return
	}

	users, err := rt.db.SearchUsers(queryParam) // ricerco l'username parametro della query nel path
	if err != nil {
		rt.baseLogger.Errorf("Error searching users: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Per il JSON sempre ritornare lo struct vuoto, piuttosto che NULL
	if users == nil {
		users = []database.User{}
	}

	_ = json.NewEncoder(w).Encode(database.UserListResponse{Users: users})
}

// ---------------------------------------------------------
// MODIFICA DATI (PUT/PATCH)
// ---------------------------------------------------------

// setMyUserName cambia lo username
func (rt *_router) setMyUserName(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	pathUsername := ps.ByName("username") // Estraggo l'username (parametro) dal path

	authHeader := r.Header.Get("Authorization")                   // Estraggo l'ID dell'utente
	requestingUserID := strings.TrimPrefix(authHeader, "Bearer ") // Rimuovo il prefisso, cosi ho solo l'ID

	// 1. Devo verificare che l'utente che fa la richiesta sia proprietario dell'account 'pathUsername'
	// Quindi Recupero l'ID dell'utente nel path
	targetUser, err := rt.db.GetUserByUsername(pathUsername)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Confronto i due ID ottenuti
	if targetUser.ID != requestingUserID {
		http.Error(w, "Forbidden: You can only modify your own profile", http.StatusForbidden)
		return
	}

	// 2. Parsing Body
	var req database.UpdateUsernameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Pulisco la stringa dagli spazi iniziali e finali
	req.NewUsername = strings.TrimSpace(req.NewUsername)

	// Se dopo il trim l'username è vuoto, blocco tutto
	if req.NewUsername == "" {
		http.Error(w, "Username cannot be empty or just spaces", http.StatusBadRequest)
		return
	}

	// 3. Validazione
	if !isValidUsername(req.NewUsername) {
		http.Error(w, "Invalid username format", http.StatusBadRequest)
		return
	}

	// 4. DB Update
	updatedUser, err := rt.db.UpdateUsername(targetUser.ID, req.NewUsername)
	if err != nil {
		if errors.Is(err, database.ErrUsernameTaken) {
			http.Error(w, "Username already taken", http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 5. Ritorna l'utente aggiornato
	_ = json.NewEncoder(w).Encode(updatedUser)
}

// setMyPhoto aggiorna la foto profilo
func (rt *_router) setMyPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	pathUsername := ps.ByName("username")
	authHeader := r.Header.Get("Authorization")
	requestingUserID := strings.TrimPrefix(authHeader, "Bearer ")

	// Verifico che l'utente che fa la richiesta sia l'utente loggato in sessione
	targetUser, err := rt.db.GetUserByUsername(pathUsername)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if targetUser.ID != requestingUserID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req database.UpdatePhotoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest) // 400
		return
	}

	// Aggiornamento DB
	err = rt.db.SetUserPhoto(targetUser.ID, req.ProfilePhoto)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Recupero utente aggiornato per la risposta
	updatedUser, _ := rt.db.GetUserByID(targetUser.ID)
	_ = json.NewEncoder(w).Encode(updatedUser)
}

// modifyProfileStatus aggiorna lo stato dell'utente
func (rt *_router) modifyProfileStatus(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	pathUsername := ps.ByName("username")
	authHeader := r.Header.Get("Authorization")
	requestingUserID := strings.TrimPrefix(authHeader, "Bearer ")

	// Classico controllo per verificare se l'utente che fa la richiesta è l'utente loggato in questa sessione
	targetUser, err := rt.db.GetUserByUsername(pathUsername)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if targetUser.ID != requestingUserID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req database.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Aggiornamento DB
	err = rt.db.SetUserStatus(targetUser.ID, req.Status)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	updatedUser, _ := rt.db.GetUserByID(targetUser.ID)
	_ = json.NewEncoder(w).Encode(updatedUser)
}
