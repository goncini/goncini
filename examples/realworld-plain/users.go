package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"unicode/utf8"
)

type user struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	Bio          *string
	Image        *string
}

func (s *server) userByID(ctx context.Context, id int64) (user, error) {
	return s.queryUser(ctx, "id = ?", id)
}

func (s *server) userByEmail(ctx context.Context, email string) (user, error) {
	return s.queryUser(ctx, "email = ?", email)
}

func (s *server) queryUser(ctx context.Context, where string, arg any) (user, error) {
	var u user
	err := s.db.QueryRowContext(ctx,
		"SELECT id, username, email, password_hash, bio, image FROM users WHERE "+where, arg,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Bio, &u.Image)
	return u, err
}

// respondUser writes u with a freshly issued token.
func (s *server) respondUser(w http.ResponseWriter, r *http.Request, status int, u user) {
	token, err := s.issueToken(u.ID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.writeJSON(w, r, status, envelope{"user": envelope{
		"email":    u.Email,
		"token":    token,
		"username": u.Username,
		"bio":      u.Bio,
		"image":    u.Image,
	}})
}

// writeTaken answers 409 if err is a duplicate username or email, and
// reports whether it did.
func (s *server) writeTaken(w http.ResponseWriter, r *http.Request, err error) bool {
	for _, field := range []string{"username", "email"} {
		if isUniqueViolation(err, "users."+field) {
			s.writeError(w, r, http.StatusConflict, field, "has already been taken")
			return true
		}
	}
	return false
}

const minPasswordLen = 8

func validatePassword(errs validationErrors, password string) {
	switch {
	case password == "":
		errs.add("password", "can't be blank")
	case utf8.RuneCountInString(password) < minPasswordLen:
		errs.add("password", "is too short (minimum is 8 characters)")
	}
}

func (s *server) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		User struct {
			Username string `json:"username"`
			Email    string `json:"email"`
			Password string `json:"password"`
		} `json:"user"`
	}
	if !s.decode(w, r, &req) {
		return
	}
	in := req.User

	errs := validationErrors{}
	if blank(in.Username) {
		errs.add("username", "can't be blank")
	}
	if blank(in.Email) {
		errs.add("email", "can't be blank")
	}
	validatePassword(errs, in.Password)
	if len(errs) > 0 {
		s.writeErrors(w, r, http.StatusUnprocessableEntity, errs)
		return
	}

	u := user{Username: in.Username, Email: in.Email, PasswordHash: hashPassword(in.Password)}
	res, err := s.db.ExecContext(r.Context(),
		"INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)",
		u.Username, u.Email, u.PasswordHash)
	if err != nil {
		if !s.writeTaken(w, r, err) {
			s.serverError(w, r, err)
		}
		return
	}
	if u.ID, err = res.LastInsertId(); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.respondUser(w, r, http.StatusCreated, u)
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		User struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		} `json:"user"`
	}
	if !s.decode(w, r, &req) {
		return
	}
	in := req.User

	errs := validationErrors{}
	if blank(in.Email) {
		errs.add("email", "can't be blank")
	}
	if in.Password == "" {
		errs.add("password", "can't be blank")
	}
	if len(errs) > 0 {
		s.writeErrors(w, r, http.StatusUnprocessableEntity, errs)
		return
	}

	u, err := s.userByEmail(r.Context(), in.Email)
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, r, http.StatusUnauthorized, "credentials", "invalid")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	ok, err := checkPassword(u.PasswordHash, in.Password)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if !ok {
		s.writeError(w, r, http.StatusUnauthorized, "credentials", "invalid")
		return
	}
	s.respondUser(w, r, http.StatusOK, u)
}

func (s *server) currentUser(w http.ResponseWriter, r *http.Request) {
	id, _ := userID(r)
	u, err := s.userByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		s.unauthorized(w, r, "is invalid")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.respondUser(w, r, http.StatusOK, u)
}

