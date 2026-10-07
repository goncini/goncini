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
