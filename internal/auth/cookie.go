package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"
)

const (
	secretKey  = "super-secret-key"
	cookieName = "user_id"
	cookiePath = "/"
)

var (
	ErrNoCookie         = errors.New("no cookie")
	ErrMalformedCookie  = errors.New("malformed cookie")
	ErrInvalidSignature = errors.New("invalid signature")
)

// GenerateUserID создаёт случайный уникальный идентификатор пользователя.
func GenerateUserID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// sign вычисляет HMAC-SHA256 подпись для userID.
func sign(userID string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(userID))
	return hex.EncodeToString(h.Sum(nil))
}

// CreateCookieValue создаёт подписанное значение cookie: userID:signature
func CreateCookieValue(userID string) string {
	return userID + ":" + sign(userID)
}

// ValidateCookieValue проверяет значение cookie и возвращает userID, если оно валидно.
// Возвращает ошибку, если формат некорректен или подпись неверна.
func ValidateCookieValue(value string) (string, error) {
	if value == "" {
		return "", ErrMalformedCookie
	}
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", ErrMalformedCookie
	}
	userID := parts[0]
	signature := parts[1]

	expected := sign(userID)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return "", ErrInvalidSignature
	}
	return userID, nil
}

// SetCookie устанавливает cookie пользователя в ответ.
func SetCookie(w http.ResponseWriter, userID string) {
	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    CreateCookieValue(userID),
		Path:     cookiePath,
		HttpOnly: true,
		Expires:  time.Now().Add(365 * 24 * time.Hour),
	}
	http.SetCookie(w, cookie)
}

// GetUserFromRequest извлекает userID из cookie запроса.
// Возвращает:
// - userID, false, nil — если валидный существующий пользователь
// - "", true, nil — если нужно создать нового (нет cookie или неверная подпись)
// - "", false, ErrMalformedCookie — если cookie есть, но формат некорректен
func GetUserFromRequest(r *http.Request) (userID string, isNew bool, err error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return "", true, nil
		}
		return "", false, err
	}

	if cookie.Value == "" {
		return "", false, ErrMalformedCookie
	}

	userID, err = ValidateCookieValue(cookie.Value)
	if err != nil {
		if errors.Is(err, ErrMalformedCookie) {
			return "", false, ErrMalformedCookie
		}
		// Неверная подпись -> считаем новым пользователем
		return "", true, nil
	}
	return userID, false, nil
}
