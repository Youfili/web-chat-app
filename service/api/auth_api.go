package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"github.com/Youfili/web-chat-app/service/database"
	"github.com/julienschmidt/httprouter"
)

// doLogin gestisce il login utente (verifica esistenza)
func (rt *_router) doLogin(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Decodifica del body JSON
	var req database.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// 2. Validazione Input
	if !isValidUsername(req.Username) {
		http.Error(w, "Invalid username format", http.StatusBadRequest)
		return
	}

	// 3. Chiamata al Database
	user, err := rt.db.GetUserByUsername(req.Username)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			// Seguendo la specifica che ho implementato nell'API Doc: Se non esiste, 404 (User must register)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		// Altrimenti, Errore interno del server
		rt.baseLogger.Errorf("Error fetching user during login: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 4. Risposta JSON (200 OK)
	// Costruisco la risposta usando la struct 'AuthResponse' definita in struct.go
	resp := database.AuthResponse{
		ID:           user.ID,
		Username:     user.Username,
		ProfilePhoto: user.ProfilePhoto,
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// registerUser crea un nuovo utente
func (rt *_router) registerUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Decodifica del body JSON
	var req database.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// 2. Validazione Input
	if !isValidUsername(req.Username) {
		http.Error(w, "Invalid username format (3-30 chars, alphanumeric)", http.StatusBadRequest)
		return
	}

	// 3. Creazione Oggetto Utente
	// Nota: L'ID viene generato dentro db.CreateUser
	newUser := database.User{
		Username:     req.Username,
		Status:       req.Status,
		ProfilePhoto: req.ProfilePhoto,
	}
	// Default profile photo (se non fornita dall'utente)
	if newUser.ProfilePhoto == "" {
		// Lo slash iniziale indica la root del server frontend
		newUser.ProfilePhoto = "/default_avatar.jpg"
	}

	// 4. Chiamata al Database
	createdUser, err := rt.db.CreateUser(newUser)
	if err != nil {
		if errors.Is(err, database.ErrUsernameTaken) {
			http.Error(w, "Username already exists", http.StatusBadRequest)
			return
		}
		rt.baseLogger.Errorf("Error creating user: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 5. Risposta JSON (201 Created)
	resp := database.AuthResponse{
		ID:           createdUser.ID,
		Username:     createdUser.Username,
		ProfilePhoto: createdUser.ProfilePhoto,
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

// Funzione che "valida" lo username scelto in fase di registrazione (secondo la REGEX definita nell'API Doc)
func isValidUsername(u string) bool {
	// Min 3, Max 30, solo lettere, numeri, . _ ~ -
	if len(u) < 3 || len(u) > 30 {
		return false
	}
	// Compila la regex
	re := regexp.MustCompile(`^[a-zA-Z0-9._~-]+$`)
	return re.MatchString(u)
}
