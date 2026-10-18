# Changelog

## Unreleased

What [dogfooding](docs/dogfood.md) a new app, `examples/heartbeat`, found.
It is a cron-job monitoring API, after Healthchecks.io, made with `goncini
new` and the makers.

**Breaking:** `security.Verifier.Verify` takes a `ctx` first:
`Verify(ctx, token)`. A verifier of the app's own adds it.

- **security:** a `Firewall` without `Tokens` hands the token to `Load` as
  is: an API key that the app keeps in its database, by its hash, and
  looks up there. An unknown key is a 401, "it isn't one of the API's
  keys".
- **httpkernel:** a `Body []byte` or `Body string` field of an input is
  the raw body, of any type, up to the kernel's `BodyLimit`: OpenAPI
  describes it as `*/*`.
- **openapi:** the contract check matches a response's media type against
  the document's ranges, `text/*` and `*/*`.
- **validator:** the `validate` tags of an `httpkernel.Optional` field
  check its value when it is present, and `omitempty` skips it when it is
  absent or null. They panicked, a 500.
- **scheduler:** `scheduler:trigger <task>…`, and `Scheduler.Trigger`, run
  a task now, once, as by hand or in a test, while no other instance runs
  it. `scheduler.Tick` is then the time it was triggered.
- **The goncini tool:**
  - `goncini new -replace` with a relative checkout keeps it relative, from
    the app, so that `go.mod` can be committed.
  - The new app's `.gitignore` ignores the binary that `go build` writes.
  - The makers keep ego's keywords, such as `check`, out of their
    variables' names, and their docs name the type: "a check", "an
    article".
  - A maker's import joins the group of imports that shares the most of
    its path.
  - When an import of a `.ego` file isn't in `go.mod`, as when an app
    first imports a goncini package with dependencies of its own, the
    error says that `go tool goncini tidy` adds it.
- **security/oidc:** an `oidc.Verifier`, a firewall's `Tokens`, verifies
  the tokens of an OpenID Connect provider, as Symfony's OIDC token
  handler does. It discovers the provider's keys from its issuer, or takes
  their JWKS's URL, and fetches them when they are an hour old, or a token
  names another, once a minute at most; while the provider is down, the
  keys it had serve, and a verifier that never had them answers 503. A
  token must be signed by one of them with an asymmetric algorithm (RSA,
  RSA-PSS, ECDSA or Ed25519; HMAC and none are refused), name the issuer
  and the API's audience, and not have expired, within a minute's leeway.
  Its `sub`, or another claim, names the user. `oidctest.NewProvider(t)`
  is a provider for an app's tests, which issues tokens with
  `Token(audience, subject, claims)`; its connections close after each
  response, so that a synctest bubble's time moves while it is up.
- **security:** `security.Claims(ctx)` returns the claims of the token that
  authenticated the request, which a verifier records with
  `security.SetClaims`, as oidc's does: `Load` reads them to create the
  user of a first request.
- **redis**, a new module, keeps messages and locks in Redis:
  `redis.Messenger` is a bus whose queues are in Redis, and `redis.Lock` a
  lock store, as the scheduler's, which share a `redis.Client` from a
  `redis.Config`. A queue is three keys in one hash slot, for Redis
  Cluster, which scripts change at once: a message's lease is the time it
  is due again. The URL `memory://` keeps them in the app's memory, for
  tests, as Symfony's `in-memory://` transport does. A message dispatched
  in a database transaction is sent at once, unlike to `messenger.SQL`'s
  queues.
- **lock/locktest** checks a `lock.Store`, as `transporttest` checks a
  transport: the memory, SQL, PostgreSQL and Redis stores pass it.
- **messenger:** `Config.TransportNames` returns the names of the
  transports that a bus of the config has, for a transport adapter.
- **The goncini tool:** `goncini generate` counts a function listed in
  several sets once, as `layer` does: a client in two of a module's sets
  was pinged twice before serving. `goncini tidy` works in a module
  without an app, as a bundle's.
- **examples/heartbeat**, the new example, has projects whose API keys
  authenticate them, checks in a workflow, public pings with raw bodies
  and a rate limit, a scheduled sweep, alerts by email and webhook through
  Messenger, and a cached badge.

