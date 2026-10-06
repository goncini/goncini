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
| Language | **The framework is written in `.ego`** (decided), with the generated Go committed, so users never need `ego`. Its public API is callable from plain Go, where effect methods take `ctx` explicitly. |
| Ecosystem | **Wrap and wire, don't replace:** sqlc, goose, cobra, go-playground/validator, golang-jwt, modernc.org/sqlite. |

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
| Dotenv, Config, environments | Config as `.ego` code in `config/`: the shared values, then one function per environment, picked by `APP_ENV` (decided). Environment variables only for secrets, read through `env.Secret`, with Symfony's `.env` file precedence. No YAML. | M1 |
| Console | The app binary is the console (cobra). Commands are services, collected by autoconfiguration. | M1 |
| ErrorHandler, `HttpException` | Error sets mapped to RFC 9457 problems by exhaustive `match`; a panic becomes a 500 at the boundary; the renderer can be swapped. | M1 |
| Serializer, `#[MapRequestPayload]`, `#[MapQueryString]` | Typed handlers in `httpkernel`, `func(ctx, In) (Out, error)`. `In` is bound from the path, query, headers and JSON body; `Out` is encoded as JSON. | M1 |
| Validator | `validate` struct tags (go-playground/validator) and an optional `Validate()` method; violations become a 422. | M1 |
| Security | M1: password hashing, a token authenticator, the current user in `ctx`. M2: voters, and firewalls per route group. | M1, M2 |
| Doctrine ORM, Migrations | No ORM: `database/sql` with queries generated by sqlc, and goose migrations behind `db:migrate`. | M1 |
| Runtime, Stopwatch, Monolog | `scope.Main`, effect-go's spans, and `slog` records carrying trace IDs. | M1 |
| WebTestCase, KernelBrowser | `webtest`: an in-process client on the test graph, with `testing/synctest` for time. | M1 |
| NelmioApiDocBundle, API Platform | OpenAPI 3.1 generated from handler types; a docs page in dev; pagination and filtering helpers; responses checked against the contract in tests. | M2 |
| RateLimiter, CORS | Middleware. | M2 |
| EventDispatcher | Typed events (generics), with subscribers collected by autoconfiguration. | M3 |
| Messenger | Messages on effect-go: retries are schedules, workers are fibers in the app scope, transports for memory and SQL first. | M3 |
| Scheduler | `repeat` with a schedule, in the app scope. | M3 |
| HttpClient | The `net/http` client with `schedule.Retry` and a span per call. | M3 |
| WebProfilerBundle | A dev profiler built from the spans and log records effect-go already produces. | M4 |
| MakerBundle | `goncini make:*`, writing `.ego` files. | M4 |
| Bundles, Flex recipes | A bundle is a package exporting a `layer.Set`, a config section, routes and commands. | M5 |
| Cache, Lock, Mailer, Notifier, Translation, Workflow | When an API needs them. | Later |
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
- **`console`:** cobra, with commands as services implementing `console.Command`.
- **`goncini generate`** (last, and only once writing these providers by hand gets annoying): runs `ego generate`, then writes the `Autoconfigured` providers:
  - a slice for each framework interface: `routing.Routes` and `console.Command` in M1;
  - one provider per config section.

  It also writes the graph `debug:container` prints. effect-go's lowering type-checks the injector files, so a missing `Autoconfigured` must not break it. The order is: write a stub, run `ego generate`, load types, write the providers, run `ego generate` again to wire the layers.
