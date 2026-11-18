package storage

import (
	"sync"
)

// UserURL представляет сокращённый URL, связанный с пользователем.
type UserURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// MemoryStorage хранит URL в памяти.
type MemoryStorage struct {
	mu       sync.RWMutex
	urls     map[string]string    // shortID -> originalURL
	userURLs map[string][]UserURL // userID -> список UserURL
}

// NewMemoryStorage создаёт новое in-memory хранилище.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		urls:     make(map[string]string),
		userURLs: make(map[string][]UserURL),
	}
}

// Save сохраняет связь между shortID и originalURL для указанного пользователя.
func (s *MemoryStorage) Save(shortID, originalURL, userID, baseURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.urls[shortID] = originalURL

	shortURL := baseURL + "/" + shortID
	s.userURLs[userID] = append(s.userURLs[userID], UserURL{
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	})
}

// Get возвращает оригинальный URL для заданного shortID.
// Возвращает пустую строку, если не найден.
func (s *MemoryStorage) Get(shortID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, ok := s.urls[shortID]
	return url, ok
}

// GetUserURLs возвращает все URL, сокращённые пользователем.
// Возвращает пустой срез, если их нет.
func (s *MemoryStorage) GetUserURLs(userID string) []UserURL {
	s.mu.RLock()
	defer s.mu.RUnlock()

	urls := s.userURLs[userID]
	if urls == nil {
		return []UserURL{}
	}
	// Возвращаем копию, чтобы избежать внешней модификации
	result := make([]UserURL, len(urls))
	copy(result, urls)
	return result
}