## v0.2.0 (2026-10-16)

Every module is tagged v0.2.0, as for v0.1.0. To upgrade an app, require
`github.com/goncini/goncini v0.2.0` and its adapters at v0.2.0, then run
`go tool goncini tidy`. The tool's new commands, `serve`, `check` and the
app's own, need it: an app on v0.1.0 runs v0.1.0's tool.

goncini requires effect-go v0.4.0, and its generated code is generated
again with it. What changes for apps:

- `a ?? b` also falls back when `a` is a nil map value or a nil result of
  a type assertion. goncini's own uses, such as the OpenAPI document's
  responses, behave as before, or fall back where a nil entry would have
  gone through.
- `goncini.Main`, through `scope.Main`, exits with status 1 when shutting
  down fails, as when a resource fails to close, even after a clean stop
  by signal.
- An app runs `go tool goncini generate` after upgrading: the Go that ego
  generates changes.

### The goncini tool, as the symfony command

- **The app's commands:** in an app, `go tool goncini list`,
  `debug:router`, `db:migrate` and the others run the app's commands.
  - It runs them from a binary in the user's cache, which `go build` links
    again only when the code changed, unlike `go run .`.
  - It generates the app first when a `.ego` file or `go.mod` is newer
    than the wiring, whose modification time is when it was last
    generated, so a command never runs stale Go.
  - It works from any directory of the app.
  - Inside an app, `goncini list` lists the app's commands, then the
    tool's.
- **`goncini serve`:** serves the app while it is developed, as `symfony
  server:start` does.
  - It listens on the config's address, which the app's `serve
    -print-addr` reports, in front of the app.
  - When the app's files change (its `.ego`, Go, `go.mod`, `.env` and
    embedded templates, but not its tests, nor its migrations, which
    `db:migrate` applies), it generates, rebuilds and restarts the app.
  - Requests made meanwhile wait for the app.
  - What doesn't compile, or why the app exited, is the answer until it is
    fixed: a problem, or text to a browser.
- **`goncini check`:** as `symfony check:security`, it checks that the Go
  of the `.ego` files and the wiring are up to date, and runs govulncheck
  v1.9.0.
- **`goncini` in the path:** installed with `go install`, it runs the app's
  own, `go tool goncini`, when the app requires another version or replaces
  goncini with another directory, so the code it generates is for the
  goncini the app builds with.
- **Removed `.ego` files:** with effect-go v0.3.5, `ego generate` removes
  the Go it wrote from a `.ego` file that is gone, and `-check` fails on
  it: the removed code compiled on. The tool counts a removed `.ego` file
  as a change, so the app's commands and `serve` generate again.
- **No more `-app`:** the wiring is in `app/`, which is part of the layout;
  `generate` and `tidy` no longer take the flag.
- **Faster `goncini generate`:** it first runs `ego generate` alone, and
  stops there when the wiring is already what it would write, as after most
  edits: about 3s instead of 6 on a new app.
- **Docs:** AGENTS.md, the skeleton and the examples use `go tool goncini`
  instead of `go run .`.

## v0.1.0 (2026-10-13)

The first release: every module is tagged v0.1.0, the root module as
`v0.1.0` and each adapter module under its directory, such as
`db/goosedb/v0.1.0`, so that `go run
github.com/goncini/goncini/cmd/goncini@v0.1.0 new example.com/acme` writes
an app that requires them.

goncini requires Go 1.27, and effect-go v0.3.4, which pins the library and
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

### event

