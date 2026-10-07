# Changelog

## Unreleased

goncini requires Go 1.27, and effect-go v0.2.2, which pins the library and
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
  behind `db:migrate`, `db:migrate:status` and `db:rollback`.
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

### Found while building

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
