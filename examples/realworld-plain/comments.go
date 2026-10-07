package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"
)

type comment struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Body      string    `json:"body"`
	Author    profile   `json:"author"`
}

// queryComments selects comments as seen by viewer, filtered and ordered by
// clause.
func (s *server) queryComments(ctx context.Context, viewer int64, clause string, args ...any) ([]comment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.created_at, c.updated_at, c.body, u.username, u.bio, u.image,
		       EXISTS (SELECT 1 FROM follows WHERE follower_id = ? AND followed_id = c.author_id)
		FROM comments c
		JOIN users u ON u.id = c.author_id
		`+clause, append([]any{viewer}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []comment{}
	for rows.Next() {
		var c comment
		var created, updated int64
		err := rows.Scan(&c.ID, &created, &updated, &c.Body,
			&c.Author.Username, &c.Author.Bio, &c.Author.Image, &c.Author.Following)
		if err != nil {
			return nil, err
		}
		c.CreatedAt = fromMicros(created)
		c.UpdatedAt = fromMicros(updated)
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (s *server) listComments(w http.ResponseWriter, r *http.Request) {
	articleID, _, err := s.findArticle(r.Context(), r.PathValue("slug"))
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, r, http.StatusNotFound, "article", "not found")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}

	viewer, _ := userID(r)
	comments, err := s.queryComments(r.Context(), viewer, "WHERE c.article_id = ? ORDER BY c.id", articleID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, envelope{"comments": comments})
}

func (s *server) addComment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Comment struct {
			Body string `json:"body"`
		} `json:"comment"`
	}
	if !s.decode(w, r, &req) {
		return
	}
	if blank(req.Comment.Body) {
		s.writeError(w, r, http.StatusUnprocessableEntity, "body", "can't be blank")
		return
	}

	articleID, _, err := s.findArticle(r.Context(), r.PathValue("slug"))
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, r, http.StatusNotFound, "article", "not found")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}

	viewer, _ := userID(r)
	now := nowMicros()
	res, err := s.db.ExecContext(r.Context(), `
		INSERT INTO comments (article_id, author_id, body, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`,
		articleID, viewer, req.Comment.Body, now, now)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		s.serverError(w, r, err)
		return
	}

	comments, err := s.queryComments(r.Context(), viewer, "WHERE c.id = ?", id)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if len(comments) == 0 {
		s.serverError(w, r, errors.New("inserted comment not found"))
		return
	}
	s.writeJSON(w, r, http.StatusCreated, envelope{"comment": comments[0]})
}

func (s *server) deleteComment(w http.ResponseWriter, r *http.Request) {
	articleID, _, err := s.findArticle(r.Context(), r.PathValue("slug"))
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, r, http.StatusNotFound, "article", "not found")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.writeError(w, r, http.StatusNotFound, "comment", "not found")
		return
	}
	var authorID int64
	err = s.db.QueryRowContext(r.Context(),
		"SELECT author_id FROM comments WHERE id = ? AND article_id = ?", id, articleID,
	).Scan(&authorID)
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, r, http.StatusNotFound, "comment", "not found")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if viewer, _ := userID(r); authorID != viewer {
		s.writeError(w, r, http.StatusForbidden, "comment", "forbidden")
		return
	}

	if _, err := s.db.ExecContext(r.Context(), "DELETE FROM comments WHERE id = ?", id); err != nil {
		s.serverError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