Symfony's EventDispatcher, with typed events: `event.On(d, func(ctx,
*ArticlePublished) error)` adds a listener, with an optional priority, and
`event.Dispatch(ctx, d, &ArticlePublished{…})` calls them in order, until
one fails or returns `event.Stop`. A `Subscriber` registers listeners, and
`goncini generate` collects the app's subscribers into its dispatcher;
`debug:event-dispatcher` lists the listeners.

The kernel dispatches its own events: `RequestEvent` before the
middleware, whose listener's error answers the request (a maintenance
503); `ErrorEvent` before a problem is rendered, whose listeners may
change it; and `ResponseEvent` once the response is written, for metrics
and audit logs.

### lock

Symfony's Lock component: a `Factory` makes locks of a key, each with an
owner of its own, which `Acquire` without waiting, `Wait` for, `Refresh`
and `Release`; `Run` holds one while a function runs, refreshing it, and
cancels the function if the lock is lost. Locks expire after their TTL, so
that a dead process holds none. `MemoryStore` keeps them in a process, and
`SQLStore` in a table of a SQLite or PostgreSQL database, created when
first needed, with one upsert per acquisition. `db.DialectOf` tells the
two apart by their driver, and `Dialect.Rebind` numbers placeholders for
PostgreSQL.

### messenger

Symfony's Messenger, on effect-go:

- **The bus:** `messenger.Handle(bus, handler)` and `messenger.Route[M](bus,
  "async")` register typed handlers and routes; `messenger.Dispatch(ctx, bus,
  msg)` handles a message now, or encodes it as JSON and sends it to its
  transport, with an optional `Delay`.
- **Transports:** `MemoryTransport`, and `SQLTransport`, a queue in a table
  of a SQLite or PostgreSQL database. A worker leases each message it
  receives, and extends the lease while it handles it: a worker that dies
  loses the message to another when the lease runs out. One statement
  leases the next message, with `FOR UPDATE SKIP LOCKED` on PostgreSQL, and
  acknowledgements and retries carry the lease's token, so that a lost
  lease can't touch the message. `transporttest` checks a transport, as it
  checks these two, on SQLite and PostgreSQL.
- **Workers:** `messenger:consume` runs a `Worker`, whose handlers are
  effect-go fibers. A failing message is retried on an effect-go
  `Schedule`, three times with exponential backoff by default, then goes to
  the failure transport, which `messenger:failed:show`, `retry` and
  `remove` deal with. Told to stop, a worker takes no new message and lets
  its handlers finish, up to its stop timeout; those it cuts short go back
  to their transport, as they were.
- `goncini generate` collects the services that register handlers, and
  adds the bus's commands.

### scheduler

Symfony's Scheduler: tasks run `Every(d)`, on the multiples of d, or as a
`Cron` expression says, in UTC or a location. With a lock store, each tick
runs in one instance of an app: the one that takes the tick's lock.
`scheduler:run` runs the tasks, and `debug:scheduler` lists them with their
next runs. Lock stores now prune expired locks, which keys used once, as
ticks are, would leave.

### profiler and tracing (M5)

Symfony's WebProfilerBundle, built from spans:

- **Tracing:**
  - `goncini.SpanProcessor` is a kind that `goncini generate` collects: an
    OpenTelemetry span processor that says whether it is enabled;
  - `NewTracing` makes a tracer provider of the enabled ones, the global one
    until the app stops, with W3C trace context as the propagator. When no
    processor wants spans, there is none, and spans cost next to nothing;
  - the app's logger adds each record to its span, as a `log` event;
  - `db.SQL.Conn` makes a span of each statement, with its text, under the
    caller's span.
- **The profiler:** `profiler.Layer`, with `profiler.Config.Enabled` in dev:
  - it makes a profile of each trace whose root span ends in the process: a
    request, or a message's handling;
  - it keeps the last 200 profiles as JSON files in `var/profiler`;
  - `/_profiler` lists them, and `/_profiler/{id}` shows one. These are plain
    handlers, which the OpenAPI document leaves out;
  - `debug:requests` prints them, and `debug:requests <id>` prints one's
    spans, queries and logs, from another terminal.
- **OTLP export:** the `otel/otlp` adapter module, `otlp.Layer`, sends the
  spans to a collector over OTLP/HTTP when `otlp.Config.Endpoint` is set,
  in batches that the app's stop flushes. It is a module of its own, because
  the exporter brings protobuf and gRPC. The articles example exports to
  `OTEL_EXPORTER_OTLP_ENDPOINT` when it is set.

Worth knowing:

- Spans get the records that are logged, at the log's level.
- Query spans follow the tracer provider of their parent span, and end when
  the query returns its rows, before they are read. They are named after
  the statement's first word, past its comments, such as sqlc's.
- A profile's ID is its root span's. Requests of a trace that started
  elsewhere, which share its ID, get a profile each, of their own spans.
- A relative `profiler.Config.Dir` is the project directory's, so that
  `debug:requests` reads it from anywhere in the project.
- `/_profiler` has no authentication: enable the profiler in dev only.
- The tracer provider is also OTel's global one, which the spans that start
  a trace use, such as a request's and a message handling's; since
  effect-go v0.3.3, the spans under them follow their parent's provider.
  When the app stops, its processors get 5 seconds to flush, and the
  globals are restored unless another app in the process has set them
  since.
- `otlp.Config.Endpoint` is a base URL, as `OTEL_EXPORTER_OTLP_ENDPOINT`
  is: the spans go to its `/v1/traces`, unless it has a path.

M5's review fixed:

- spans exported to the collector's `/`, not to `/v1/traces`;
- concurrent requests of one remote trace sharing one profile;
- `/_profiler` and `debug:requests` failing on a profile half written;
- a panic on a negative `Keep`;
- the pending traces' order that grew with each request;
- reflection on each recorded query, to name its database;
- query spans named `--` or `SELECT\n`;
- a profile that couldn't be saved, without a word: it is logged.

### goncini new and make:* (M5)

Symfony's `symfony new`, for an API:
`go run github.com/goncini/goncini/cmd/goncini@latest new example.com/acme`.

- It writes a module in goncini's layout: `main.ego`, `app/`, `config/` with
  dev, test and prod, `migrations/`, `.env`, `.env.test`, and a `home`
  feature whose `GET /` says the API is up, with its test.
- The app uses SQLite through goose, with the profiler on in dev.
- It requires the version of goncini that the tool was built from, or the
  latest when that isn't known. `-replace dir` requires a goncini checkout
  instead.
- It tidies the module and runs `goncini generate`. `go test ./...` passes,
  and `go run .` serves, with nothing to edit first.

Symfony's MakerBundle, as the tool's `make:*` commands, run in the app's
module: each writes `.ego` files in goncini's layout, adds their services to
`app/services.ego`, and runs `goncini generate`, so that what it writes is
wired, and its tests pass, at once. If what it wrote doesn't compile, it
puts the app back as it was. It refuses names that would clash with the
app's, the feature's or its own, and prints the files it wrote.

- `make:feature notes` writes the `notes` package: a `Note`, its error set
  and its problems, a store, endpoints to list, show, create and delete
  notes, their test, and the table's migration. `make:feature people Person`
  names the type when the singular isn't the name without its `s`.
- `make:endpoint [-method POST] [-path /notes/{id}/archive] notes Archive`
  adds an endpoint, its input with the path's parameters, its output, and
  its route, named as the feature's others, to a feature's controller. An
  `{id}` is a `uid.UUID` with its requirement when the controller uses
  them.
- `make:message notes NoteAdded` writes a message and its handler, routed to
  the `async` queue. An app with no bus gets `messenger.SQL`.
- `make:command -summary text notes:prune` writes a console command.
- `make:migration add_note_tags` writes the next goose migration.

`goncini tidy` is `go mod tidy` for `.ego` files, which `go mod tidy`
doesn't read: it missed a module that a `.ego` file imported, which
`goncini generate` then couldn't compile, and dropped one that only `.ego`
files used. It tidies the module with the `.ego` files' imports standing in
as a package of blank imports, generates the app, and tidies it again
without them. `goncini generate` says to run it when a module is missing,
and puts `autoconfigured_gen.go` back as it was when it fails, rather than
leave it half written.

Worth knowing:

- `goncini generate` now finds the app's package through symlinks, such as
  macOS's `/var`.
- The makers edit `app/services.ego` as text: they add to a
  `var Services = layer.Set(…)`, and say what to add when it isn't one.
  `make:message` adds `messenger.SQL` unless the file mentions a bus.
- A new app gets goncini's AGENTS.md. `goncini new` checks the module path,
  names the app after it without its major version, and removes what it
  wrote if it fails.
- `goncini new` requires the version it was built from, which goncini's
  modules share; a build from a checkout, or a replaced one, requires the
  latest.
- A feature's queries take `?` parameters, as SQLite, a new app's database,
  does; an app on PostgreSQL writes `$1`.

### Bundles (M5)

Symfony's bundles, as packages: a bundle exports a `Layer`, a
`layer.Set` of its providers, and an app adds it to its `Services`. Nothing
else is wired by hand:

- `goncini generate` hands goncini the bundle's routes, commands,
  subscribers, message handlers, voters and the rest, as it does the app's.
- A bundle's `Config`, a struct type named `Config` in the package that
  exports the `Layer`, gets its zero value, its default, when a provider
  takes it and the app's config has no field of that type: a section in
  the app's config changes it. goncini's packages' `Config`s are defaulted
  the same way. `goncini generate` used to default only goncini's, from a
  list.
- Any other section that the config lacks, such as one of the app's own
  features, stays an error, which `layer` reports: a missing section
  isn't silently zero.
- [examples/maintenance](examples/maintenance), a module of its own as a
  third party's would be, takes an API down for maintenance: a subscriber
  answers requests with 503, `maintenance:down` and `maintenance:up` switch
  it, and `GET /_maintenance` says whether it is down. The articles example
  adds it with one line, and `TestBundle` checks each part.

Worth knowing:

- A bundle's `Config` must work as its zero value, as goncini's do. A
  config field of type `*maintenance.Config` isn't a section of
  `maintenance.Config`.
- A relative `maintenance.Config.File` is the project directory's, as
  `goncini.ProjectDir` finds it: commands run from anywhere in the project
  see the server's file. A deployed binary's is its working directory.
- `goncini generate` writes a stub `Autoconfigured` before it loads the
  app, so that a generated file that doesn't compile doesn't stop it.

### workflow (M4)

Symfony's Workflow, as state machines: a `Definition` has places, an
initial one and transitions, checked when `New` builds it. `Apply` moves a
subject from its state by a transition, refusing one its state doesn't
allow (a 409) or whose guard, a security attribute that the `Access`'s
voters and roles decide, isn't granted (401 or 403); it dispatches
`Transitioning`, whose listeners may refuse, and `Transitioned`. `Can` and
`Enabled` say what's possible, and `workflow:dump` draws a workflow as a
Mermaid state diagram. The articles example reviews its articles: authors
submit drafts, and editors, with API keys, publish or reject them: M4's
sixth gate.

### translation (M4)

Symfony's Translation, for an API's messages: a `Catalog` maps messages,
written in the app's language, to their translations, with no keys to
invent; a message with `{placeholders}` translates every message it
matches, so validators' and error sets' messages translate without
changing them. The `Translator` subscribes to the kernel's `ErrorEvent`
and translates problems' titles, details and violations into the language
that `Accept-Language` prefers, falling back from `fr-CA` to `fr`, then to
the message, and sets `Content-Language`. `translation:lint` lists the
messages each catalog lacks. The articles example speaks French: M4's
fifth gate.

### notifier (M4)

Symfony's Notifier: a `Notification`'s importance chooses its channels,
through the config's policy, and a `Recipient` may choose theirs. The
channels are email, through the mailer's transport, and chat webhooks that
take Slack's `{"text": …}`, through `httpclient`, so they're retried.
`notifier.Async` makes each delivery a Messenger message, retried on its
own. `Env.OptionalSecret` reads the secrets an app can do without, such as
a chat's webhook.

RealWorld tells an author by email when a moderator removes their article,
and the moderators' chat if `MODERATORS_CHAT_URL` is set, through its
queue: M4's fourth gate. `goncini generate` now wires the app's package
before the others, so that a layer's error, such as a dependency cycle,
shows instead of the import errors it causes.

### mime and mailer (M4)

Symfony's Mime and Mailer. `mime.Email` has its addresses, subject, text
and HTML versions, attachments, inline images and extra headers, and
`Bytes` writes it as an RFC 5322 message: multipart alternatives, related
parts and attachments, quoted-printable text, encoded non-ASCII headers,
no Bcc, and line breaks refused in addresses and headers. Bodies are the
app's strings: goncini has no templates.

`mailer.Mailer` sends emails through the transport its DSN names: SMTP,
with STARTTLS when offered or over TLS, `log://`, `memory://` or
`null://`. `mailer.Async` sends them as `SendEmail` messages that a worker
sends, so that a request doesn't wait for the mail server; `mailer.Sync`
sends them at once. `mailertest` runs an SMTP server in a test.
`messenger:consume -drain` stops once the queues are empty, for tests.

