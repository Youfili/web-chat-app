package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

const maxUploadSize = 10 << 20 // 10 MB
const baseUploadDir = "./images"

// uploadFile gestisce il caricamento delle immagini
func (rt *_router) uploadFile(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Verifico Autenticazione
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Gestione delle Cartelle --> Leggo il parametro ?type=avatar o ?type=media
	// --> per capire il tipo di operazione che sto facendo (cambio immagine profilo o invio immagine in chat)
	uploadType := r.URL.Query().Get("type")

	subFolder := "media" // Default: foto che invio in chat
	if uploadType == "avatar" {
		subFolder = "avatars"
	}

	// Percorso fisico: ./images/avatars o ./images/media
	finalDir := filepath.Join(baseUploadDir, subFolder)

	// Creo la cartella se non esiste
	if _, err := os.Stat(finalDir); os.IsNotExist(err) {
		if err := os.MkdirAll(finalDir, 0755); err != nil {
			rt.baseLogger.Errorf("Error creating directory: %v", err)
			http.Error(w, "Server configuration error (mkdir)", http.StatusInternalServerError)
			return
		}
	}

	// 3. Limito la dimensione dell'immagine
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "File too big (Max 10MB)", http.StatusBadRequest)
		return
	}

	// 4. Recupero il file
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Invalid file (key must be 'file')", http.StatusBadRequest)
		return
	}

	// Gestione chiusura file sorgente
	defer func() {
		if err := file.Close(); err != nil {
			rt.baseLogger.Errorf("Error closing source file: %v", err)
		}
	}()

	// 5. Validazione Magic Number --> mi serve a verificare se è davvero un immagine
	buff := make([]byte, 512)
	if _, err := file.Read(buff); err != nil {
		http.Error(w, "File read error", http.StatusInternalServerError)
		return
	}
	filetype := http.DetectContentType(buff)
	if filetype != "image/jpeg" && filetype != "image/png" && filetype != "image/gif" {
		http.Error(w, "Only JPEG, PNG or GIF allowed", http.StatusBadRequest)
		return
	}
	// Resetto il cursore di lettura all'inizio del file
	if _, err := file.Seek(0, 0); err != nil {
		rt.baseLogger.Errorf("Error rewinding file: %v", err)
		http.Error(w, "File processing error", http.StatusInternalServerError)
		return
	}

	// 6. Faccio la Generazione del Nome Univoco (aggiungo l'estensione se non presente)
	extension := filepath.Ext(fileHeader.Filename)
	if extension == "" {
		if filetype == "image/png" {
			extension = ".png"
		} else {
			extension = ".jpg"
		}
	}

	newFileName := uuid.New().String() + extension
	destinationPath := filepath.Join(finalDir, newFileName)

	// 7. Salvataggio su Disco
	dst, err := os.Create(destinationPath)
	if err != nil {
		rt.baseLogger.Errorf("Error creating file: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Gestione chiusura file destinazione
	defer func() {
		if err := dst.Close(); err != nil {
			rt.baseLogger.Errorf("Error closing destination file: %v", err)
		}
	}()

	if _, err := io.Copy(dst, file); err != nil {
		rt.baseLogger.Errorf("Error copying file: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 8. Generazione URL pubblico
	fileURL := "http://" + r.Host + "/images/" + subFolder + "/" + newFileName

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"url": fileURL})
}