func (s *server) updateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		User struct {
			Username optional[string] `json:"username"`
			Email    optional[string] `json:"email"`
			Password optional[string] `json:"password"`
			Bio      optional[string] `json:"bio"`
			Image    optional[string] `json:"image"`
		} `json:"user"`
	}
	if !s.decode(w, r, &req) {
		return
	}
	in := req.User

	errs := validationErrors{}
	if in.Username.Set && (in.Username.Value == nil || blank(*in.Username.Value)) {
		errs.add("username", "can't be blank")
	}
	if in.Email.Set && (in.Email.Value == nil || blank(*in.Email.Value)) {
		errs.add("email", "can't be blank")
	}
	if in.Password.Set {
		if in.Password.Value == nil {
			errs.add("password", "can't be blank")
		} else {
			validatePassword(errs, *in.Password.Value)
		}
	}
	if len(errs) > 0 {
		s.writeErrors(w, r, http.StatusUnprocessableEntity, errs)
		return
	}

	id, _ := userID(r)
	u, err := s.userByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		s.unauthorized(w, r, "is invalid")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if in.Username.Set {
		u.Username = *in.Username.Value
	}
	if in.Email.Set {
		u.Email = *in.Email.Value
	}
	if in.Password.Set {
		u.PasswordHash = hashPassword(*in.Password.Value)
	}
	if in.Bio.Set {
		u.Bio = nilIfEmpty(in.Bio.Value)
	}
	if in.Image.Set {
		u.Image = nilIfEmpty(in.Image.Value)
	}

	_, err = s.db.ExecContext(r.Context(),
		"UPDATE users SET username = ?, email = ?, password_hash = ?, bio = ?, image = ? WHERE id = ?",
		u.Username, u.Email, u.PasswordHash, u.Bio, u.Image, u.ID)
	if err != nil {
		if !s.writeTaken(w, r, err) {
			s.serverError(w, r, err)
		}
		return
	}
	s.respondUser(w, r, http.StatusOK, u)
}

func nilIfEmpty(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}

// Profiles

type profile struct {
	Username  string  `json:"username"`
	Bio       *string `json:"bio"`
	Image     *string `json:"image"`
	Following bool    `json:"following"`
}

// profileByUsername returns the user's ID and profile as seen by viewer
// (0 for an anonymous viewer).
func (s *server) profileByUsername(ctx context.Context, viewer int64, username string) (int64, profile, error) {
	var id int64
	var p profile
	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, bio, image,
		       EXISTS (SELECT 1 FROM follows WHERE follower_id = ? AND followed_id = users.id)
		FROM users WHERE username = ?`, viewer, username,
	).Scan(&id, &p.Username, &p.Bio, &p.Image, &p.Following)
	return id, p, err
}

func (s *server) getProfile(w http.ResponseWriter, r *http.Request) {
	viewer, _ := userID(r)
	_, p, err := s.profileByUsername(r.Context(), viewer, r.PathValue("username"))
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, r, http.StatusNotFound, "profile", "not found")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, envelope{"profile": p})
}

func (s *server) follow(w http.ResponseWriter, r *http.Request) {
	s.setFollowing(w, r, true)
}

func (s *server) unfollow(w http.ResponseWriter, r *http.Request) {
	s.setFollowing(w, r, false)
}

func (s *server) setFollowing(w http.ResponseWriter, r *http.Request, following bool) {
	viewer, _ := userID(r)
	id, p, err := s.profileByUsername(r.Context(), viewer, r.PathValue("username"))
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, r, http.StatusNotFound, "profile", "not found")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}

	query := "DELETE FROM follows WHERE follower_id = ? AND followed_id = ?"
	if following {
		query = "INSERT INTO follows (follower_id, followed_id) VALUES (?, ?) ON CONFLICT DO NOTHING"
	}
	if _, err := s.db.ExecContext(r.Context(), query, viewer, id); err != nil {
		s.serverError(w, r, err)
		return
	}
	p.Following = following
	s.writeJSON(w, r, http.StatusOK, envelope{"profile": p})
}
