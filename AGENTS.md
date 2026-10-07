# Building APIs with goncini

goncini is a framework for JSON APIs in Go, after Symfony, on `net/http`
and effect-go. Apps are written in effect-go's `.ego` dialect. This file
is what you need to work on a goncini app; `examples/realworld` is a
complete one.

## Commands

```bash
go tool ego generate ./...   # after editing a .ego file: writes x_ego.go next to x.ego
go test ./...
go run . list                # the app's commands; go run . serves it
go run . db:migrate          # applies the migrations
go run . debug:router        # lists the routes
```

Edit `.ego` files only, never the generated `_ego.go` or `layers_ego.go`,
and run `go tool ego generate ./...` before building or testing.

## Layout

```
main.ego              goncini.Main(config.Load, app.Build)
.env, .env.test       APP_ENV and secrets for local work and tests
config/               type Config, Load(env), and one file per environment
app/                  the wiring: inject.go, services.ego, autoconfigured.ego
migrations/           goose SQL files, embedded
<feature>/            one package per feature, such as articles/:
  <feature>.ego         types, the error set and its problems
  controller.ego        Routes, inputs and endpoints
  store.ego             queries
```

## The `.ego` dialect

`.ego` is Go plus these. Everything else is Go.

| Write | Means |
|---|---|
| `x := check f()` | if `f` fails, return its error, wrapped as `"f: <err>"` |
| `x := check f() ""` | the same, unwrapped |
| `x := check f() as Unavailable` | the same, as the case `Unavailable{Cause: err}` of the function's error set |
| `check err ""` | return `err` if it isn't nil |
| `fail NotFound{Slug: s}` | return this error, with zero values for the other results |
| `x := must f()` | panic if `f` fails: for what can't fail unless the code is wrong |
| `x := f() else fallback` | use `fallback` if `f` fails |
| `match v { A(e) => …; _ => … }` | an exhaustive switch on an error set's cases, or values |
| `(x int) => x * 2`, `x => x.Name` | a function literal with an expression body; write the parameter types where the call doesn't give them. A literal with a block body and results is a plain `func(x int) int { … }` |
| `if ok { a } else { b }` | a conditional value |
| `f"{n} items, {name:q}"` | `fmt.Sprintf` |
| `a ?? b` | `b` when `a` is nil, a missing map key, or a failed type assertion |
| `u?.Name` | the zero value when `u` is nil |

Rules that the compiler enforces:

- `check` and `must` start a statement or the right side of `:=` or `=`
  (`n := check res.RowsAffected()`, `c, ok = check s.Get(id)`). Not
  `return check f()`, not in an `if` header, not inside an expression:
  assign first.
- A function that returns an error set returns only its cases: each
  `check` there needs `as Case`, unless the callee returns the same set;
  `return` gives `nil`, a case, or a call that returns the set.
- An `effect` method gets a `ctx context.Context` first parameter and a
  span: `effect (s *Store) Get(slug string) (Article, error)` compiles to
  `func (s *Store) Get(ctx context.Context, slug string) (Article, error)`.
  Inside, `ctx` is in scope, and calls to other effect methods pass it.

## Errors

Each feature declares what it fails with as an error set, and says what
each case looks like over HTTP with an exhaustive `match`. Adding a case
without an arm doesn't compile.

```go
error ArticleError {
	NoArticle{ Slug string } "no article {Slug}"
	NotAuthor{ Slug string } "not the author of {Slug}"
	Unavailable{ Cause error } "the articles are unavailable"
}

func Problem(err ArticleError) httpkernel.Problem {
	return match err {
		nil            => httpkernel.Problem{}
		NoArticle(_)   => conduit.Problem(http.StatusNotFound, "article", "not found")
		NotAuthor(_)   => conduit.Problem(http.StatusForbidden, "article", "forbidden")
		Unavailable(_) => httpkernel.Problem{Status: http.StatusServiceUnavailable}
	}
}

var Problems = httpkernel.Map[ArticleError](Problem)
```

Each case has a doc comment when its name doesn't say it all. `Problems`
is registered in `app/autoconfigured.ego`'s `problems`. Store failures
become the `Unavailable` case: `check s.store.Get(slug) as Unavailable`.
A case that a callback returns, such as a transaction's, passes through
`as Unavailable` unchanged.

## Endpoints

An endpoint is an effect method `(In) (Out, ErrorSet)`, served by
`httpkernel.Endpoint`. Its input struct says where values come from:

