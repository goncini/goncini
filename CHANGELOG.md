# Changelog

## Unreleased

goncini requires Go 1.27, and effect-go v0.3.0, which pins the library and
the `ego` tool together.

### httpkernel

Symfony's HttpFoundation and HttpKernel in one package, on `net/http`:

- **Request helpers:**
  - typed query, path and header parameters;
  - JSON bodies whose errors are RFC 9457 problems pointing at the failing field;
  - content and language negotiation;
  - the client's address, scheme and host behind trusted proxies, and `BaseURL` for absolute URLs;
  - conditional requests and `Cache-Control`.
- **The kernel:**
  - typed endpoints (`Endpoint`) with argument binding and validation;
  - error mappers built from effect-go error sets;
  - a panic boundary;
  - a scope per request;
  - OpenTelemetry spans named after the route;
  - access logs;
  - a server that shuts down gracefully.

Worth knowing:

- The kernel answers requests that match no route with 404 and 405 problems,
  behind a `ServeMux` or a `routing.Router`.
- Binding errors on path, query and header values are 400s; validation
  errors (`Invalid`) are 422s.
- `BaseURL` takes the host from the request: without trusted proxies, that is
  the Host header the client sent. A URL that outlives the response should
  start with a configured base URL.
- encoding/json/v2 has no default form for `time.Duration`: use a string
  field, or an encoding of your own.
- **Cost per request (Apple M4 Pro):**
  - a hand-written handler takes 570 ns;
  - with `Endpoint` binding, 665 ns;
  - inside a `Kernel`, 1.13 µs.

### routing

Symfony's Routing component on `net/http`'s ServeMux, with no dependency
outside the standard library:

- **Registering:** `Get`, `Post`, `Put`, `Patch`, `Delete`, `Method` and
  `Handle` register routes, and `Name` names them.
  - `Group`, `NamePrefix` and `With` derive routers with a path prefix, a name
    prefix or middleware, sharing the mux and the names.
  - Controllers implement `Routes`, and `New` or `Include` registers them.
- **Exact matching:** routes match their path exactly. `/articles/` doesn't
  take every path below it, as a ServeMux pattern does; a subtree is
  `/files/{path...}`.
- **URLs:** `Route.URL` and `Router.URL` build URLs, escaping each value so
  that the route gets it back, with the other parameters in the query string.
  They refuse what would go astray, and their errors are the `URLError` set:
  - a `{name}` value of `/`;
  - dot segments, since RFC 3986 and browsers treat `%2E` as a dot;
  - URLs that a more specific route would serve (`ShadowedURL`).
- **Requirements,** as in Symfony: `/articles/{id<\d+>}` or
  `Require("id", routing.Digits)`. Routes with the same path are tried in
  order, so `/articles/{id<\d+>}` and `/articles/{slug}` can live together;
  a request that meets no requirement gets a 404, and URLs check them too.
  It costs about 7 ns per request.
- **Startup checks:** conflicting routes, duplicate names, and paths ServeMux
  can't route panic at startup, saying where each route was registered.
- **Listing:** `List`, `Match` and `WriteTable` list the routes for
  `debug:router`; `Endpoint` handlers name their function there.
- **Cost:** serving through a `Router` costs what the ServeMux does; building
  a URL takes about 500 ns.

### goncini, console, webtest

Running an app, in the [layout](docs/layout.md) of goncini apps:

- **`goncini.Main`** is an app's main function: it loads the environment and
  the config, builds the app with its effect-go `layer` injector in a scope
  that SIGINT and SIGTERM cancel, and runs a command.
- **Config is code.** An app's `config.Load` fills its `Config` for an `Env`:
  what every environment shares, then what the one named by `APP_ENV`
  changes. Environment variables only carry `APP_ENV` and secrets, read with
  `Env.Secret` from the process or from `.env` files, in Symfony's order. An
  unknown environment and every missing secret fail at boot, together.
- **Logging:** through `log/slog`, Go's PSR-3. `NewLogHandler` is the
  default handler, text or JSON on stderr; an app that logs with zap,
  zerolog, logrus, logr or OpenTelemetry provides that library's
  `slog.Handler` instead, and `NewLogger` still adds trace IDs to its
  records. [docs/logging.md](docs/logging.md) has a recipe for each,
  compiled against the libraries.
- **Providers:** `Framework` sets up the logger, router, kernel and server,
  configured by the `HTTP` and `Log` sections, whose zero values are the
  defaults; `HTTP` has the server's and the kernel's timeouts too. Logs carry
  trace IDs; the kernel sits behind the trusted proxies and hosts, with an
  access log. An app replaces any provider by passing its own to
  `layer.Build`: a `Renderer` for an error format of its own, its
  `[]httpkernel.Middleware`, a `Validator`.