RealWorld welcomes registered users by email, through its queue: M4's
third gate.

### httpclient (M4)

Symfony's HttpClient, on `net/http`: a `Client` of an API, with a base URL
and default headers, retries on an effect-go schedule, and a span per
request that carries the trace context to the API called. It retries
connection failures, timeouts, 408, 425, 429 and 5xx but 501, of requests
whose method is idempotent or that carry an `Idempotency-Key`, waiting at
least what `Retry-After` asks, up to a cap. `Get` and `Send` encode and
decode JSON, and fail with a `StatusError` for a status that isn't 2xx.
`Handler` serves requests with an `http.Handler` in the process, for tests,
as Symfony's MockHttpClient does.

### cache (M4)

Symfony's Cache, with tags: `cache.Get(ctx, c, key, compute)` returns the
value kept for key, or computes, keeps and returns it, with the TTL and
tags that the computation gives its `Item`. Tags are versions, so stores
only get, set and delete bytes, and `Invalidate` forgets every value of a
tag at once. Concurrent misses of a key in a process compute it once.
`MemoryStore` evicts the least recently used values; `cache/rediscache`, a
module of its own, keeps them in Redis, which an app's instances share,
and is checked before serving. `cache.Memory` and `rediscache.Layer` are
the sets to wire; `storetest` checks a store, as it checks both.