```go
type UpdateInput struct {
	Slug  string `path:"slug"`
	Draft bool   `query:"draft"`                 // query:"name,required", default:"20", enum:"a,b"
	Token string `header:"X-Token"`
	Body  struct {                               // the JSON body
		Article struct {
			Title string `json:"title" validate:"required,max=120"`
		} `json:"article"`
	}
}
```

- `validate` tags are go-playground/validator rules. A failing value is a
  422 that points at it. Rules across fields, or on the app's data, go in a
  `Validate() error` method on the input's pointer, which runs after the
  tags and returns `httpkernel.Invalid(httpkernel.Violation{Pointer:
  "#/article/title", Detail: "can't be blank"})`.
- `httpkernel.Optional[T]` tells an absent member from `null` and from a
  value, for updates: `.Set`, `.Null`, `.Value`, `.Present()`.
- The result is written as JSON with 200. `httpkernel.Created[T]{Body: b}`
  answers 201, and `httpkernel.NoContent{}` answers 204.
- An input with no values is `struct{}`.

A request is checked in this order, each step answering for itself: the
route's middleware, such as `security.Required` (401); binding the path,
query and headers (400) and the JSON body (400, 413, 415, 422); the
`validate` tags, then `Validate()` (422); then the endpoint, with its own
404s and 403s.

## Routes

A controller registers its endpoints:

```go
func (a *Articles) Routes(r *routing.Router) {
	r.Get("/articles/{slug}", httpkernel.Endpoint(a.Show)).Name("article_show")
	auth := r.With(security.Required) // routes that need a user
	auth.Put("/articles/{slug}", httpkernel.Endpoint(a.Update)).Name("article_update")
	auth.Delete("/articles/{slug}/comments/{id<\\d+>}", httpkernel.Endpoint(a.DeleteComment))
}
```

Paths match exactly. `{id<\d+>}` requires a value to match a regular
expression, or the route doesn't match. A controller is registered in
`app/autoconfigured.ego`'s `routes`.

## Users

`security.CurrentUser[*users.User](ctx)` returns the authenticated user,
or a 401 error; `security.User[*users.User](ctx)` reports whether there is
one. In the RealWorld app, `users.Current(ctx)` and `users.Viewer(ctx)`
(an ID, 0 for anonymous) wrap them.

## Database

Stores query through `db.SQL`:

```go
effect (s *Store) Get(id int64) (Comment, bool, error) {
	row := s.sql.Conn(ctx).QueryRowContext(ctx, "SELECT … WHERE id = ?", id)
	…
}
```

`s.sql.InTx(ctx, func(ctx context.Context) error { … })` runs a function in
a transaction, which the store calls made with that `ctx` take part in. A
check and a write that must not be separated can also be one statement,
such as `UPDATE … WHERE id = ? AND created_at >= ?`, whose
`res.RowsAffected()` says whether it matched. A
schema change is a new goose file in `migrations/`, such as
`00002_add_comment_edits.sql`, with `-- +goose Up` and `-- +goose Down`
sections; never edit an applied one.

## Wiring

`app/inject.go` builds the app with effect-go's `layer`: providers are
plain constructors, matched by their result types.

- `app/services.ego`'s `Services` lists the app's constructors, such as
  `articles.NewStore`: add a new service there.
- `app/autoconfigured.ego` hands goncini the routes, error mappers,
  commands and config sections.
- A service that needs the time takes a `now func() time.Time`, which the
  app's `clock` provider gives: `time.Now`. Tests change time with
  `testing/synctest`, which fakes `time.Now` and `time.Sleep`.

## Tests

```go
a := webtest.Boot(t, config.Load, app.Build) // the app in the test environment
webtest.Run(t, a, "db:migrate")              // a new in-memory database
c := webtest.NewClient(t, a.Kernel)          // a *webtest.Client
c.Post("/api/articles", `{"article":{…}}`).Status(201)
c.Put(path, body).Status(200).JSON(&out)     // decodes the JSON body into out
c.Get("/api/articles/nope").Status(404).Contains(`"article":["not found"]`) // a substring of the body
var ann *webtest.Client = c.WithHeader("Authorization", "Token "+token)
```

Each check reports its failure and returns the response, for more checks.
`Problem(status)` checks an RFC 9457 problem+json response, and
`.Violation(where, detail)` one of its violations; an app with an error
format of its own, such as RealWorld's, is checked with `Status` and
`Contains`.

`webtest.Boot` and the client work inside a `testing/synctest` bubble,
where time is fake and `time.Sleep(time.Hour)` takes no time:

```go
synctest.Test(t, (t *testing.T) => {
	a := webtest.Boot(t, config.Load, app.Build)
	…
	time.Sleep(16 * time.Minute)
	…
})
```
