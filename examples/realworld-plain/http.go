package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

type server struct {
	db     *sql.DB
	secret []byte
	log    *slog.Logger
}

func newServer(db *sql.DB, secret []byte, logger *slog.Logger) http.Handler {
	s := &server{db: db, secret: secret, log: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/users", s.register)
	mux.HandleFunc("POST /api/users/login", s.login)
	mux.HandleFunc("GET /api/user", s.requireAuth(s.currentUser))
	mux.HandleFunc("PUT /api/user", s.requireAuth(s.updateUser))

	mux.HandleFunc("GET /api/profiles/{username}", s.getProfile)
	mux.HandleFunc("POST /api/profiles/{username}/follow", s.requireAuth(s.follow))
	mux.HandleFunc("DELETE /api/profiles/{username}/follow", s.requireAuth(s.unfollow))

	mux.HandleFunc("GET /api/articles", s.listArticles)
	mux.HandleFunc("GET /api/articles/feed", s.requireAuth(s.feed))
	mux.HandleFunc("POST /api/articles", s.requireAuth(s.createArticle))
	mux.HandleFunc("GET /api/articles/{slug}", s.getArticle)
	mux.HandleFunc("PUT /api/articles/{slug}", s.requireAuth(s.updateArticle))
	mux.HandleFunc("DELETE /api/articles/{slug}", s.requireAuth(s.deleteArticle))
	mux.HandleFunc("POST /api/articles/{slug}/favorite", s.requireAuth(s.favorite))
	mux.HandleFunc("DELETE /api/articles/{slug}/favorite", s.requireAuth(s.unfavorite))

	mux.HandleFunc("GET /api/articles/{slug}/comments", s.listComments)
	mux.HandleFunc("POST /api/articles/{slug}/comments", s.requireAuth(s.addComment))
	mux.HandleFunc("DELETE /api/articles/{slug}/comments/{id}", s.requireAuth(s.deleteComment))

	mux.HandleFunc("GET /api/tags", s.listTags)

	return s.logFailures(s.recoverPanics(s.authenticate(mux)))
}

// Middleware

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// logFailures logs every response with a 4xx or 5xx status.
func (s *server) logFailures(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		if rec.status < 400 {
			return
		}
		level := slog.LevelInfo
		if rec.status >= 500 {
			level = slog.LevelError
		}
		s.log.LogAttrs(r.Context(), level, "request failed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rec.status),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

func (s *server) recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if v == http.ErrAbortHandler {
				panic(v)
			}
			s.log.ErrorContext(r.Context(), "panic",
				"method", r.Method, "path", r.URL.Path,
				"panic", v, "stack", string(debug.Stack()))
			s.writeError(w, r, http.StatusInternalServerError, "server", "internal error")
		}()
		next.ServeHTTP(w, r)
	})
}

type userIDKey struct{}

// authenticate puts the ID of the user named by a valid "Authorization:
// Token <jwt>" header into the request context. A request without the
// header passes through anonymously; one with a bad token is rejected.
func (s *server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			next.ServeHTTP(w, r)
			return
		}
		token, ok := strings.CutPrefix(header, "Token ")
		if !ok {
			s.unauthorized(w, r, "is invalid")
			return
		}
		id, err := s.parseToken(token)
		if err != nil {
			s.unauthorized(w, r, "is invalid")
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := userID(r); !ok {
			s.unauthorized(w, r, "is missing")
			return
		}
		next(w, r)
	}
}

// userID returns the authenticated user's ID, or 0 and false for an
// anonymous request.
func userID(r *http.Request) (int64, bool) {
	id, ok := r.Context().Value(userIDKey{}).(int64)
	return id, ok
}

// Requests and responses

const maxBodyBytes = 1 << 20

// decode reads the JSON request body into dst. If it fails, it writes the
// error response and returns false.
func (s *server) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(dst)
	if err == nil {
		return true
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		s.writeError(w, r, http.StatusRequestEntityTooLarge, "body", "is too large")
	} else {
		s.writeError(w, r, http.StatusBadRequest, "body", "is not valid JSON")
	}
	return false
}

// optional is a request field that tells an absent key (Set is false) from
// null (Value is nil).
type optional[T any] struct {
	Set   bool
	Value *T
}

func (o *optional[T]) UnmarshalJSON(b []byte) error {
	o.Set = true
	return json.Unmarshal(b, &o.Value)
}

type envelope map[string]any

func (s *server) writeJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		s.log.ErrorContext(r.Context(), "encoding response", "path", r.URL.Path, "err", err)
		http.Error(w, `{"errors":{"server":["internal error"]}}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	// A failed write means the client went away; there is no one to tell.
	_, _ = w.Write(body)
}

type validationErrors map[string][]string

func (e validationErrors) add(field, message string) {
	e[field] = append(e[field], message)
}

func blank(s string) bool { return strings.TrimSpace(s) == "" }

func (s *server) writeErrors(w http.ResponseWriter, r *http.Request, status int, errs validationErrors) {
	s.writeJSON(w, r, status, envelope{"errors": errs})
}

func (s *server) writeError(w http.ResponseWriter, r *http.Request, status int, field, message string) {
	s.writeErrors(w, r, status, validationErrors{field: {message}})
}

func (s *server) unauthorized(w http.ResponseWriter, r *http.Request, message string) {
	w.Header().Set("WWW-Authenticate", "Token")
	s.writeError(w, r, http.StatusUnauthorized, "token", message)
}

// serverError logs an unexpected error and answers with a generic 500.
func (s *server) serverError(w http.ResponseWriter, r *http.Request, err error) {
	s.log.ErrorContext(r.Context(), "internal error",
		"method", r.Method, "path", r.URL.Path, "err", err)
	s.writeError(w, r, http.StatusInternalServerError, "server", "internal error")
}