RealWorld caches its tags and the article lists that anonymous readers
get; the articles' changes, and their authors' through a `users.Updated`
event, invalidate them, and the Hurl suite runs through the cache.

### Bridges as layers

goncini's packages and adapters export what an app lists in its
`layer.Set`, instead of the providers each app wrote by hand:

- `messenger.SQL` and `messenger.Memory`: a bus and its queues, named by
  `messenger.Config.Queues`; `lock.SQL` and `lock.Memory`, whose store fills
  a `lock.Store` parameter, such as the scheduler's; `pgxdb.Layer`: the
  pool, its transactor, and a `*sql.DB` on it for goose and goncini's SQL
  stores.
- `log/zaplog` and `log/zerologlog`, modules of their own: the library's
  logger from `goncini.Log`, and the handler that replaces goncini's. The
  articles example logs with zap.
- A service that is a `goncini.Background` runs beside the server in
  `serve`, as a fiber of the app's scope, stopping with it:
  `messenger.Config.Consume` makes the bus run a worker so, and
  `scheduler.Config.Run` the scheduler.
- `goncini generate` follows the sets of other packages, such as
  `messenger.SQL`, to find what the app provides, and gives
  `messenger.Config` and `scheduler.Config` defaults when the config has no
  section for them.

### RealWorld's notifications (M3)

