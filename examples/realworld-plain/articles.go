package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type article struct {
	ID             int64     `json:"-"`
	Slug           string    `json:"slug"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Body           *string   `json:"body,omitempty"` // Left out of lists.
	TagList        []string  `json:"tagList"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	Favorited      bool      `json:"favorited"`
	FavoritesCount int       `json:"favoritesCount"`
	Author         profile   `json:"author"`
}

// articleQuery selects articles as seen by a viewer, whose ID is its first
// two arguments.
const articleQuery = `
SELECT a.id, a.slug, a.title, a.description, a.body, a.created_at, a.updated_at,
       u.username, u.bio, u.image,
       EXISTS (SELECT 1 FROM follows WHERE follower_id = ? AND followed_id = a.author_id),
       EXISTS (SELECT 1 FROM favorites WHERE user_id = ? AND article_id = a.id),
       (SELECT COUNT(*) FROM favorites WHERE article_id = a.id)
FROM articles a
JOIN users u ON u.id = a.author_id
`

// queryArticles runs articleQuery followed by clause, and loads the tags of
// the articles it finds.
func (s *server) queryArticles(ctx context.Context, viewer int64, clause string, args ...any) ([]article, error) {
	rows, err := s.db.QueryContext(ctx, articleQuery+clause, append([]any{viewer, viewer}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	articles := []article{}
	for rows.Next() {
		var a article
		var body string
		var created, updated int64
		err := rows.Scan(&a.ID, &a.Slug, &a.Title, &a.Description, &body, &created, &updated,
			&a.Author.Username, &a.Author.Bio, &a.Author.Image, &a.Author.Following,
			&a.Favorited, &a.FavoritesCount)
		if err != nil {
			return nil, err
		}
		a.Body = &body
		a.TagList = []string{}
		a.CreatedAt = fromMicros(created)
		a.UpdatedAt = fromMicros(updated)
		articles = append(articles, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.loadTags(ctx, articles); err != nil {
		return nil, err
	}
	return articles, nil
}

func (s *server) loadTags(ctx context.Context, articles []article) error {
	if len(articles) == 0 {
		return nil
	}
	index := make(map[int64]int, len(articles))
	args := make([]any, len(articles))
	for i, a := range articles {
		index[a.ID] = i
		args[i] = a.ID
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(articles)), ", ")
	rows, err := s.db.QueryContext(ctx,
		"SELECT article_id, tag FROM article_tags WHERE article_id IN ("+placeholders+") ORDER BY position",
		args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var tag string
		if err := rows.Scan(&id, &tag); err != nil {
			return err
		}
		a := &articles[index[id]]
		a.TagList = append(a.TagList, tag)
	}
	return rows.Err()
}

func (s *server) articleByID(ctx context.Context, viewer, id int64) (article, error) {
	articles, err := s.queryArticles(ctx, viewer, "WHERE a.id = ?", id)
	if err != nil {
		return article{}, err
	}
	if len(articles) == 0 {
		return article{}, sql.ErrNoRows
	}
	return articles[0], nil
}

// findArticle returns the ID and author of the article with the slug.
func (s *server) findArticle(ctx context.Context, slug string) (id, authorID int64, err error) {
	err = s.db.QueryRowContext(ctx, "SELECT id, author_id FROM articles WHERE slug = ?", slug).
		Scan(&id, &authorID)
	return id, authorID, err
}

func (s *server) getArticle(w http.ResponseWriter, r *http.Request) {
	viewer, _ := userID(r)
	articles, err := s.queryArticles(r.Context(), viewer, "WHERE a.slug = ?", r.PathValue("slug"))
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if len(articles) == 0 {
		s.writeError(w, r, http.StatusNotFound, "article", "not found")
		return
	}
	s.writeJSON(w, r, http.StatusOK, envelope{"article": articles[0]})
}

func (s *server) listArticles(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var conds []string
	var args []any
	if tag := q.Get("tag"); tag != "" {
		conds = append(conds, "a.id IN (SELECT article_id FROM article_tags WHERE tag = ?)")
		args = append(args, tag)
	}
	if author := q.Get("author"); author != "" {
		conds = append(conds, "u.username = ?")
		args = append(args, author)
	}
	if favorited := q.Get("favorited"); favorited != "" {
		conds = append(conds, `a.id IN (
			SELECT f.article_id FROM favorites f JOIN users fu ON fu.id = f.user_id
			WHERE fu.username = ?)`)
		args = append(args, favorited)
	}
	s.respondArticles(w, r, conds, args)
}

func (s *server) feed(w http.ResponseWriter, r *http.Request) {
	viewer, _ := userID(r)
	s.respondArticles(w, r,
		[]string{"a.author_id IN (SELECT followed_id FROM follows WHERE follower_id = ?)"},
		[]any{viewer})
}

// respondArticles writes the page of articles matching all conds, newest
// first, as requested by the limit and offset query parameters.
func (s *server) respondArticles(w http.ResponseWriter, r *http.Request, conds []string, args []any) {
	q := r.URL.Query()
	errs := validationErrors{}
	limit := intParam(q, "limit", 20, errs)
	offset := intParam(q, "offset", 0, errs)
	if len(errs) > 0 {
		s.writeErrors(w, r, http.StatusUnprocessableEntity, errs)
		return
	}

	var where string
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	var count int
	err := s.db.QueryRowContext(r.Context(),
		"SELECT COUNT(*) FROM articles a JOIN users u ON u.id = a.author_id "+where, args...,
	).Scan(&count)
	if err != nil {
		s.serverError(w, r, err)
		return
	}

	viewer, _ := userID(r)
	articles, err := s.queryArticles(r.Context(), viewer,
		where+" ORDER BY a.created_at DESC, a.id DESC LIMIT ? OFFSET ?",
		append(args, limit, offset)...)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	for i := range articles {
		articles[i].Body = nil
	}
	s.writeJSON(w, r, http.StatusOK, envelope{"articles": articles, "articlesCount": count})
}

// intParam returns the non-negative integer query parameter name, or def if
// it is absent.
func intParam(q url.Values, name string, def int, errs validationErrors) int {
	v := q.Get(name)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		errs.add(name, "must be a non-negative integer")
		return def
	}
	return n
}

func (s *server) createArticle(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Article struct {
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Body        string   `json:"body"`
			TagList     []string `json:"tagList"`
		} `json:"article"`
	}
	if !s.decode(w, r, &req) {
		return
	}
	in := req.Article

	errs := validationErrors{}
	if blank(in.Title) {
		errs.add("title", "can't be blank")
	}
	if blank(in.Description) {
		errs.add("description", "can't be blank")
	}
	if blank(in.Body) {
		errs.add("body", "can't be blank")
	}
	if len(errs) > 0 {
		s.writeErrors(w, r, http.StatusUnprocessableEntity, errs)
		return
	}

	ctx := r.Context()
	viewer, _ := userID(r)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	defer tx.Rollback()

	var id int64
	now := nowMicros()
	err = withUniqueSlug(in.Title, func(slug string) error {
		res, err := tx.ExecContext(ctx, `
			INSERT INTO articles (slug, title, description, body, author_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			slug, in.Title, in.Description, in.Body, viewer, now, now)
		if err != nil {
			return err
		}
		id, err = res.LastInsertId()
		return err
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if err := setTags(ctx, tx, id, in.TagList); err != nil {
		s.serverError(w, r, err)
		return
	}
	if err := tx.Commit(); err != nil {
		s.serverError(w, r, err)
		return
	}

	a, err := s.articleByID(ctx, viewer, id)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusCreated, envelope{"article": a})
}

func (s *server) updateArticle(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Article struct {
			Title       optional[string]   `json:"title"`
			Description optional[string]   `json:"description"`
			Body        optional[string]   `json:"body"`
			TagList     optional[[]string] `json:"tagList"`
		} `json:"article"`
	}
	if !s.decode(w, r, &req) {
		return
	}
	in := req.Article

	errs := validationErrors{}
	for field, v := range map[string]optional[string]{
		"title":       in.Title,
		"description": in.Description,
		"body":        in.Body,
	} {
		if v.Set && (v.Value == nil || blank(*v.Value)) {
			errs.add(field, "can't be blank")
		}
	}
	if in.TagList.Set && in.TagList.Value == nil {
		errs.add("tagList", "can't be null")
	}
	if len(errs) > 0 {
		s.writeErrors(w, r, http.StatusUnprocessableEntity, errs)
		return
	}

	ctx := r.Context()
	viewer, _ := userID(r)
	slug := r.PathValue("slug")
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	defer tx.Rollback()

	var id, authorID int64
	var title, description, body string
	err = tx.QueryRowContext(ctx,
		"SELECT id, author_id, title, description, body FROM articles WHERE slug = ?", slug,
	).Scan(&id, &authorID, &title, &description, &body)
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, r, http.StatusNotFound, "article", "not found")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if authorID != viewer {
		s.writeError(w, r, http.StatusForbidden, "article", "forbidden")
		return
	}

	if in.Description.Set {
		description = *in.Description.Value
	}
	if in.Body.Set {
		body = *in.Body.Value
	}
	now := nowMicros()
	save := func(slug string) error {
		_, err := tx.ExecContext(ctx, `
			UPDATE articles SET slug = ?, title = ?, description = ?, body = ?, updated_at = ?
			WHERE id = ?`,
			slug, title, description, body, now, id)
		return err
	}
	if in.Title.Set && *in.Title.Value != title {
		title = *in.Title.Value
		err = withUniqueSlug(title, save)
	} else {
		err = save(slug)
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if in.TagList.Set {
		if err := setTags(ctx, tx, id, *in.TagList.Value); err != nil {
			s.serverError(w, r, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		s.serverError(w, r, err)
		return
	}

	a, err := s.articleByID(ctx, viewer, id)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, envelope{"article": a})
}

func (s *server) deleteArticle(w http.ResponseWriter, r *http.Request) {
	viewer, _ := userID(r)
	id, authorID, err := s.findArticle(r.Context(), r.PathValue("slug"))
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, r, http.StatusNotFound, "article", "not found")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if authorID != viewer {
		s.writeError(w, r, http.StatusForbidden, "article", "forbidden")
		return
	}
	if _, err := s.db.ExecContext(r.Context(), "DELETE FROM articles WHERE id = ?", id); err != nil {
		s.serverError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) favorite(w http.ResponseWriter, r *http.Request) {
	s.setFavorite(w, r, true)
}

func (s *server) unfavorite(w http.ResponseWriter, r *http.Request) {
	s.setFavorite(w, r, false)
}

func (s *server) setFavorite(w http.ResponseWriter, r *http.Request, favorite bool) {
	viewer, _ := userID(r)
	id, _, err := s.findArticle(r.Context(), r.PathValue("slug"))
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, r, http.StatusNotFound, "article", "not found")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}

	query := "DELETE FROM favorites WHERE user_id = ? AND article_id = ?"
	if favorite {
		query = "INSERT INTO favorites (user_id, article_id) VALUES (?, ?) ON CONFLICT DO NOTHING"
	}
	if _, err := s.db.ExecContext(r.Context(), query, viewer, id); err != nil {
		s.serverError(w, r, err)
		return
	}

	a, err := s.articleByID(r.Context(), viewer, id)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, envelope{"article": a})
}

func (s *server) listTags(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(),
		"SELECT tag FROM article_tags GROUP BY tag ORDER BY COUNT(*) DESC, tag")
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	defer rows.Close()

	tags := []string{}
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			s.serverError(w, r, err)
			return
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.writeJSON(w, r, http.StatusOK, envelope{"tags": tags})
}

// setTags replaces the article's tags, dropping blanks and duplicates.
func setTags(ctx context.Context, tx *sql.Tx, articleID int64, tags []string) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM article_tags WHERE article_id = ?", articleID); err != nil {
		return err
	}
	seen := make(map[string]bool, len(tags))
	position := 0
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		_, err := tx.ExecContext(ctx,
			"INSERT INTO article_tags (article_id, position, tag) VALUES (?, ?, ?)",
			articleID, position, tag)
		if err != nil {
			return err
		}
		position++
	}
	return nil
}

// withUniqueSlug calls save with a slug made from title, adding a random
// suffix while the slug is taken. The slug "feed" is reserved by the
// GET /api/articles/feed route.
func withUniqueSlug(title string, save func(slug string) error) error {
	base := slugify(title)
	slug := base
	if slug == "feed" {
		slug = base + "-" + slugSuffix()
	}
	for range 5 {
		err := save(slug)
		if !isUniqueViolation(err, "articles.slug") {
			return err
		}
		slug = base + "-" + slugSuffix()
	}
	return fmt.Errorf("no free slug for title %q", title)
}

func slugify(title string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(title) {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			dash = true
			continue
		}
		if dash && b.Len() > 0 {
			b.WriteByte('-')
		}
		dash = false
		b.WriteRune(r)
	}
	if b.Len() == 0 {
		return "article"
	}
	return b.String()
}

func slugSuffix() string {
	return fmt.Sprintf("%06x", rand.N(1<<24))
}
