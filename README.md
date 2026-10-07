# goncini

A framework for building APIs in Go: Symfony's architecture, adapted to Go,
its ecosystem, and [effect-go](https://github.com/effect-go/effect-go).

- **Plain `net/http`.** Handlers are `http.Handler`s, middleware is
  `func(http.Handler) http.Handler`, and every package works on its own.
- **Errors are values with a response each.** An effect-go error set is
  mapped to responses by an exhaustive `match`: a new case without a
  response doesn't compile.
- **Typed endpoints.** An `effect` method takes its input struct and returns
  its result; goncini binds the path, query, headers and JSON body, validates,
  and writes JSON or an RFC 9457 problem.
- **Named routes on `ServeMux`.** Routes are registered in plain Go, in
  groups with their own middleware, and URLs are built from route names.
- **Every setting has a default, and every default can be changed.** A new
  app works without writing any config; an app that needs another error body
  or request size limit changes that one default, in Go, and keeps the others.
- **Structured concurrency.** Each request runs in its own effect-go scope,
  and the server shuts down gracefully.

Status: early. [`httpkernel`](httpkernel), [`routing`](routing),
[`validator`](validator), config as code, the [`console`](console),
`goncini.Main` and [`db`](db), with adapters for pgx, GORM, goose and
golang-migrate, and [`webtest`](webtest) are implemented
([CHANGELOG.md](CHANGELOG.md)); `security` comes next. The plan is in
[docs/assessment.md](docs/assessment.md), and the layout of an app in
[docs/layout.md](docs/layout.md).

## A taste

```go
error ArticleError {
	NotFound{ Slug string } "no article {Slug}"
	Duplicate{ Slug string } "an article {Slug} already exists"
}

func Problem(err ArticleError) httpkernel.Problem {
	return match err {
		nil          => httpkernel.Problem{}
		NotFound(e)  => httpkernel.Problem{Status: http.StatusNotFound, Detail: e.Error()}
		Duplicate(e) => httpkernel.Problem{Status: http.StatusConflict, Detail: e.Error()}
	}
}

var Problems = httpkernel.Map[ArticleError](Problem)

type ShowInput struct {
	Slug string `path:"slug"`
}

effect (a *Articles) Show(in ShowInput) (Article, ArticleError) {
	art, ok := a.bySlug[in.Slug]
	if !ok {
		fail NotFound{Slug: in.Slug}
	}
	return art, nil
}

func main() {
	articles := &Articles{bySlug: map[string]Article{}}
	router := routing.New()
	router.Get("/articles/{slug}", httpkernel.Endpoint(articles.Show)).Name("article_show")
	kernel := &httpkernel.Kernel{
		Handler:      router,
		ErrorMappers: []httpkernel.ErrorMapper{Problems},
	}
	srv := &httpkernel.Server{Addr: ":8080", Handler: kernel}
	scope.Main(s => srv.ListenAndServe(s.Context()))
}
```

```console
$ curl localhost:8080/articles/nope
{"title":"Not Found","status":404,"detail":"no article nope"}
```

Handlers build URLs from route names, escaped:
`router.URL("article_show", routing.Params{"slug": "café"})` gives
`/articles/caf%C3%A9`.

The full example is [examples/articles](examples/articles), an app in
goncini's layout: in its directory, `go run .` serves it, `go run . list`
lists its commands, and `go run . debug:router` its routes.

## Developing

goncini is written in effect-go's `.ego` dialect, with the generated Go
committed, so using it never requires `ego`. The `ego` tool is pinned in
`go.mod`:

```bash
go tool ego generate ./...   # after editing a .ego file
go test -race ./...
```

Go 1.27 or later.

MIT licensed.