Publishing an article dispatches `articles.Published`; the notifications'
subscriber sends a `NotifyFollowers` message to a SQL queue, which a worker
handles outside the request, writing a notification for each follower of
the author, once even if handled twice. `GET /api/notifications` lists
them, and a daily task, locked to one instance, prunes those older than 30
days.

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

- **M4's review:** two independent reviews found 19 defects, each
  reproduced, then fixed with a regression test. Among them:
  - the cache kept a value computed while its tag was invalidated, as
    fresh; waiters failed with a cancelled computation, rather than compute
    it again; and a negative TTL kept an item forever;
  - `httpclient` sent its default headers, such as `Authorization`, to
    other hosts, didn't follow redirects, and its test `Handler` ignored
    timeouts;
  - an email that the SMTP server had accepted failed when the server then
    dropped the connection, so it was sent again; a stalled server held a
    sending past its context; and a DSN with an unencoded `@` sent the
    credentials to the wrong host;
  - attachments whose type has parameters, such as `.txt` files, got an
    empty `Content-Type`; addresses with a display name went through to
    `RCPT`; long headers weren't folded;
  - chat notifications posted their text raw, so an article's title could
    ping a whole channel;
  - translations matched a general pattern before a more specific one,
    substituted values twice, and translated problems lacked
    `Vary: Accept-Language`;
  - RealWorld answered 503 after saving a user whose welcome email
    couldn't be queued, and after deleting an article.

  Decisions taken on the user's behalf:
  - **The SQL transport joins the context's transaction,** as an outbox:
    a message dispatched in `db.SQL.InTx` is queued if, and only if, the
    transaction commits. M3's review left the outbox for an app that needs
    it, and RealWorld now does: its writes and their messages are one
    transaction.
  - **`workflow.Workflow.Apply` checks the guard before the state,** so
    that who may not apply a transition doesn't learn the state, **and
    takes a `save` function,** called between `Transitioning` and
    `Transitioned`, so that `Transitioned`'s listeners see a saved state.
  - **Tag versions are nanosecond times:** invalidations by other
    processes during a computation are told by the clock, as far as clocks
    agree; this process's are tracked exactly.

  What was left, deliberately: `STARTTLS` stays opportunistic without
  credentials, as Symfony's is; a cache invalidation that fails after a
  commit is still a 503; a notification's email gets a new `Message-ID` on
  each retry.