- **`db`:** a `*sql.DB` provider closed with the scope, a transaction helper, and goose migrations embedded in the app. The reference app uses SQLite through modernc.org/sqlite, with no cgo.
- **`security` (lite):** password hashing, a token authenticator (RealWorld's `Authorization: Token …`), the current user in `ctx`, and a `Required` middleware. Its failures are an error set, mapped like any other.
- **`webtest`:** an in-process client on a test graph (`layer.Build(Services, …, NewMemStore)`), with assertion helpers.

### 3.3 Gates

1. **RealWorld's Hurl suite passes** against `examples/realworld`: all 13 files, error bodies included.
2. **The articles slice is at least 30% shorter** than the same slice written with plain `net/http` and hand wiring. We write both, as effect-go did for its demos.
3. **Adding a case to a mapped error set without an arm fails the build.**
4. **SIGTERM during a slow request:** the request completes, the database closes after it, and no goroutine is left. Tested under `synctest`.
5. **An agent given only goncini's AGENTS.md** adds an endpoint with validation and a new error case correctly on its first try, and hidden tests pass. This is the gate effect-go used.

### 3.4 Order of work (components first)

> **Status (2026-10-04):** steps 1, 2 and 4 are built: [`httpkernel`](../httpkernel) and [`routing`](../routing), used by [examples/articles](../examples/articles). [CHANGELOG.md](../CHANGELOG.md) says what they do, what building them found, and where they differ from this plan.

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

| Milestone | Content | Gate (draft) |
|---|---|---|
| **M2: API contract and security** | OpenAPI 3.1 from handler types (`/openapi.json` and a docs page in dev, `openapi:dump`); responses checked against the contract in `webtest`; pagination and filtering helpers; API versioning; voters; firewalls per route group; rate limiting; CORS | Our OpenAPI document is equivalent to RealWorld's `openapi.yml`, and a client generated from it (oapi-codegen) passes the Hurl suite's scenarios |
| **M3: Messenger and Scheduler** | Typed events; messages with memory and SQL transports; retries as schedules; `messenger:consume` workers as fibers; a failure transport; recurring tasks with `repeat`; outgoing webhooks; an HTTP client with retries | Killing a worker mid-message loses nothing; SIGTERM drains within the deadline |
| **M4: developer experience** | A profiler for API requests built from spans and logs (JSON endpoints, a `debug:requests` command, OTLP export to existing trace viewers); `goncini new`; `goncini make:*` | A new API goes from `goncini new` to a passing test in one command |
| **M5: ecosystem** | Bundles; cache; mailer; streaming responses (NDJSON, server-sent events) on effect-go's planned iterator support | A third-party bundle adds services, routes and commands without touching the app's wiring |

## 5. Risks

- **Tooling, once `goncini generate` comes.** Generation runs twice around `ego generate`, and stale generated files are the classic failure. If it's clumsy, ask effect-go for a hook between lowering and layer wiring; its maintainer session has offered to fix what a framework genuinely needs.
- **`layer` maturity.** goncini becomes its first wide graph. Keep goncini's generator thin, so that what it adds can move into `layer`.
- **Reflection in binding, validation and OpenAPI.** This is the cost of plain-Go routing. It runs once per route at startup; requests follow plans built in advance.
- **Go culture distrusts frameworks.** Every package stays usable on its own, everything is `net/http`, there's no global state, and leaving is cheap because generated code is plain Go.
- **Scope creep.** Symfony has about 50 components. The milestones list what's in; everything else waits for an app that needs it.
- **One reference app.** RealWorld is CRUD. Messenger and Scheduler need their own demo (M3).

## 6. Open questions

- Console: cobra, or a small package of our own?
- Are error mappers registered on the kernel (proposed) or provided as services?
- Validation: struct tags plus `Validate()` (proposed), or rules in Go code only?
- Default error body: RFC 9457 problem+json (proposed)?
- Routing: host routes, and requirements on wildcards? Without them, `/articles/abc` for an integer id is a 400 from binding rather than a 404. Redirects between `/x` and `/x/` are ServeMux's alone.

## Prior art

- **[Huma](https://github.com/danielgtaylor/huma) and [Fuego](https://github.com/go-fuego/fuego):** typed handlers and OpenAPI from Go types. The model for `web.JSON` and M2.
- **[wire](https://github.com/google/wire) (archived) and [fx](https://github.com/uber-go/fx):** compile-time and runtime dependency injection in Go. `layer` follows wire.
- **[API Platform](https://api-platform.com):** Symfony's API framework. The model for M2's contract features: OpenAPI, pagination, filters, problem details.
- **[RealWorld](https://github.com/gothinkster/realworld):** the reference app and its test suite.