- **`validator`** checks `validate` struct tags with
  go-playground/validator, and answers a 422 with a violation per failing
  value: a pointer into the body, or the parameter or header. It runs
  before an input's `Validate` method, which can rely on the tags.
- **Commands:** `console` runs an app's commands, which are services. A
  command with flags declares them in a `Flags` method, and console parses
  them. goncini adds `serve` (the default, with `-addr` for a busy port),
  `debug:router`, `debug:config`, which masks secrets, `list` and
  `help <command>`. A mistake in the command line, such as an unknown
  command or flag, is printed plainly and exits with status 2.
- **`webtest`:** `Boot` builds an app for a test, in the test environment,
  and closes its scope when the test ends; `Run` runs one of its commands.
  A `Client` sends requests in process and checks the responses, problems
  and violations included:
  `c.Post("/articles", body).Problem(422).Violation("#/title", "is required")`.
  `Isolate` runs a test in a database transaction that is rolled back, which
  the app's own transactions join, for databases that outlive a test, such
  as PostgreSQL. Everything works inside a `testing/synctest` bubble.
- **`httpkernel.TrustHosts`** answers 400 to requests for other hosts, so
  that `Host` and `BaseURL` can't return one the client made up.
- `examples/articles` is now its own module, laid out as `goncini new` will
  write apps.

### db

Databases, whatever library an app queries them with:

- **`db`** (standard library only): `Open` opens a `database/sql` pool from
  the `db.Config` section, without connecting, and closes it with the app.
  `SQL.InTx` runs a function in a transaction that `SQL.Conn` finds in its
  `ctx`, joining an outer one; `db.Conn` is sqlc's `DBTX`. `Migrator` is
  behind `db:migrate`, `db:migrate:status` and `db:rollback`. `db.All`,
  `db.One` and `db.Value` run a query and read its rows with the app's
  scan function, or its one value.
- **Checks:** an app's `[]goncini.Check`, such as `pool.PingContext`, run
  before `serve` listens. Nothing connects while the app is built, so
  `help`, `list` and `debug:*` work without the database.
- **Adapters,** each a module of its own so that goncini doesn't require
  their libraries:
  - `pgxdb`: pgx's native pool, with transactions in `ctx` and sqlc's pgx
    `DBTX`, tested against PostgreSQL;
  - `gormdb`: GORM on goncini's pool, with transactions in `ctx`, and
    `AutoMigrate` behind `db:migrate`;
  - `goosedb` and `migratedb`: goose and golang-migrate migrations, from
    embedded files.
- **`db.Isolator`**, which `SQL` and the pgx and GORM adapters implement, is
  what `webtest.Isolate` uses.
- `examples/articles` keeps its articles in SQLite, with goose migrations;
  its tests each get a database of their own.

### security

Symfony's Security component, the light version:

- **`Hasher`** hashes passwords with argon2id (OWASP's parameters by
  default, PHC strings), verifies bcrypt hashes too, and says when a hash
  `NeedsRehash`.
- **`Tokens`** issues and verifies JWTs signed with HMAC-SHA256, naming a
  user, from the `security.Config` section: a secret of 32 bytes or more,
  a TTL, an issuer.
- **`Firewall[U]`** authenticates requests with a token in their
  `Authorization` header, and loads their user; requests without one go
  through as anonymous, and an invalid token is a 401 even on a public
  route. `User[U]` and `CurrentUser[U]` give handlers the user, and
  `Required` guards routes.
- Its failures are the `AuthError` cases, 401 problems with a
  `WWW-Authenticate` header. For that, `httpkernel.Problem` gained a
  `Header` of response headers, which the kernel's 405 now uses for `Allow`.
- **Voters and roles (M2):** `Access` decides from voters' votes, with the
  affirmative, consensus or unanimous strategy, and grants `ROLE_*`
  attributes to users whose `Roles` include them, through
  `Config.RoleHierarchy`. `security.On` makes a voter of the subjects of one
  type; `Check` is Symfony's `denyAccessUnlessGranted`, a 401 or a 403;
  `Require` guards routes; `goncini generate` collects the voters.
- **Firewalls per route group (M2):** a firewall's middleware goes on a
  group with `routing.With`, and its `Tokens` is any `Verifier`: `Tokens`'
  JWTs, or `APIKeys`, the hashed keys of machine clients. RealWorld puts its
  users' firewall on `/api`, and a moderators' firewall of API keys on
  `/api/admin`, where they may delete any article: the `Authorship` voter
  lets them, as it lets authors edit and delete their own.