- **M3's re-review:** after M5, three more independent reviews of M3's
  packages found defects that the first had missed, each fixed with a
  regression test. Among them:
  - in New York, `*/15 * * * *` skipped the fall's repeated hour: a cron
    whose minute or hour starts with `*` now runs through both, as Vixie
    cron's does, and one with a fixed time still runs once;
  - a task that outlasted its interval in one instance overlapped the next
    tick's run in another: a lock is held while it runs, and that tick is
    skipped, as it is in an instance alone;
  - a wall clock set back ran a tick again; a tick at a fraction of a
    second shared its lock with its second's other ticks; a task's panic
    was logged without its stack; a `February 29th` cron gave up after 5
    years, before 2104's;
  - the event dispatcher looked its listeners up by reflection at each
    dispatch: it keys them by a generic type now;
  - on PostgreSQL, a SQL transport that hadn't set up its table ran
    `CREATE INDEX IF NOT EXISTS` in each transaction that dispatched,
    holding a process lock while it waited on the others: transactions
    that dispatched locked each other out. In a transaction, it now only
    looks whether the table is there;
  - with `pgxdb.Layer`, a message dispatched in a transaction was queued
    even if it rolled back: pgxdb's transactions now run on a connection
    of its `*sql.DB`, which `db.Join` hands to database/sql's users, so the
    outbox holds with pgx too;
  - a message moved to the failure queue, whose acknowledgement then
    failed, was handled again at each lease, its move refused as a
    duplicate: `Send` replaces a message of the same ID now, in both
    transports, and `messenger:failed:retry` carries on past a failure;
  - `messenger:consume -drain` polled forever a transport that failed;
    serve's worker consumed `Config.Queues`, not the app's own transports;
    a worker could consume the failure transport, which failed over to
    itself; lease extensions that failed weren't logged.

  Worth knowing, now documented: a tick runs once across instances only
  with a lock store they share; the ticks missed while no scheduler runs
  are skipped; `scheduler.Tick(ctx)` is the time a run was due at; SQL
  locks are taken outside the context's transaction; `ErrorEvent` isn't
  dispatched for an error after the response has started; `-drain` leaves
  the messages whose delay isn't due; messages are json/v2, which can't
  encode a `time.Duration`.

  Decisions taken on the user's behalf: `pgxdb.New` takes the `*sql.DB`
  whose users join its transactions, and `db.SQL.Tx` returns a `db.Conn`,
  which may be a joined connection rather than a `*sql.Tx`.

  What was left, deliberately: leases are timed by each worker's clock,
  as lock expiry is; the memory transport refuses to extend a lease that
  ran out even if no one took the message, where SQL's extends it.
