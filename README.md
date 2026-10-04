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
- **Structured concurrency.** Each request runs in its own effect-go scope,
  and the server shuts down gracefully.

Status: early. [`httpkernel`](httpkernel) is implemented; routing comes next.
The plan is in [docs/assessment.md](docs/assessment.md).

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
	mux := http.NewServeMux()
	mux.Handle("GET /articles/{slug}", httpkernel.Endpoint(articles.Show))
	kernel := &httpkernel.Kernel{
		Handler:      mux,
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

The full example is [examples/articles](examples/articles): `go run ./examples/articles`.

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
