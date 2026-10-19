# goncini: assessment and direction

*2026-10-03. Baseline: effect-go v0.1.0 (commit 5922162), Go 1.27.1. Decisions marked **decided** were made by the user on 2026-10-03; the rest are proposals.*

## TL;DR

- **goncini is a framework for building APIs** (decided), with Symfony's architecture adapted to Go and effect-go: a kernel, a compiled container, a console, and components that also work on their own. No templating, forms or HTML (decided).
- **Every setting has a default, and every default can be changed** (decided). A new app works without writing any config: goncini chooses the folder layout, the error body, the request size limit and the rest. An app that needs something else changes that one default, in Go, and keeps the others.
- **Everything is `net/http`.** An app is an `http.Handler`, middleware is `func(http.Handler) http.Handler`, and any handler from the ecosystem plugs in.
- **Wiring is generated, not reflected.** effect-go's `layer` builds the graph, and goncini generates what `layer` lacks. This is Symfony's compiled container, in Go.
- **Routes are registered in plain Go** (decided), on `net/http`'s `ServeMux`.
- **Errors are effect-go error sets, each mapped to a response by an exhaustive `match`.** Adding a case without a response fails the build.
- **Every milestone is judged by how easy it makes building an API:** typed handlers, validation, errors with exact bodies, an OpenAPI contract, auth, background work. The reference app is RealWorld's "Conduit" API, and its official test suite is milestone 1's gate.
- **Don't build what the ecosystem has:** no ORM, and no templating or frontend tooling at all.

## 1. Decisions

