package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/alexfishkazzz/shortener/internal/auth"
	"github.com/alexfishkazzz/shortener/internal/storage"
)

const baseURL = "http://localhost:8080"

// App содержит зависимости приложения.
type App struct {
	storage *storage.MemoryStorage
}

// New создаёт новый экземпляр App.
func New() *App {
	return &App{
		storage: storage.NewMemoryStorage(),
	}
}

// generateShortID генерирует случайный короткий идентификатор.
func generateShortID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ShortenRequest — тело запроса для POST /api/shorten.
type ShortenRequest struct {
	URL string `json:"url"`
}

// ShortenResponse — тело ответа для POST /api/shorten.
type ShortenResponse struct {
	Result string `json:"result"`
}

// ShortenHandler обрабатывает POST /api/shorten.
func (a *App) ShortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}

	userID, isNew, err := auth.GetUserFromRequest(r)
	if err != nil {
		// Для эндпоинта shorten считаем некорректную cookie как нового пользователя (выдаём новую)
		if errors.Is(err, auth.ErrMalformedCookie) {
			isNew = true
			err = nil
		} else {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	if isNew {
		var genErr error
		userID, genErr = auth.GenerateUserID()
		if genErr != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		auth.SetCookie(w, userID)
	}

	shortID, err := generateShortID()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Гарантируем уникальность (простой повтор, вероятность коллизии низкая)
	for i := 0; i < 5; i++ {
		if _, exists := a.storage.Get(shortID); !exists {
			break
		}
		shortID, err = generateShortID()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	a.storage.Save(shortID, req.URL, userID, baseURL)

	resp := ShortenResponse{
		Result: baseURL + "/" + shortID,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// RedirectHandler обрабатывает GET /{id}.
func (a *App) RedirectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID из пути: /{id}
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" || strings.Contains(path, "/") {
		http.NotFound(w, r)
		return
	}
	shortID := path

	original, found := a.storage.Get(shortID)
	if !found {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, original, http.StatusTemporaryRedirect)
}

// UserURLsHandler обрабатывает GET /api/user/urls.
func (a *App) UserURLsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, isNew, err := auth.GetUserFromRequest(r)
	if err != nil {
		if errors.Is(err, auth.ErrMalformedCookie) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if isNew {
		// Выдаём новую cookie новому пользователю
		var genErr error
		userID, genErr = auth.GenerateUserID()
		if genErr != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		auth.SetCookie(w, userID)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	urls := a.storage.GetUserURLs(userID)
	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(urls)
}

// ServeHTTP маршрутизирует запросы.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/shorten":
		a.ShortenHandler(w, r)
	case r.URL.Path == "/api/user/urls":
		a.UserURLsHandler(w, r)
	case strings.HasPrefix(r.URL.Path, "/") && len(r.URL.Path) > 1:
		a.RedirectHandler(w, r)
	default:
		http.NotFound(w, r)
	}
}