### openapi

NelmioApiDocBundle and API Platform's documentation: an OpenAPI 3.1
document generated from the code, with nothing to annotate.

- **`Generate`** describes the routes whose handlers are endpoints: the
  parameters and body from the input type, with the rules of its
  `validate` tags; the successful response from the output type, as
  json/v2 encodes it (201 for `Created`, 204 for `NoContent`); and a
  response for each error the endpoint can fail with, rendered by the
  kernel's mappers and renderer, with an example of each. RealWorld's
  error bodies come out as `{"errors": …}` because its renderer writes
  them, with nothing to declare.
- **Middleware describes itself** by returning a `Describer`: the firewall
  adds its security scheme, an optional token and the 401 for an invalid
  one; `security.Required` makes the token required and adds the 401
  without one.
- **A goncini app** serves its document at `/openapi.json`, a Swagger UI
  page where `openapi.Config.DocsPath` says (in dev, for RealWorld), and
  prints it with `openapi:dump`.
- **The contract is checked in tests.** `Document.Check` says how an
  exchange breaks the document: an undeclared status or media type, a
  body its schema doesn't allow, or an accepted request whose body it
  doesn't allow. A `webtest` client of an app, `webtest.NewClient(t, a)`,
  checks every response; `webtest.Contract` is the same as middleware, for
  an app served over HTTP. RealWorld's Hurl suite runs through it: all 154
  requests hold.
- **M2's contract gates pass.** RealWorld's document covers RealWorld's
  own `openapi.yml`: the same operations, parameters, request bodies and
  successful responses, with compatible schemas. A client that
  oapi-codegen generates from it drives every operation. Comparing them
  found that RealWorld's single `Article` type left its body optional:
  lists now return a `Summary`, without one.
- In a request, a struct member whose own members are required is
  required: without it, they would be missing.
- An input field that a nearer field of the same name hides, which
  binding would never fill, now panics when the route is registered.
- `httpkernel.Describe` says what an endpoint takes and returns, and
  `routing.Info` has the route's middleware.

### cors

NelmioCorsBundle's role: `cors.New(Config)` is middleware that answers
preflight requests and adds the CORS headers for the origins it allows,
exact or with a wildcard subdomain (`https://*.example.com`), with the
methods, headers, exposed headers, credentials and max age it says. A
goncini app sets it in `goncini.HTTP.CORS`; RealWorld allows any origin.

### ratelimit

Symfony's RateLimiter: a `Limiter` applies a policy to the state of each
key, which a `Store` keeps.

- **Policies:** `FixedWindow`, `SlidingWindow` and `TokenBucket`. Taking
  tokens returns a `Result` with what's left, when to retry, and when the
  limit is whole again; `Err` makes it an `Exceeded`, a 429 with
  `Retry-After`.
- **Stores:** `MemoryStore`, the default, for one process; the `Store`
  interface is what shared stores implement.
- **`Middleware`** limits requests by a key, such as `ByIP`, sets the
  `RateLimit-*` headers, and documents its 429 in the OpenAPI document.
- RealWorld throttles logins as Symfony's security does: 5 a minute per
  email and client address, which a successful login gives back. Endpoints
  get the client's address with `httpkernel.ClientIPOf(ctx)`.

### uid

Symfony's Uid component: `uid.UUID`, version 4 or 7, time-ordered even
within a millisecond, and `uid.ULID`, which convert to each other. Both
bind from requests, scan from text or 16 bytes, and describe themselves in
OpenAPI documents; `routing.ULID` joins `routing.UUID`. The articles
example gives its articles UUIDs, with a migration that gives the articles
there are some.

### listing

API Platform's pagination and order filter: `Window` binds
`?limit=&offset=`, `Page[T]` writes a page with RFC 8288 `Link` headers to
the first, previous, next and last pages, `Cursor[K]` pages by opaque keys,
and `Sort[O]` binds `?sort=-createdAt,title` against the fields an `Order`
allows, for an `ORDER BY` that can't be injected. The articles example lists
with them.

### The RealWorld app

[`examples/realworld`](examples/realworld) is the RealWorld "Conduit" API,
in goncini's layout 2, on SQLite: users, profiles and follows; articles,
tags, favorites, the feed and comments. It passes RealWorld's official Hurl
suite, 13 files and 154 requests, which `TestSpec` runs and CI requires.
It is 1,192 lines of `.ego`, plus a 58-line migration.

