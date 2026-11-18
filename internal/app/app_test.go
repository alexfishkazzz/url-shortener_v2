package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexfishkazzz/shortener/internal/auth"
	"github.com/alexfishkazzz/shortener/internal/storage"
)

func TestShortenAndRedirect(t *testing.T) {
	a := New()

	// Create short URL
	body := bytes.NewBufferString(`{"url":"https://example.com/long/path"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	a.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	// Check cookie was set
	cookies := rr.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected cookie to be set")
	}
	cookie := cookies[0]
	if cookie.Name != "user_id" {
		t.Errorf("expected cookie name user_id, got %s", cookie.Name)
	}

	var resp ShortenResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Result == "" {
		t.Fatal("expected result URL")
	}

	// Extract short ID and test redirect
	shortPath := resp.Result[len("http://localhost:8080"):]
	req2 := httptest.NewRequest(http.MethodGet, shortPath, nil)
	rr2 := httptest.NewRecorder()
	a.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected redirect %d, got %d", http.StatusTemporaryRedirect, rr2.Code)
	}
	if loc := rr2.Header().Get("Location"); loc != "https://example.com/long/path" {
		t.Errorf("expected redirect to original, got %s", loc)
	}
}

func TestUserURLs(t *testing.T) {
	a := New()

	// New user, no URLs -> 204
	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	rr := httptest.NewRecorder()
	a.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
	// Should set cookie
	if len(rr.Result().Cookies()) == 0 {
		t.Fatal("expected cookie")
	}

	// Create a URL with the cookie
	cookie := rr.Result().Cookies()[0]
	body := bytes.NewBufferString(`{"url":"https://example.com/test"}`)
	req2 := httptest.NewRequest(http.MethodPost, "/api/shorten", body)
	req2.AddCookie(cookie)
	rr2 := httptest.NewRecorder()
	a.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusCreated {
		t.Fatalf("expected created, got %d", rr2.Code)
	}

	// Get user URLs
	req3 := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req3.AddCookie(cookie)
	rr3 := httptest.NewRecorder()
	a.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr3.Code)
	}

	var urls []storage.UserURL
	if err := json.NewDecoder(rr3.Body).Decode(&urls); err != nil {
		t.Fatal(err)
	}
	if len(urls) != 1 {
		t.Fatalf("expected 1 url, got %d", len(urls))
	}
	if urls[0].OriginalURL != "https://example.com/test" {
		t.Errorf("unexpected original URL: %s", urls[0].OriginalURL)
	}
}

func TestMalformedCookie(t *testing.T) {
	a := New()

	// Empty cookie
	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req.AddCookie(&http.Cookie{Name: "user_id", Value: ""})
	rr := httptest.NewRecorder()
	a.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for empty cookie, got %d", rr.Code)
	}

	// Malformed cookie (no colon)
	req2 := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req2.AddCookie(&http.Cookie{Name: "user_id", Value: "invalidvalue"})
	rr2 := httptest.NewRecorder()
	a.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for malformed cookie, got %d", rr2.Code)
	}
}

func TestInvalidSignature(t *testing.T) {
	a := New()

	// Cookie with invalid signature -> treat as new, 204
	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req.AddCookie(&http.Cookie{Name: "user_id", Value: "someuserid:invalidsig"})
	rr := httptest.NewRecorder()
	a.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for invalid signature, got %d", rr.Code)
	}
	// Should set new cookie
	if len(rr.Result().Cookies()) == 0 {
		t.Fatal("expected new cookie")
	}
}

func TestNotFound(t *testing.T) {
	a := New()

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rr := httptest.NewRecorder()
	a.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestAuthCookie(t *testing.T) {
	userID, err := auth.GenerateUserID()
	if err != nil {
		t.Fatal(err)
	}
	value := auth.CreateCookieValue(userID)

	got, err := auth.ValidateCookieValue(value)
	if err != nil {
		t.Fatal(err)
	}
	if got != userID {
		t.Errorf("expected %s, got %s", userID, got)
	}

	// Tampered
	_, err = auth.ValidateCookieValue(userID + ":tampered")
	if err == nil {
		t.Error("expected error for tampered signature")
	}
}