| Decision | Choice |
|---|---|
| Container | **Build on effect-go's `layer`** (decided). goncini generates what `layer` lacks, as ordinary providers: one that collects every service of a kind into a slice (Symfony's tags), and one per config section. What proves general moves into `layer`. |
| Routing | **Plain Go** (decided): `r.Get(pattern, handler).Name(name)`. URLs are built by route name at run time, so `debug:router` and tests catch typos. No route directives. Routes match their path exactly, and `URL` refuses a URL that a more specific route would serve (both decided). |
| Scope | **APIs only** (decided). No Twig equivalent, no templating, no forms. JSON first; other formats only if an API needs them. |
| Reference app | **RealWorld "Conduit"** (proposed). Its [spec](https://github.com/gothinkster/realworld/tree/main/specs/api) ships an OpenAPI file and a Hurl suite of 13 files, error cases included: 401, 403, 404, 409 and 422, with exact bodies. Passing it is an objective gate, and the app exists in dozens of frameworks to compare with. |
| Packages | **`httpkernel` merges Symfony's HttpFoundation and HttpKernel** (decided): the request and response helpers `net/http` lacks, typed handlers with argument binding, error-to-problem mapping, and the kernel. `routing`, `validator`, `config`, `console`, `db`, `security` and `webtest` are separate packages, as in Symfony. |
| Project layout | **Four layouts, from one package to hexagonal, with packages by feature as the default; the wiring in an `app` package; config as `.ego` code in `config/`** (decided 2026-10-05 and 06). Environment variables are only for `APP_ENV` and secrets. See [layout.md](layout.md). |
| Go version | **1.27** (decided). `encoding/json/v2`, standard from 1.27, says which field failed (`/article/title`), which binding needs, and generic methods allow typed getters such as `Params.Enum`. effect-go itself stays on 1.26. On 1.26, json/v2 is only an experiment behind `GOEXPERIMENT=jsonv2`, with an older API (`inline` where 1.27 has `embed`), and 1.26 leaves support when 1.28 ships, around February 2027. |
| Language | **The framework is written in `.ego`** (decided), with the generated Go committed, so users never need `ego`. Its public API is callable from plain Go, where effect methods take `ctx` explicitly. The generated Go stays committed next to its `.ego` (decided 2026-10-19, as the most maintainable): a package is its directory, so Go in a folder of its own would change the import paths, and Go only on release commits would leave untagged commits that `go get` can't build. `.gitattributes` marks it generated, and the editors nest it under its `.ego`. |
| Ecosystem | **Wrap and wire, don't replace:** go-playground/validator, golang-jwt, and every important data library (below). |
| Data | **Work with every important data library** (decided 2026-10-07): goncini owns what surrounds the queries (the pool's lifecycle, config, transactions in `ctx`, migration commands, test databases), as Symfony's DoctrineBridge does. The core `db` package is on `database/sql`, which sqlc, sqlx, bun, ent, GORM and others take a `*sql.DB` from; adapters in modules of their own add pgx's pool, GORM's transactions, and goose or golang-migrate migrations. |

What effect-go provides and doesn't, per its maintainer session:
- **Nothing HTTP-specific is planned in effect-go.** Handler adapters, error-to-status mapping, middleware, the panic-to-500 boundary and per-request scopes are goncini's.
- **`layer` is effect-go's least settled API.** It matches parameters by type and can't collect providers into a slice or bind config fields. It was a net loss on miniflux's narrow graph; a framework's graph is wide, which is the case it was built for.
- **`ego generate` has no plugin hook.** `//goncini:` comments would survive into the generated Go, but plain-Go routing doesn't need them.

## 2. Symfony, component by component

| Symfony | In goncini | When |
|---|---|---|
| HttpFoundation, HttpKernel | One package, `httpkernel`, on top of `net/http`. First the helpers Go lacks: typed parameters, JSON bodies with readable errors, problem details, content negotiation, trusted proxies, HTTP caching. Then the kernel: a middleware chain plus a panic and error boundary, with kernel events as middleware. | M1 |
| Routing | Plain-Go registration on `http.ServeMux`: names, groups, prefixes, URL generation, `debug:router`. | M1 |
| DependencyInjection | effect-go `layer`, plus `goncini generate` for autoconfiguration (tag collections) and config sections. As in Symfony, the container is compiled: generated Go, no reflection. | M1 |
| Dotenv, Config, environments | Config as `.ego` code in `config/`: the shared values, then one function per environment, picked by `APP_ENV` (decided). Environment variables only for secrets, read through `env.Secret`, with Symfony's `.env` file precedence. No YAML, TOML or JSON config files, and no loader for them (decided). | M1 |
| Console | The app binary is the console: `console`, a small package on the standard library's `flag`. Commands are services, collected by autoconfiguration. | M1 |
| ErrorHandler, `HttpException` | Error sets mapped to RFC 9457 problems by exhaustive `match`; a panic becomes a 500 at the boundary; the renderer can be swapped. | M1 |
| Serializer, `#[MapRequestPayload]`, `#[MapQueryString]` | Typed handlers in `httpkernel`, `func(ctx, In) (Out, error)`. `In` is bound from the path, query, headers and JSON body; `Out` is encoded as JSON. | M1 |
| Validator | `validate` struct tags (go-playground/validator) and an optional `Validate()` method; violations become a 422. | M1 |
| Security | M1: password hashing, a token authenticator, the current user in `ctx`. M2: voters, and firewalls per route group. | M1, M2 |
| Doctrine ORM, Migrations, DoctrineBridge | No ORM of goncini's own: `db` on `database/sql`, with adapters (`pgxdb`, `gormdb`, `goosedb`, `migratedb`) behind `db.Transactor` and `db.Migrator`, and the `db:migrate` commands. ent, bun and Atlas follow the same pattern later. | M1 |
| Runtime, Stopwatch, Monolog | `scope.Main`, effect-go's spans, and `slog` records carrying trace IDs. | M1 |
| WebTestCase, KernelBrowser, DAMADoctrineTestBundle | `webtest`: an in-process client on the test graph, checks of problems and violations, tests in rolled-back transactions, `testing/synctest` for time. | M1 |
| NelmioApiDocBundle, API Platform | OpenAPI 3.1 generated from handler types; a docs page in dev; pagination and filtering helpers; responses checked against the contract in tests. | M2 |
| RateLimiter, CORS | Middleware. | M2 |
| EventDispatcher | Typed events (generics), with subscribers collected by autoconfiguration. | M3 |
| Messenger | Messages on effect-go: retries are schedules, workers are fibers in the app scope, transports for memory and SQL first. | M3 |
| Scheduler, Lock | `repeat` with a schedule, in the app scope; locks in memory and in the database. | M3 |
| HttpClient | The `net/http` client with `schedule.Retry` and a span per call. | M4 |
| WebProfilerBundle | A dev profiler built from the spans and log records effect-go already produces. | M5 |
| MakerBundle | `goncini make:*`, writing `.ego` files. | M5 |
| Bundles, Flex recipes | A bundle is a package exporting a `layer.Set`, a config section, routes and commands. | M5 |
| Cache, Mailer, Mime, Notifier, Translation, Workflow | Cache in memory and Redis, with tags; mail and notifications sent through Messenger; error messages translated; state machines. | M4 |
| Uid | UUIDs and ULIDs that bind, scan and route. | M2 |
| Twig, Form, Asset, AssetMapper, UX | Dropped (decided): goncini is for APIs. | — |
| YAML/XML config, ExpressionLanguage, PropertyAccess, service locators, lazy proxies | Dropped: Go code, typed values and plain constructors replace them. | — |

## 3. Milestone 1: the kernel

### 3.1 What an app looks like

A sketch of the RealWorld articles slice, not a final API.

```go
// articles/errors.ego
error ArticleError {
	NotFound{ Slug string } "no article {Slug}"
	Forbidden{ Slug string } "not the author of {Slug}"
	Storage{ Cause error }
}

// Problem says what each case looks like over HTTP. A new case without an
// arm doesn't compile.
func Problem(err ArticleError) httpkernel.Problem {
	return match err {
		NotFound(_)  => httpkernel.Problem{Status: http.StatusNotFound, Detail: err.Error()}
		Forbidden(_) => httpkernel.Problem{Status: http.StatusForbidden, Detail: err.Error()}
		Storage(_)   => httpkernel.Problem{Status: http.StatusServiceUnavailable}
	}
}

// Problems pairs the set with its responses; kernels register it.
var Problems = httpkernel.Map[ArticleError](Problem)
```

```go
// articles/controller.ego
type Controller struct{ articles *Service }

func NewController(s *Service) *Controller { return &Controller{s} }

// Routes makes Controller a routing.Routes, so it is collected with the others.
func (c *Controller) Routes(r *routing.Router) {
	r.Get("/api/articles/{slug}", httpkernel.Endpoint(c.Show)).Name("article_show")
	r.With(security.Required).Post("/api/articles", httpkernel.Endpoint(c.Create)).Name("article_create")
}

type ShowInput struct {
	Slug string `path:"slug"`
}

// Show compiles to func (c *Controller) Show(ctx context.Context, in ShowInput) (Envelope, error).
effect (c *Controller) Show(in ShowInput) (Envelope, ArticleError) {
	a := check c.articles.BySlug(in.Slug)
	return Envelope{Article: a}, nil
}
```

```go
// inject.go
//go:build egolayers

// BuildKernel is the app's graph. NewConduitRenderer, passed to Build
// directly, replaces the framework's problem+json renderer with RealWorld's
// {"errors": {...}} bodies.
func BuildKernel(ctx context.Context, s *scope.Scope, cfg config.Config) (*goncini.Kernel, error) {
	panic(layer.Build(Services, goncini.Framework, Autoconfigured, NewConduitRenderer))
}
```

```go
// autoconfigured.go: written by hand until goncini generate exists
var Autoconfigured = layer.Set(routes, commands, dbConfig, authConfig)

func routes(a *articles.Controller, u *users.Controller) []routing.Routes { return []routing.Routes{a, u} }
func dbConfig(c Config) db.Config                                       { return c.DB }
```

`main` is `goncini.Main(config.Load, app.Build)` ([layout.md](layout.md)). It loads the config, runs the kernel in `scope.Main`, and dispatches console commands: `serve` (the default), `debug:router`, `debug:container`, `db:migrate`, and the app's own.

### 3.2 Parts

- **`httpkernel`, the HttpFoundation half:** what `net/http` lacks, as functions over `*http.Request` and `http.ResponseWriter` rather than wrappers around them:
  - typed parameters: `Query(r).Int("limit", 20)`;
  - JSON bodies with a size limit and errors that name the field;
  - JSON responses and RFC 9457 problem responses;
  - content negotiation;
  - the client IP and scheme behind trusted proxies;
  - conditional requests and `Cache-Control` for API responses;
  - a response wrapper that lets middleware see the status.
- **`httpkernel`, the HttpKernel half:**
  - typed handlers through `httpkernel.Endpoint`, with binding plans built once per route at startup from tags (`path`, `query`, `header`, and a `Body` field), then validated;
  - error mappers and a swappable `Renderer`;
  - each request runs in its own scope, so fibers forked during a request are awaited before it ends;
  - a recovery boundary turns panics into 500s, re-panicking `http.ErrAbortHandler`; in dev the response includes the stack, from `*scope.Panic`;
  - a span per request, named after the route pattern;
  - graceful shutdown on SIGTERM.
- **`routing`:** a `Router` on `ServeMux`, with names, groups, `With(middleware)`, `URL(name, params)` and the listing behind `debug:router`.
- **`config`:** `APP_ENV`, and secrets through `env.Secret` from `.env`, `.env.local`, `.env.$APP_ENV` and `.env.$APP_ENV.local`, with real environment variables winning. Missing secrets and an unknown `APP_ENV` fail at boot, all at once. The settings themselves are the app's `.ego` code ([layout.md](layout.md)); `debug:config` prints them with secrets masked.
- **`console`:** commands as services implementing `console.Command`, on the standard library.
- **`goncini generate`** (last, and only once writing these providers by hand gets annoying): runs `ego generate`, then writes the `Autoconfigured` providers:
  - a slice for each framework interface: `routing.Routes` and `console.Command` in M1;
  - one provider per config section.

  It also writes the graph `debug:container` prints. effect-go's lowering type-checks the injector files, so a missing `Autoconfigured` must not break it. The order is: write a stub, run `ego generate`, load types, write the providers, run `ego generate` again to wire the layers.
- **`db`:** a `*sql.DB` provider closed with the scope, transactions in `ctx` behind `db.Transactor`, and migrations behind `db.Migrator`, with adapters for pgx, GORM, goose and golang-migrate. The reference app uses SQLite through modernc.org/sqlite, with no cgo.
- **`security` (lite):** password hashing, a token authenticator (RealWorld's `Authorization: Token …`), the current user in `ctx`, and a `Required` middleware. Its failures are an error set, mapped like any other.
- **`webtest`:** an in-process client on a test graph (`layer.Build(Services, …, NewMemStore)`), with assertion helpers.

### 3.3 Gates

1. **RealWorld's Hurl suite passes** against `examples/realworld`: all 13 files, error bodies included.
2. **The articles slice is at least 25% shorter** (decided 2026-10-07, down from 30%) than the same slice written with plain `net/http` and hand wiring. We write both, as effect-go did for its demos.
3. **Adding a case to a mapped error set without an arm fails the build.**
4. **SIGTERM during a slow request:** the request completes, the database closes after it, and no goroutine is left. Tested under `synctest`.
5. **An agent given only goncini's AGENTS.md** adds an endpoint with validation and a new error case correctly on its first try, and hidden tests pass. This is the gate effect-go used.

### 3.4 Order of work (components first)

> **Status (2026-10-07):** steps 1 to 5 are built: [`httpkernel`](../httpkernel), [`routing`](../routing) with requirements, [`validator`](../validator), config as code, [`console`](../console) and `goncini.Main`; and step 6: [`db`](../db) and its adapters, [`webtest`](../webtest) and [`security`](../security). They are used by [examples/articles](../examples/articles) in the [layout](layout.md). Step 7 is built: the RealWorld app ([examples/realworld](../examples/realworld)) passes the Hurl suite, and the other gates pass: its articles feature is 34% shorter than [the plain `net/http` version](../examples/realworld-plain), a case without a response fails the build, shutdown waits for requests, and an agent given only [AGENTS.md](../AGENTS.md) added an endpoint that passed hidden tests. Step 8 is built: `goncini generate` writes the apps' `Autoconfigured`, and the description that `debug:container` lists. Exporting spans is left for when an app needs it: the kernel's spans go to the global tracer provider, and log records carry their trace IDs. [CHANGELOG.md](../CHANGELOG.md) says what they do, what building them found, and where they differ from this plan.

Each step is a package that works in any `net/http` app, the way Laravel uses Symfony's HttpFoundation. effect-go matters most from step 4.

1. **`httpkernel`'s HttpFoundation half:** parameters, JSON bodies, problems and negotiation first; then trusted proxies, HTTP caching and the response wrapper.
2. **`routing`:** names, groups, URLs, the route listing.
3. **Argument binding and `validator`:** typed inputs from the path, query, headers and body, with violations.
4. **`httpkernel`'s HttpKernel half:** typed handlers, error mapping and the renderer swap, the panic boundary, request scopes, shutdown.
5. **Config, console, `goncini.Main`, logging and tracing,** with `examples/articles` reshaped into the [layout](layout.md). With config come trusted hosts, so that `httpkernel.BaseURL` can't use a host the client made up.
6. **`db`, `security` (lite) and `webtest`.**
7. **The RealWorld app**, wired by hand, then the remaining gates: the plain-Go comparison, AGENTS.md and the agent test.
8. **`goncini generate`**, if the hand-written wiring got annoying.

## 4. Later milestones

The goal, set by the user on 2026-10-07, is parity with Symfony's components that an API uses, milestone by milestone. Each milestone's packages have tests and docs, an example uses them, its gate passes, and a review of the milestone has its findings fixed.

| Milestone | Content | Gate |
|---|---|---|
| **M2: API contract and security** | OpenAPI 3.1 from endpoint types (`/openapi.json`, a docs page in dev, `openapi:dump`); responses checked against the contract in `webtest`; CORS; RateLimiter; voters; firewalls per route group; Uid; pagination and filtering helpers | Done (2026-10-13): the four gates of 4.1 |
| **M3: async** | EventDispatcher (typed events, kernel events); Messenger (memory and SQL transports, retries as schedules, a failure transport, `messenger:consume` workers as fibers); Scheduler; Lock | Done (2026-10-13): killing a worker mid-message loses nothing; SIGTERM drains within the deadline |
| **M4: integrations** | Cache (memory, Redis in an adapter module, tags); HttpClient (retries, tracing); Mailer and Mime; Notifier; Translation of error messages; Workflow | Done (2026-10-13): the six gates of 4.3 |
| **M5: developer experience** | A profiler built from spans (`debug:requests`, OTLP export); `goncini new` and `make:*`; bundles | Done (2026-10-13): the four gates of 4.4 |

> **Status of the goal (2026-10-13): complete.** M2 to M5 are done: each milestone's packages have tests and docs and are used by an example; each review's findings are fixed, M2's and M3's after a second review too; and CI is green on all 13 jobs. On 2026-10-13 every gate test passed again, none skipped, Hurl and Redis included: M2's `TestDocumentCoversRealWorlds`, `TestSpec` (the Hurl suite, checked against the document), `TestClient` (oapi-codegen's client) and `TestDocumentIsDumped`; M3's `TestKilledWorkerLosesNothing`, `TestSIGTERMDrains`, `TestRetriesAndFailures`, `TestFailedCommands`, `TestOneInstancePerTickInSQL` and `TestNotifications`; M4's `TestCaching`, `rediscache`'s suite, `httpclient`'s tests, `TestWelcomeEmail`, `TestRemovalNotices`, `TestFrench` and `TestReview`; M5's `TestProfiler`, `TestExport`, `TestNew` and `TestBundle`.

### 4.1 Milestone 2

**Gates** (proposed 2026-10-07, refining the user's "a client generated from our OpenAPI document passes the Hurl suite"; the user accepted the refinement on 2026-10-13):

1. **The document covers RealWorld's.** RealWorld's [`openapi.yml`](https://github.com/gothinkster/realworld/blob/main/specs/api/openapi.yml) and the document `examples/realworld` generates have the same operations (method and path), parameters, request bodies and success statuses, and ours declares every error status RealWorld's tests expect. A test compares them.
2. **The document holds for the whole Hurl suite.** Running the suite with every request and response checked against our document finds no response, status or body that the document doesn't allow.
3. **A generated client works.** A client generated from our document by oapi-codegen compiles, and a test drives every RealWorld operation through it against the app, decoding each typed response.
4. **No annotations.** The RealWorld app gets its document without writing any OpenAPI by hand beyond a title, a version and doc comments.

> **Status (2026-10-07):** steps 1 to 7 are built: [`openapi`](../openapi), the annotations that `goncini generate` writes, contract checks in `webtest`, [`cors`](../cors), [`ratelimit`](../ratelimit), voters and firewalls per route group in [`security`](../security), [`uid`](../uid) and [`listing`](../listing). Gates 1 to 4 pass, in [examples/realworld](../examples/realworld): `TestDocumentCoversRealWorlds`, `TestSpec`, `TestClient`, and an app whose only OpenAPI settings are a title and a version.

**Decisions taken on the user's behalf:**

- The errors an endpoint can fail with are found in its code by `goncini generate`, following its calls, rather than declared: effect-go erases the error set from the compiled signature, and the error set alone would claim a 403 on reads.
- The document is served at `/openapi.json` in every environment, since clients and gateways read it; the Swagger UI page, loaded from a CDN, is off unless `DocsPath` is set, as RealWorld does in dev.
- The docs page is Swagger UI, as NelmioApiDocBundle's is.
- Error bodies are described by the Go type that the renderer writes, which the renderer tells a `httpkernel.ValueRecorder`, so a custom error format needs no declaration.

**Order of work:**

1. `openapi`: JSON Schemas from Go types, operations from endpoints and route middleware, the error responses from error sets, `/openapi.json`, a docs page in dev, `openapi:dump`.
2. `goncini generate` finds which error cases each endpoint can return.
3. `webtest` checks responses against the document.
4. The gates.
5. CORS and RateLimiter, as middleware.
6. Voters, and firewalls per route group.
7. Uid; pagination and filtering helpers.

### 4.2 Milestone 3

**Gates** (proposed 2026-10-07, making the user's "killing a worker mid-message loses nothing; SIGTERM drains within the deadline" testable):

1. **A killed worker loses nothing.** A test sends 100 messages to the SQL transport and runs `messenger:consume` as a separate process; it kills the worker with SIGKILL while a handler is running, then starts another. Every message is handled to completion, the one interrupted again after its lease runs out (at least once), and the transport ends empty.
2. **SIGTERM drains within the deadline.** A worker sent SIGTERM mid-message takes no new message, finishes the one it has, and exits 0 before its shutdown timeout; a handler that outlives the timeout is cancelled, and its message goes back to the transport, not lost.
3. **Failures are retried, then kept.** Under `testing/synctest`, a failing handler is retried on its schedule (exponential backoff), and after its last attempt the message lands in the failure transport, from which `messenger:failed:retry` sends it again.
4. **One instance runs each task.** Two app instances sharing a database run a scheduled task each period, and the lock lets exactly one of them run it each time.
5. **RealWorld uses them:** publishing an article dispatches an event, whose subscribers send a message that a worker handles, outside the request.

> **Status (2026-10-07):** built: [`event`](../event) with kernel events, [`lock`](../lock), [`messenger`](../messenger) and [`scheduler`](../scheduler). The five gates pass: `TestKilledWorkerLosesNothing` and `TestSIGTERMDrains` run workers as processes; `TestRetriesAndFailures` and `TestFailedCommands`; `TestOneInstancePerTickInSQL`; and RealWorld's `TestNotifications`.

**Decisions taken on the user's behalf:**

- goncini's own SQL stores (locks, then Messenger's transport) work with SQLite and PostgreSQL, the databases its examples and adapters test; MySQL waits for an app that needs it. They create their tables when first used, as Messenger's Doctrine transport does by default, or leave it to the app's migrations.
- The root module requires modernc.org/sqlite for its tests only; Go's module graph pruning keeps it out of apps' builds. PostgreSQL is tested in the `pgxdb` module, whose CI job has a database.
- Kernel events are `RequestEvent`, `ErrorEvent` and `ResponseEvent`; `RequestEvent` runs inside the kernel's middleware, so that its listeners see trusted proxies' and hosts' work, and their answers go through the access log and CORS.
- Cron follows Vixie cron where cron implementations differ: a day field starting with `*` restricts nothing for the OR of the two days, a time skipped by daylight saving doesn't run, and a repeated one runs once.
- Packages export `layer.Set`s, such as `messenger.SQL`, and adapters of logging libraries are modules of their own, `log/zaplog` and `log/zerologlog`, so that wiring them is one line (asked by the user on 2026-10-07). Symfony's `kernel.terminate`, for work after the response, is Messenger's job.

**Order of work:** `event` (typed events, subscribers collected by `goncini generate`, kernel events); `lock` (memory and SQL); `messenger` (bus, handlers, memory and SQL transports, retries, the failure transport, `messenger:consume`); `scheduler`; then the gates.

### 4.3 Milestone 4

**Gates** (proposed 2026-10-07):

1. **Caching is invisible but for speed.** RealWorld caches its tags and article lists with tags of their own: a second request makes no query, and publishing, editing or deleting an article invalidates what it changes, so no test of the Hurl suite sees a stale answer. Concurrent misses of a key compute it once. The Redis adapter passes the same conformance suite as the memory cache, against a real Redis in CI.
2. **Outgoing HTTP is resilient and traced.** The HTTP client retries connection failures, 429s and 5xxs of idempotent requests on a schedule, honoring `Retry-After`, never retries a POST unless asked, propagates the trace context, and a mock transport answers tests, as Symfony's MockHttpClient does.
3. **Mail goes out of the request.** RealWorld welcomes a registered user by email: a Messenger message, handled by a worker, renders a multipart text and HTML email that an in-process SMTP server receives and parses back intact.
4. **Notifications reach their channels.** A notification goes to the channels its importance asks for, email and a chat webhook, each recipient's way, as Symfony's Notifier does.
5. **Errors speak the client's language.** In the articles example, `Accept-Language: fr` gives French problem titles and violation messages from a catalog, falling back to English, and `translation:lint` lists the messages a catalog lacks.
6. **Workflows guard their transitions.** The articles example gets a review workflow, draft to reviewed to published: a transition that the article's state doesn't allow is a 409, a guard is a security voter, each transition dispatches events, and `workflow:dump` draws the graph in Mermaid.

> **Status (2026-10-07):** built: [`cache`](../cache) and [`cache/rediscache`](../cache/rediscache), [`httpclient`](../httpclient), [`mime`](../mime) and [`mailer`](../mailer), [`notifier`](../notifier), [`translation`](../translation) and [`workflow`](../workflow). The six gates pass: RealWorld's `TestCaching` with the Hurl suite through the cache and `rediscache`'s suite on Redis; `httpclient`'s tests; `TestWelcomeEmail`; `TestRemovalNotices`; the articles example's `TestFrench` and `TestReview`. M4's review (2026-10-13) found 19 defects, fixed with regression tests; see [CHANGELOG.md](../CHANGELOG.md). M4 is done; M5 is next.

**Decisions taken on the user's behalf:**

- Translation catalogs are Go maps whose keys are the app's messages, as gettext's are, with `{placeholders}` matching messages that carry values: no keys to invent, and goncini's and validators' messages translate unchanged.
- Mail and notifications have no templates, as the user excluded templating: bodies are the app's strings, or the standard library's `html/template`.
- Chat channels post Slack's `{"text": …}`, which Slack, Mattermost, Rocket.Chat and Discord's Slack endpoints take; SMS waits for an app that needs it.
- Workflows are state machines, a subject in one place at a time; Symfony's workflows of several places at once wait for an app that needs them.

**Order of work:** `cache` (memory, tags, stampede protection), `cache/rediscache`; `httpclient`; `mime` and `mailer` (SMTP and in-memory transports, sending through Messenger); `notifier`; `translation`; `workflow`; then the gates.

### 4.4 Milestone 5

**Gates** (proposed 2026-10-07, approved by the user on 2026-10-13):

1. **The profiler shows what a request did.** In dev, each request's profile keeps its route, status and duration, the spans of the endpoint, the store's queries and outgoing HTTP calls, and its log records. `/_profiler` lists the profiles as JSON, and `debug:requests`, a process of its own, prints the last ones. With an OTLP endpoint configured, the same spans reach a collector, which a test runs in the process.
2. **A new API in one command.** `goncini new acme` writes a module in goncini's layout whose `go test ./...` passes and whose `go run . serve` answers, with nothing to edit first.
3. **Makers write features that work.** In that new app, `goncini make:feature notes` writes a feature (its error set, endpoints, store, migration and test) that `goncini generate` wires, whose test passes, and whose endpoints the OpenAPI document lists; `make:endpoint`, `make:message`, `make:command` and `make:migration` add to it.
4. **A bundle plugs in with one line.** A third-party bundle, a package exporting a `layer.Set`, a config section, routes, commands and a subscriber, adds them all to the articles example when its set joins the app's `Services`: no other wiring, and its config section has a default when the app's config has none.

> **Status (2026-10-13):** built: [`profiler`](../profiler) with tracing, query spans and [`otel/otlp`](../otel/otlp); `goncini new` and the makers in [cmd/goncini](../cmd/goncini); bundles, with any provider's missing `Config` defaulted, and [examples/maintenance](../examples/maintenance). The four gates pass: the articles example's `TestProfiler` and `otlp`'s `TestExport`; `TestNew`, which writes an app, tests and serves it, then runs each maker on it and tests it again; and the articles example's `TestBundle`. M5's review (2026-10-13) found about 25 defects, fixed with tests; see [CHANGELOG.md](../CHANGELOG.md). Its decisions: the tracer provider stays OTel's global one, which the spans that start a trace use, restored only if still the app's (effect-go v0.3.3, adopted, makes the spans under them follow their parent's provider); a profile is saved on the request's goroutine, in dev; only goncini's and bundles' `Config`s are defaulted, so that a missing section of the app's own stays an error. M5 is done. On 2026-10-13 the user accepted M2's and M3's reviews of 2026-10-07 as their milestones' reviews, and the goal, M2 to M5, is complete. The user then asked for M2 and M3 to be reviewed again: five more independent reviews (2026-10-13) found defects that the first ones had missed, about 50 in all, fixed with regression tests; see CHANGELOG.md's re-reviews for them and for the decisions taken. Among those: `pgxdb.New` takes the `*sql.DB` whose users join its transactions, through `db.Join`, so that the messenger's outbox holds with pgx; `ParseULID` is strict, as Symfony's is.

**Order of work:** the profiler (`profiler`, in dev, with a file store) and OTLP export (`otel/otlp`, an adapter module); `goncini new`; `make:*`; bundles, with `goncini generate` giving any provider's missing config section a default; then the gates.

### 4.5 After the milestones

**Order of work** (proposed 2026-10-18, approved by the user the same day):
dogfood a real API; OIDC authentication, verifying an identity provider's
tokens against its JWKS; Redis transports for Messenger and locks, in an
adapter module; webhooks, received with their signatures checked and sent
through Messenger with retries; then a secrets vault, a Dockerfile from
`goncini new`, plurals in translations, and mail API transports.

> **Status (2026-10-18):** the dogfood is [examples/heartbeat](../examples/heartbeat), a cron-job monitoring API after Healthchecks.io, made with `goncini new` and the makers, then by hand. The friction it found, eleven items, is fixed in goncini but for two decisions to keep things as they are; see [dogfood.md](dogfood.md). Decisions taken on the user's behalf: `Verifier.Verify` takes a `ctx`, a breaking change for verifiers of an app's own; a firewall without `Tokens` hands the token to `Load`, for keys that the app keeps, rather than a `VerifierFunc`; a test that waits for a retry over a real socket runs in real time, as a synctest bubble's time doesn't move while the app waits on the network, until the webhooks step brings a receiver that needs none.

> **Status (2026-10-18):** OIDC is [security/oidc](../security/oidc), tried in an app made with `goncini new`, whose `/me` serves the provider's user. Decisions taken on the user's behalf: a verifier hands the token's claims to `Load` through `security.SetClaims` and `security.Claims(ctx)`, rather than a `Verifier` that returns them, so that `Verify` keeps its signature; the keys are fetched when a token first needs them, not at boot, so that an app starts while its provider is down, and they serve for an hour, then while the provider is down; a token with an unknown key fetches them once a minute at most; only asymmetric algorithms are accepted, as the provider's keys are public; and `oidctest`'s provider closes its connections after each response, which lets a synctest bubble's time move while it is up, if it is made outside the bubble. That fix may serve the webhooks step's tests, and heartbeat's alerts test.

> **Status (2026-10-18):** Redis is the [redis](../redis) module, tried in an app made with `goncini new`, whose message went through Redis. Decisions taken on the user's behalf: one module for the messages and the locks, which need a Redis that keeps what it is given, apart from `rediscache`, whose Redis may evict: its client is a type of its own, so that the two don't clash; `memory://` in `.env.test` keeps them in memory, as Symfony's `in-memory://` does, as an app's test packages, which run at once, would otherwise share a Redis's queues; a queue is a hash, a sorted set by due time and a hash of leases, which Lua scripts change, rather than a stream, which can't delay a message; the scripts take the transport's clock, so that `transporttest`'s fake one moves it, and the locks expire with Redis's own.

> **Status (2026-10-19):** webhooks are the [webhook](../webhook) package, used by heartbeat's webhook channels. Decisions taken on the user's behalf: the Standard Webhooks specification signs what goncini sends, as Svix and others do, rather than a scheme of goncini's; receiving checks Standard's, GitHub's and Stripe's schemes, and an app writes another as a `Scheme`; a delivery is a message of its own, retried 10 times over about 8 hours with `messenger.Retry`, new, rather than on the bus's schedule; it goes to public addresses only, with `httpclient.Public`, checked as it connects, after the name is resolved, unless `Config.Private` says otherwise, as in dev and tests; the receiver dedupes by the webhook's ID, which goncini doesn't store for it; and a handler that sends webhooks, or dispatches messages, keeps the bus that its `Handlers` method is given, since it can't take the bus, which takes the handlers: a `layer` provider that breaks such a cycle, lazily, would be effect-go's to add, if apps need one more than this.

> **Status (2026-10-19), step 5:** the secrets vault is [secrets](../secrets), with the goncini tool's `secrets:` commands. Decisions taken on the user's behalf: HPKE from the standard library, with ML-KEM-768 and X25519, rather than a dependency on libsodium's sealed boxes as Symfony's, and post-quantum, since a vault is committed for years; a file per secret, so that two branches that set different secrets merge; the vault over the `.env` files, which hold dev's values, and under the process's environment; and a locked vault's secrets taking no `.env` value, so that a server without the key fails to start rather than running on dev's database.

> **Status (2026-10-19), step 5:** `goncini new` writes a Dockerfile. Decisions taken on the user's behalf: no `.env` file in the image, which the `.dockerignore` keeps out of the build, so that a missing secret fails at start, rather than taking dev's value; distroless's static image, run as nonroot, since the binary is static, with modernc's SQLite; a `/data` that nonroot owns, for a new app's SQLite database in a volume; and no CI job that builds it, since a new app requires the released goncini, and the Dockerfile was checked by building and running a new app's image.

> **Status (2026-10-19), step 5:** translations have plurals. Decisions taken on the user's behalf: ICU's plural syntax, as Symfony's ICU messages have it, but in catalogs only, since messages stay their own IDs, which the app writes as it formats them; a message's plural matches any of its forms, so that a translation of "{n, plural, one {# item} other {# items}}" serves "1 item" and "3 items" alike; CLDR's rules from x/text, already a dependency, rather than tables of goncini's own; no `select` or nested arguments beyond plurals, which no message needs yet.

> **Status (2026-10-19), step 5:** the mailer has API transports for Postmark, SendGrid, Mailgun and Resend. Step 5 is done, and with it the order of work of 2026-10-18. Decisions taken on the user's behalf: the transports parse the message that `Bytes` wrote back into an `Email`, with `mime.Parse`, rather than taking the `Email`, so that `Transport` and the queued `SendEmail` messages stay as they are, and a worker sends what was dispatched; a Bcc is what the envelope has that To and Cc don't; Symfony's DSNs, with `default` for the provider's endpoint; four providers, among the most used, while SES and the others work over `smtp://`; no retries by the transport, as a send isn't idempotent: Messenger retries the async ones.

### 4.6 Gaps with Symfony

**Backlog** (proposed 2026-10-19), the backend parts of Symfony that
goncini lacks, most needed first. The user approved the first three the
same day, as the order of work.

1. **Login throttling**, as a service of the framework's: RealWorld's was
   the app's own.
2. **File uploads**, `multipart/form-data` bodies in endpoints, described
   in the OpenAPI document.
3. **Server-sent events**, streamed responses, for what Symfony does with
   Mercure, then a hub that fans their events out across the app's
   instances, on Redis's pub/sub.
4. **Content negotiation and serialization groups:** a response is its
   output type, as JSON only. Decided 2026-10-19: no groups, and JSON
   only, with a 406 for the clients that accept none of it.
5. **Messenger transports** beyond SQL, Redis and memory: AMQP, SQS, Kafka.
6. **Notifier bridges** beyond email and chat webhooks: SMS, push.
7. **Profiler UI:** a JSON page and `debug:requests`, no toolbar.
8. **Wiring:** service decoration, lazy services.
9. **Validator depth:** go-playground's tags, not Symfony's constraints.
10. **Mailer bridges** beyond SMTP and four APIs.
11. **Components without an equivalent:** Intl, HtmlSanitizer, Semaphore,
    ObjectMapper.

Form login, sessions, remember-me and CSRF stay out, as browser concerns.

> **Status (2026-10-19), step 1:** login throttling is `ratelimit.Logins`, which RealWorld uses, with a store in Redis, `redis.RateLimit`. Decisions taken on the user's behalf: Symfony's defaults and its two limits, 5 attempts a minute at a user from an address, which a success gives back, and five times as many from an address, which it doesn't; a sliding window for both; users named without case; `ratelimit.Config` for the limits, in the package's `Config` rather than a section of their own, as `goncini generate` defaults only types named `Config`; the Redis store watches the key and tries again, up to 50 times, rather than a Lua script per policy, since the policies are Go: a store's `Update` may call its function more than once.

> **Status (2026-10-19), step 2:** file uploads are form bodies in `httpkernel`, described in the OpenAPI document and sent by `webtest`'s client. Decisions taken on the user's behalf: a form is a `Body` struct whose fields have `form` tags, rather than inputs of their own, so that the input still says where every value comes from; its values bind as query parameters do, with the same options, and its files as a `File`, which a required option or a pointer must make explicit; a form's invalid values are a 422 pointing at them, as a JSON body's are, not a 400 as parameters are; URL-encoded forms too, for the webhooks of Slack and Twilio, which send them; the files are removed once the endpoint returns, as PHP does, and a file kept for later is the app's to copy; a megabyte of files in memory, the rest in temporary files; the body's limit as a tag on the `Body` field, which serves raw and JSON bodies too, rather than an option of `Endpoint`, so that an input type says it all; no checks of a file's type or size of goncini's, which the input's `Validate` makes, since the client's `ContentType` can't be trusted anyway.

> **Status (2026-10-19), step 3:** server-sent events are `httpkernel.Stream`, documented in the OpenAPI document and read by `webtest`. The first three steps of the backlog are done. Decisions taken on the user's behalf: events come from a channel, which the stream selects on with its keep-alive and the request's end, in the request's goroutine, rather than from a callback, which would need a lock around the writer; each event's data is JSON, so that the stream has a schema; the stream ends when the server begins to shut down, through `Stopping`, so that a shutdown doesn't wait 30 seconds for streams, whose clients reconnect elsewhere; a keep-alive comment every 15 seconds without events, for proxies; no fan-out of events across instances, which Symfony gets from Mercure: a hub on Redis's pub/sub would be the next step, when an app needs one, and is now in the backlog; OpenAPI 3.1 has no schema for the events of a stream, so the response's schema is their data's, as 3.2's `itemSchema` would say.

> **Status (2026-10-19), step 3, the hub:** the [hub](../hub) package fans a stream's events out across the app's instances, with `redis.Hub`, and heartbeat streams its flips with it. The user decided the same day the forks of steps 4 to 6: separate output types rather than serialization groups, JSON only with a 406, AMQP and SQS transports, and Twilio and Firebase notifiers. Decisions taken on the user's behalf: the hub is in the app, rather than a Mercure server of its own, so that subscriptions are the app's endpoints, with its authentication; a hub hands an event to its own subscribers as it is published, and to the others' through the transport, which skips the publisher's, so that no event is lost before a transport listens; Redis's pub/sub, which misses what an instance isn't subscribed for, rather than a stream per topic, with each instance keeping the last 100 events of each topic for the clients that reconnect, as it gets them; an event replayed after the reconnecting client's by its place in that history, or else by its UUIDv7, which sorts by time; a client that falls 64 events behind is let go, rather than slowing the others, and reconnects for what it missed; publishing from a message's handler, after the commit, as heartbeat does, rather than a transaction hook of the hub's.

> **Status (2026-10-19), step 4:** an endpoint answers 406 when the `Accept` header allows none of its result's media type, before it binds anything, so that a refused request has no effect. Serialization groups are not built, as the user decided: a response whose members differ from another's is a type of its own, which the OpenAPI document describes exactly. Decisions taken on the user's behalf: any `+json` type in `Accept` allows JSON, as a `Response` may answer `application/vnd.api+json`; an `Accept` header that doesn't parse is ignored rather than refused; a `Responder` of the app's own, whose media type is unknown, isn't checked; the 406 isn't in each operation of the OpenAPI document, as no client that reads the document asks for anything else.

> **Status (2026-10-19), step 5:** RabbitMQ and Amazon SQS are transports of the [messenger/amqp](../messenger/amqp) and [messenger/sqs](../messenger/sqs) modules, which `transporttest.RunBroker` checks against RabbitMQ 4 and ElasticMQ, in CI too. Kafka isn't built, as the user chose. Decisions taken on the user's behalf: a module each, so that goncini requires neither client; the failure transport in the app's database, as Symfony advises, since neither broker can list a queue's messages nor remove one of them; the brokers time the delays and the leases, rather than the workers, so `RunBroker` checks them by the clock, and the brokers' documentation says what they don't do: a queue's messages come in the order they were sent, and a message sent twice is there twice; RabbitMQ's delays go through a queue per power of two seconds, declared when first needed, whose messages expire back to the queue, which delays again those that are early, rather than through a queue per delay, as Symfony's do, or the delayed-message plugin, which not every RabbitMQ has; a RabbitMQ lease is a `basic.get` on a channel of its own, which holds the message until the channel closes, with the connection, rather than for a time, which RabbitMQ doesn't keep; every message is published with a confirm; SQS delays a message up to its 15 minutes, and one due later is sent again for the rest; SQS's queues are created when first used, keeping their messages 14 days rather than its default 4; the AWS SDK, for its chain of credentials, rather than requests of goncini's own.

> **Status (2026-10-19), step 6:** the notifier texts with Twilio and pushes with Firebase Cloud Messaging, through `httpclient`, without their SDKs, as the user chose. Decisions taken on the user's behalf: a recipient's `Phone` and `Devices`, rather than an address book of the notifier's; an SMS is the notification's subject, as Symfony's is, since an SMS should be short; a push has its subject as the title and its content as the body, one delivery per device, so that a device that fails doesn't send the others again; a typed section per provider, `Twilio` and `Firebase`, rather than DSNs, since Firebase's credentials are a JSON file; a service account's key, which the channel trades for an access token that it keeps for its hour, rather than Google's default credentials, which would need Google's library; a delivery that the API refused, with a 4xx other than 408 and 429, isn't retried, through `messenger.Unrecoverable`, which Symfony has as `UnrecoverableMessageHandlingException`; a device that is no longer registered is an error of its own, for the app to forget, rather than an event, which the notifier can't dispatch, as the subscribers that notify take the notifier.

> **Status (2026-10-19), step 7:** the profiler's routes are pages in a browser, its list of requests and each one's timeline, queries and logs, and still JSON for the others. The user decided the same day the forks of steps 7 to 11: HTML pages for the profiler; decoration and lazy services asked of effect-go, whose `layer` would own them, rather than built in goncini's generator; Symfony's constraints that go-playground lacks as tags of goncini's, with a table of those it has; Amazon SES, Brevo and Mailjet for the mailer; an HTML sanitizer and a semaphore. Decisions taken on the user's behalf: the same URLs as the JSON, chosen by the `Accept` header, so that a link from a log or `debug:requests` opens in a browser; `html/template` with the page embedded, without JavaScript, the details of a span unfolding with `<details>`; no toolbar, which an API's responses can't show.

> **Status (2026-10-19), step 8:** decoration and lazy services are written up as requests to effect-go, in §4.7, with a formatting bug that step 6 found; goncini builds them once `layer` has them.

> **Status (2026-10-19), step 9:** goncini's validator has tags for the constraints of Symfony's that go-playground lacks, and [validator/constraints.md](../validator/constraints.md) maps each of Symfony's to its tag. Decisions taken on the user's behalf: `regex`, `iban`, `isin`, `divisible_by`, `password_strength` and `not_compromised`, the constraints that APIs need most, rather than `Charset`, `WordCount`, `Week` or `NoSuspiciousCharacters`, which ICU's checks would need; `regex`, which go-playground left out as its patterns' commas and bars clash with the tags' syntax, with go-playground's escapes for them; an IBAN's check digits, without the lengths of each country's, which change; a password's strength estimated as Symfony does, so that its scores mean the same; `not_compromised` passes a password when Have I Been Pwned can't be asked, rather than keeping everyone from signing up, and asks for padded answers; its config in `validator.Config`, which `goncini.NewValidator` takes, so that `config/test.ego` keeps tests off the network.

> **Status (2026-10-19), step 10:** the mailer sends through Brevo's and Mailjet's APIs, and Amazon SES's, in the [mailer/ses](../mailer/ses) module, as the user chose. Decisions taken on the user's behalf: Brevo and Mailjet through `httpclient`, as the other APIs, without their SDKs; an email with more than one Reply-To is refused by those two, which take one, rather than sent without the others, as Symfony's Brevo transport does; Mailjet's inline parts as its inlined attachments, and Brevo's as attachments, as Brevo has no Content-IDs; SES's raw messages, which send the email as mailer writes it, with its parts and headers, rather than SES's simple content, which would rebuild it; the AWS SDK for SES, in a module of its own, for its chain of credentials, as the SQS transport does; `ses.Async` and `ses.Sync`, which give mailer's transport for the DSNs that aren't SES's, so that dev and tests keep `log://` and mailertest's, rather than a registry of transports that the module would fill as it is imported; `?configuration_set=` in the DSN, as Postmark's `?message_stream=`, rather than a header.

> **Status (2026-10-19), step 11:** semaphores are `lock.Semaphore`, and the [htmlsanitizer](../htmlsanitizer) module sanitizes users' HTML, as the user chose; Intl and ObjectMapper stay out. The backlog's steps are done, but for 8, which waits on effect-go. Decisions taken on the user's behalf: a semaphore's slots are locks of the lock store, tried in turn, so that every store keeps semaphores without a schema or a script of its own, at the cost of a request per slot taken, which suits limits of a few, rather than a store interface of semaphores; no weights, which Symfony's semaphores have and few apps use; the sanitizer in a module of its own, so that goncini requires neither bluemonday nor x/net; bluemonday's policy of user content as the default, rather than Symfony's, which keeps nothing until told, so that the zero config is the useful one; every URL is checked, even a strict policy's, which bluemonday doesn't do by itself; no elements removed from the default, which bluemonday can't, but a strict policy that keeps what it lists; no length of its own, which a validate tag bounds, rather than cutting the HTML as Symfony does; relative URLs removed when the hosts are restricted, as //host/path names one.

### 4.7 Asked of effect-go

Step 8 of the backlog needs effect-go's `layer`, whose resolver `ego
generate` runs, as the user decided on 2026-10-19. Its maintainers' session
wasn't running that day; these are the requests to give it.

1. **Decoration**, as Symfony's `#[AsDecorator]`: `layer.Decorate(NewCachedRepo)`,
   whose provider `func NewCachedRepo(inner articles.Repo, c *cache.Cache) articles.Repo`
   takes the service it wraps as its parameter of its own result's type.
   Every other provider that needs a `Repo` gets the decorator's; the
   decorated one, only the decorator. Several decorators of a type wrap in
   the order they are listed, the first nearest the original. goncini would
   use it to wrap a `mailer.Transport` with one that logs, a store with a
   cache, and a bundle's service with an app's.
2. **Lazy services**: a parameter of type `layer.Lazy[T]`, whose `Get()
   (T, error)` builds T the first time, with the dependencies that nothing
   eager built yet, once, its cleanup registered with the scope. goncini
   builds the whole app for each console command, `list` included: what
   only some commands or requests need, such as a client that loads a
   service account's keys, would wait for them.
3. **A formatting fix**: an expression-bodied lambda whose parameter's
   type is a struct literal that spans lines is lowered to
   `func(x struct {…}) bool { return … })` on one line, which `gofmt -l`
   reports, so the generated file fails a check that it is formatted:

   ```go
   func f(xs []struct {
   	A string `json:"a"`
   }) bool {
   	return slices.ContainsFunc(xs, x => x.A == "")
   }
   ```

   goncini names the type instead, in `notifier/push.ego`.

## 5. Risks

- **Tooling, once `goncini generate` comes.** Generation runs twice around `ego generate`, and stale generated files are the classic failure. If it's clumsy, ask effect-go for a hook between lowering and layer wiring; its maintainer session has offered to fix what a framework genuinely needs.
- **`layer` maturity.** goncini becomes its first wide graph. Keep goncini's generator thin, so that what it adds can move into `layer`.
- **Reflection in binding, validation and OpenAPI.** This is the cost of plain-Go routing. It runs once per route at startup; requests follow plans built in advance.
- **Go culture distrusts frameworks.** Every package stays usable on its own, everything is `net/http`, there's no global state, and leaving is cheap because generated code is plain Go.
- **Scope creep.** Symfony has about 50 components. The milestones list what's in; everything else waits for an app that needs it.
- **One reference app.** RealWorld is CRUD. Messenger and Scheduler need their own demo (M3).

## 6. Open questions

- Are error mappers registered on the kernel (proposed) or provided as services?
- Validation: struct tags plus `Validate()` (proposed), or rules in Go code only?
- Default error body: RFC 9457 problem+json (proposed)?
- Routing: host routes? Redirects between `/x` and `/x/` are ServeMux's alone. (Requirements on wildcards are built, as in Symfony.)

## Prior art

- **[Huma](https://github.com/danielgtaylor/huma) and [Fuego](https://github.com/go-fuego/fuego):** typed handlers and OpenAPI from Go types. The model for `web.JSON` and M2.
- **[wire](https://github.com/google/wire) (archived) and [fx](https://github.com/uber-go/fx):** compile-time and runtime dependency injection in Go. `layer` follows wire.
- **[API Platform](https://api-platform.com):** Symfony's API framework. The model for M2's contract features: OpenAPI, pagination, filters, problem details.
- **[RealWorld](https://github.com/gothinkster/realworld):** the reference app and its test suite.