What it needed, now in goncini:

- `httpkernel.Optional[T]`, for updates that tell an absent member from
  `null` and from a value;
- `validator.Message`, to change a rule's message, such as `required`'s
  to "can't be blank";
- `routing.Prefix`, to register controllers under `/api`.

RealWorld's error format, `{"errors": {"title": ["can't be blank"]}}`, is
a `Renderer` of its own, and security's 401s are mapped to it: both replace
goncini's defaults with providers passed to `layer.Build`.

M1's gates:

- RealWorld's suite passes.
- A case added to an error set without a response fails the build:
  `match on UserError doesn't handle Suspended`.
- When the app is stopped during a slow request, the request completes,
  its database is released after it, and no goroutine is left
  (`TestShutdownDuringASlowRequest`). It runs on a real network, which
  `testing/synctest` can't fake, rather than under synctest as planned.
- The same API written with plain `net/http` and hand wiring,
  [examples/realworld-plain](examples/realworld-plain), passes the same
  suite. Without blank lines and comments, goncini's articles feature is
  34% shorter (421 lines against 640), the whole app 34% (874 against
  1,333), the users feature 6%. Half of each feature is SQL in both: the
  articles feature was 29% shorter until `db.All`, `db.One` and `db.Value`
  took the rows' boilerplate.
- An agent given only [AGENTS.md](AGENTS.md) added an endpoint with
  validation and a new error case, and hidden tests passed at the first
  attempt; its feedback went into AGENTS.md. The task and the tests are
  in `examples/realworld/testdata/agent`, to run again.

### goncini generate

`go tool goncini generate` runs `ego generate`, and writes the app's
`app/autoconfigured_gen.go`: the `Autoconfigured` providers, which hand
goncini the app's services of each kind it uses, found by their types, as
Symfony's autoconfiguration does:

- the services with a `Routes` method, in the order of `Services`;
- every exported `httpkernel.ErrorMapper` variable of the app's packages;
- the services that are `console.Command`s, and the migrator's commands;
- the services with `Ping` or `PingContext`, as `Check`s before serving;
- a provider per field of the config, such as `goncini.HTTP`.

- `autoOpenAPI`: the doc comments of the endpoints and of the types they
  take and return, and the error cases each endpoint can fail with. It
  finds them by following the endpoint's calls through the module's
  packages and goncini's: the cases it builds, such as `fail
  NoArticle{…}` or `check … as Unavailable`, of the error sets that a
  mapper maps, or that answer for themselves. `GET /articles/{slug}`
  documents 404 and 503, not the 403 of the set's `NotAuthor`.
- a default for each of goncini's config sections that the config
  doesn't have, such as `openapi.Config`.

A kind that the app provides itself is left to it: RealWorld provides its
routes, under `/api`. An app is generated from nothing in one run, and
`-check` fails when the file isn't up to date, which CI checks.

It also describes the services that the app's layers build, in order,
with their providers and what they need, which `debug:container` lists.

### Found while building

- **M2:** the repository's `.gitignore` ignored `*.test`, Go's test
  binaries, and so the examples' `.env.test` files: CI ran their tests with
  the dev `.env`, on a database file. They are committed now.

- **Reviews:** independent reviews found 15 defects in `httpkernel` and 9 in
  `routing`, each reproduced and fixed with a regression test. Among them:
  - problems that failed to encode were sent as an empty 200;
  - a request past its deadline got an empty 200 instead of a 504;
  - clients could spoof their address through a `Forwarded` header;
  - a `{name}` value of `/` gave a URL that no route served;
  - `URL` could return a URL that another route served.
- **effect-go issues, all fixed upstream:**
  - v0.2.0:
    - `scope.Run` lost the handler's panic when a fiber nobody joined had panicked too;
    - `StopTimeout` cost a goroutine and a timer per call;
    - a pointer to an error case didn't match as the case;
    - external test packages written in `.ego` failed to generate.
  - v0.2.1:
    - comments on error-set cases didn't reach the generated Go;
    - documented declarations after an error set got wrong `//line` positions,
      so stack traces pointed up to 16 lines off.
  - v0.2.2:
    - `layer` only read the `layer.Set`s declared in the injector's own
      package, so goncini couldn't export `Framework`;
    - `ego generate ./...` never finished on a fresh app whose packages
      import each other's `.ego` code;
    - an external test package that imports a package depending on the
      package under test failed to type-check, though `go test` accepts it.
  - v0.3.0:
    - `check f() as Case` wrapped errors that already were cases of the set,
      so a case returned through a callback, such as a transaction's, lost
      its response.
