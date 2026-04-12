package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ── sessionStore ──────────────────────────────────────────────────────────────

func TestSessionStore_CreateAndValid(t *testing.T) {
	ss := newSessionStore(time.Hour)
	token := ss.create()
	if !ss.valid(token) {
		t.Error("expected newly-created token to be valid")
	}
}

func TestSessionStore_UnknownTokenInvalid(t *testing.T) {
	ss := newSessionStore(time.Hour)
	if ss.valid("not-a-real-token") {
		t.Error("expected unknown token to be invalid")
	}
}

func TestSessionStore_ExpiredTokenInvalid(t *testing.T) {
	ss := newSessionStore(time.Millisecond)
	token := ss.create()
	time.Sleep(10 * time.Millisecond)
	if ss.valid(token) {
		t.Error("expected expired token to be invalid")
	}
}

func TestSessionStore_Delete(t *testing.T) {
	ss := newSessionStore(time.Hour)
	token := ss.create()
	ss.delete(token)
	if ss.valid(token) {
		t.Error("expected deleted token to be invalid")
	}
}

// ── login handler ─────────────────────────────────────────────────────────────

func newAuthServer() *server {
	return &server{
		store:        newStore(),
		sessions:     newSessionStore(time.Hour),
		authUsername: "admin",
		authPassword: "secret",
	}
}

func TestLogin_Success(t *testing.T) {
	srv := newAuthServer()
	body := `{"username":"admin","password":"secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var found bool
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookieName {
			found = true
			if !c.HttpOnly {
				t.Error("session cookie must be HttpOnly")
			}
		}
	}
	if !found {
		t.Errorf("expected %q cookie in response", sessionCookieName)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	srv := newAuthServer()
	body := `{"username":"admin","password":"nope"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestLogin_BadJSON(t *testing.T) {
	srv := newAuthServer()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ── requireAuth middleware ────────────────────────────────────────────────────

func TestRequireAuth_NoCookie(t *testing.T) {
	srv := newAuthServer()
	called := false
	handler := srv.requireAuth(func(w http.ResponseWriter, r *http.Request) { called = true })
	req := httptest.NewRequest(http.MethodGet, "/api/documents", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if called {
		t.Error("inner handler should not be called without valid cookie")
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	srv := newAuthServer()
	called := false
	handler := srv.requireAuth(func(w http.ResponseWriter, r *http.Request) { called = true })
	req := httptest.NewRequest(http.MethodGet, "/api/documents", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "bogus"})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if called {
		t.Error("inner handler should not be called with invalid token")
	}
}

func TestRequireAuth_ValidToken(t *testing.T) {
	ss := newSessionStore(time.Hour)
	token := ss.create()
	srv := &server{store: newStore(), sessions: ss}
	called := false
	handler := srv.requireAuth(func(w http.ResponseWriter, r *http.Request) { called = true })
	req := httptest.NewRequest(http.MethodGet, "/api/documents", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	w := httptest.NewRecorder()

	handler(w, req)

	if !called {
		t.Error("inner handler should be called with a valid token")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ── logout handler ────────────────────────────────────────────────────────────

func TestLogout_ClearsSession(t *testing.T) {
	ss := newSessionStore(time.Hour)
	token := ss.create()
	srv := &server{store: newStore(), sessions: ss}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	w := httptest.NewRecorder()

	srv.logout(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ss.valid(token) {
		t.Error("session should be invalidated after logout")
	}
}
