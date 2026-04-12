package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

const sessionCookieName = "sme_session"

// sessionStore holds in-memory session tokens with expiry.
type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]time.Time
	ttl      time.Duration
}

func newSessionStore(ttl time.Duration) *sessionStore {
	return &sessionStore{
		sessions: make(map[string]time.Time),
		ttl:      ttl,
	}
}

func (ss *sessionStore) create() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand: " + err.Error())
	}
	token := hex.EncodeToString(b)
	ss.mu.Lock()
	ss.sessions[token] = time.Now().Add(ss.ttl)
	ss.mu.Unlock()
	return token
}

func (ss *sessionStore) valid(token string) bool {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	exp, ok := ss.sessions[token]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(ss.sessions, token)
		return false
	}
	return true
}

func (ss *sessionStore) delete(token string) {
	ss.mu.Lock()
	delete(ss.sessions, token)
	ss.mu.Unlock()
}

// requireAuth wraps a handler and returns 401 if no valid session cookie is present.
func (s *server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || !s.sessions.valid(cookie.Value) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r)
	}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// POST /api/auth/login
func (s *server) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Username != s.authUsername || req.Password != s.authPassword {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	token := s.sessions.create()
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(24 * time.Hour / time.Second),
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// POST /api/auth/logout
func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		s.sessions.delete(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// GET /api/auth/me — returns 200 if the session cookie is valid (used by the
// frontend to check auth state on load). The requireAuth middleware handles the
// 401 case so this handler is only reached when the session is already valid.
func (s *server) authMe(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