- **M3's review:** two independent reviews found 13 defects, each
  reproduced, then fixed with a regression test. Among them:
  - a message that failed for good couldn't move to the SQL failure queue,
    whose table it shared under the same primary key, and was handled again
    forever; `messenger:failed:retry` failed the same way;
  - a handler that ignored its context held a stopping worker past its stop
    timeout, and one failed receive stopped a worker;
  - in New York, a daily cron looped forever on the spring's missing hour,
    and ran twice on the fall's repeated one;
  - a listener added during a dispatch corrupted it;
  - RealWorld didn't notify the followers of an article that reused a slug.

  What was left, deliberately: an article is committed before its event's
  message is sent, so a crash between the two loses the notification; an
  outbox, the message written in the article's transaction, is the fix,
  which waits for an app that needs it. Lock expiry uses each process's
  clock.
- **M2's re-review:** after M5, two more independent reviews of M2's
  packages found defects that the first had missed, each fixed with a
  regression test. Among them:
  - a `TokenBucket` with a daily quota of a hundred thousand overflowed its
    TTL, which turned the limiter off; `ByIP` keyed IPv6 clients by their
    address, which they rotate within their /64; RealWorld's logins had no
    limit across emails: each address has 25 a minute now, as Symfony's
    login throttling has;
  - the contract check failed a 304 to a conditional GET, a
    `Response[T]`'s other statuses and media types, a `+json` request
    body, and blamed an operation for the 404 of a path its requirement
    refuses; `Response[T]` is documented as a `2XX` now;
  - components of generic types got names that OpenAPI refuses, such as
    `BoxInterface{}`, and a type named `ThingInput` was renamed
    `ThingInput2` or not depending on the routes' order;
  - `time.Duration` parameters were described as ISO 8601 durations, which
    `time.ParseDuration` refuses; json/v2's `inline` option, which it
    ignores, promoted members; an `Optional` in a response was written as a
    struct, while documented as its value: it is written as its value now;
  - `goncini generate` missed generic functions' endpoints, and the
    status of a `httpkernel.Problem{Status: http.StatusConflict}`;
  - CORS accepted exact origins that no browser sends, such as
    `example.com`, and wrote sub-second and negative `MaxAge`s wrong;
    `ParseULID` accepted I, L and O, which `routing.ULID` refuses: it is
    strict now, as Symfony's is; `ULID.Scan` refused a PostgreSQL `uuid`
    column's value; a sort could name a field again and again.

  Decisions taken on the user's behalf: `ParseULID` is strict, rather than
  the route requirement and the schema lenient; `Tokens` keep reading
  `time.Now`, which `testing/synctest` fakes, rather than take a clock;
  `Window`'s bounds stay validate tags, which a kernel without a
  `Validator` doesn't check, as documented; the docs page pins Swagger UI
  with integrity hashes.

  What was left, deliberately: the memory rate-limit store isn't capped;
  a firewall's `Load` must not report a nil user, as documented, since
  telling a typed nil apart would need reflection per request.
- **M2's review:** three independent reviews found 23 defects in M2's
  packages, each reproduced, then fixed with a regression test. Among them:
  - schemas didn't follow json/v2's rules when embedded fields shared a
    name, and a request body could share a response's component through a
    nested type whose schemas differed;
  - two routes that OpenAPI can't tell apart overwrote each other: that is
    an error now;
  - `Links` overflowed past the end of a list, and dropped a path's escaping;
  - CORS responses without an `Origin` lacked `Vary: Origin`, which let
    shared caches serve them to other origins;
  - `ClientIPOf` ignored trusted proxies outside endpoints;
  - RealWorld's login told registered emails from others by its timing,
    and the articles example lost an article whose slug looked like a UUID.

  What was left, deliberately: `Authorship` checks a moderator's own roles,
  not the hierarchy, which RealWorld doesn't configure; `Check` can't match
  a `{path...}` wildcard across segments; page links are relative to the
  request's path, behind a proxy's prefix too.
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
  - v0.3.1:
    - `must (x).M()` and `check (x).M()` parsed as calls of functions named
      `must` and `check`, which `ego fmt` then wrote that way.
  - v0.3.2:
    - `??` on a map lookup failed in an expression lambda whose parameter
      type is inferred, as in `name => values[name] ?? name`.
  - v0.3.3, asked for by M5's review:
    - effect methods' spans went to OTel's global tracer provider, even
      under a span of another one; they follow their parent span's now.
  - v0.3.4: a cleanup of v0.3.2's fix, generating the same code.
