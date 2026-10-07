# What a goncini app looks like

*2026-10-06. Decided with the user: four layouts, from one package to hexagonal, with packages by feature as the default; the wiring in an `app` package; and config as `.ego` code in `config/`, with environment variables only for `APP_ENV` and secrets. The code below sketches the shape; the API comes with step 5 of the [plan](assessment.md#34-order-of-work-components-first).*

These are conventions, not requirements: an app laid out this way needs no settings to start, and one laid out otherwise only has to say where things are.

The RealWorld app, as `goncini new` would write it:

```
conduit/
├── go.mod                  # requires goncini, with a tool directive for ego
├── main.ego                # goncini.Main(config.Load, app.Build): the binary is also the console
├── .env                    # committed: APP_ENV=dev, and secrets that are safe for dev
├── .env.test               # committed: the secrets of the test environment
├── AGENTS.md
├── config/                 # package config: what Symfony keeps in config/packages/*.yaml
│   ├── config.ego          # type Config, and Load: the values of every environment
│   ├── dev.ego             # func dev(c *Config): what dev changes
│   ├── test.ego
│   ├── prod.ego
│   └── config_test.ego     # builds the app in every environment
├── app/                    # package app: the wiring, Symfony's Kernel.php and services.yaml
│   ├── inject.go           # //go:build egolayers: Build, from layer.Build
│   ├── services.ego        # var Services = layer.Set(...): the app's own providers
│   └── autoconfigured.ego  # routes, commands and config sections, until goncini generate writes it
├── articles/               # one package per feature
│   ├── articles.ego        # types, the ArticleError set and its problems
│   ├── controller.ego      # Routes, and the endpoints
│   ├── store.ego           # storage, through db.SQL.Conn and its transactions
│   ├── commands.ego        # the feature's console commands, if any
│   └── controller_test.ego # API tests through webtest, on the kernel booted with APP_ENV=test
├── users/
└── migrations/             # 0001_create_articles.sql: embedded, run by `conduit db:migrate`
```

Each `.ego` file sits next to the `_ego.go` file generated from it, and both are committed.

## Choosing a layout

The tree above is the second of four layouts. They differ only in how the app's own code is split into packages: `main.ego`, `config/`, `app/` and `migrations/` are the same in all four. Moving from one layout to the next means moving code between packages, and the framework doesn't notice. Routes, commands and config sections are found by their types (`routing.Routes`, `console.Command`), never by their folder.

| Layout | Packages | For |
|---|---|---|
| 1. One package | `api/` holds every feature | prototypes and small services |
| 2. By feature (default) | one package per feature, with its routes, errors and storage | most APIs |
| 3. Frontends, domains, infra | `api/` and `cli/` for the frontends, a domain package per feature, a package per integration | larger apps, or several frontends on one domain |
| 4. Hexagonal | domains that declare ports, and adapters that implement them | teams that want the domain to depend on nothing |

`goncini new` writes layout 2, or layout 1 on request. Layouts 3 and 4 are documented here and in a worked example, until an app needs `make:*` to write them.

Whatever the layout, packages that aren't the wiring don't import `app` or `config`, and Go's ban on import cycles enforces that. A package that needs settings declares its own section type, such as `articles.Config`. That section becomes a field of `config.Config`, and the container hands the package its section.

### 1. One package

```
conduit/
├── main.ego, config/, app/, migrations/   # as in every layout
└── api/                    # every feature
    ├── articles.ego        # Article, ArticleError and its problems, Routes, the endpoints
    ├── users.ego
    ├── store.ego           # storage, on db
    └── api_test.ego
```

Nothing to decide, and nothing to name. Once a file holds more than one feature's worth of code, it's time for layout 2.

### 2. By feature

The tree at the top. A feature package holds everything about one part of the API: its routes, endpoints, errors, storage and commands. This is the Go idiom, and it avoids names like `controller.ArticleController`. Symfony's own bundles were organized this way before `src/Controller` and `src/Entity` became the default.

### 3. Frontends, domains, infra

```
conduit/
├── main.ego, config/, app/, migrations/
├── api/                    # the HTTP frontend, for every feature
│   ├── articles.ego        # Routes, the endpoints, and the match from articles.Error to problems
│   ├── users.ego
│   └── articles_test.ego   # API tests
├── cli/                    # the console frontend: the app's commands
│   └── users.ego
├── articles/               # the domain: types, rules and the error set
│   ├── articles.ego
│   ├── service.ego         # calls postgres directly
│   └── service_test.ego
├── users/
├── postgres/               # one package per integration
│   ├── articles.ego        # queries, returning postgres's own row types
│   └── users.ego
└── stripe/
```

The domain doesn't know about HTTP or the console, but it calls its infrastructure directly, as a Symfony service calls a Doctrine repository. That saves the ports and adapters of layout 4, which most apps pay for and never use.

- **Infra doesn't import the domain.** `postgres` returns its own row types, and the domain turns them into its own; otherwise `articles` and `postgres` would import each other, which Go refuses.
- **Errors are mapped where they're seen.** The domain declares its error set and turns infra errors into its cases. `api/` maps the set to problems with an exhaustive `match`, so a new case without a response still fails the build.
- **Fakes need no ports.** Go interfaces are satisfied implicitly, so a domain package that wants a fake in its tests declares an interface for just what it uses, next to the code that uses it:

```go
// articles/service.ego

// rows is what Service needs from postgres.Articles, so tests can fake it.
type rows interface {
	BySlug(ctx context.Context, slug string) (postgres.Article, error)
}
```

That gives most of what hexagonal architecture offers, only where a test needs it.

### 4. Hexagonal

```
conduit/
├── main.ego, config/, app/, migrations/
├── articles/               # the domain: imports nothing but the standard library and effect-go
│   ├── articles.ego        # types, rules and the error set
│   ├── ports.ego           # what it needs, as interfaces: Repository, Clock
│   └── service.ego         # the use cases
├── users/
└── adapters/
    ├── api/                # driving: Routes, the endpoints, problems
    ├── cli/
    ├── postgres/           # driven: implements articles.Repository
    └── stripe/
```

The domain declares a port for everything it needs from outside, and adapters import the domain to implement them. The domain can be read and tested without any infrastructure, at the cost of a port, an adapter and a binding in `app/` for each one: `func(p *postgres.Articles) articles.Repository { return p }`. Go's implicit interfaces keep each one small, but layout 3 already gives most of the benefit for less.

## Config is code

```go
// config/config.ego
package config

type Config struct {
	HTTP     goncini.HTTP // address, trusted proxies and hosts, body limit
	DB       db.Config
	Articles articles.Config
}

// Load returns the config of env: the values of every environment, then
// the changes of env's own.
func Load(env *goncini.Env) Config {
	c := Config{
		HTTP:     goncini.HTTP{Addr: ":8080", TrustedProxies: []string{"private"}},
		DB:       db.Config{URL: env.Secret("DATABASE_URL"), PoolSize: 10},
		Articles: articles.Config{PageSize: 20},
	}
	switch env.Name {
	case "dev":
		dev(&c)
	case "test":
		test(&c)
	case "prod":
		prod(&c)
	default:
		env.Unknown() // APP_ENV names no environment: Main fails
	}
	return c
}
```

```go
// config/prod.ego
package config

func prod(c *Config) {
	c.HTTP.TrustedHosts = []string{"api.conduit.dev"}
	c.DB.PoolSize = 50
}
```

- **Config is only code.** There are no YAML, TOML or JSON config files, and goncini has no loader for them.
- **It's checked by the compiler.** A misspelled key or a wrong type fails the build, a renamed field is renamed in the config too, and the editor completes it. There's no parser and no decoding errors. Symfony compiles its YAML into the container anyway; here the config starts out as code.
- **An environment is a function** that changes what it needs on top of the shared values, the way `when@prod` does in Symfony. `APP_ENV` picks the function, and it defaults to `dev`. An `APP_ENV` that `Load` doesn't know fails at boot, so a typo such as `prdo` can't silently run with the shared values alone.
- **Environment variables are for secrets only, plus `APP_ENV`.** `env.Secret` is the only way to read one, so `config/` lists every variable the app reads. A missing secret fails at boot, and every missing secret is reported in one error, not one per restart.
- **`.env` files hold secrets for local work**, as in Symfony. goncini reads `.env`, `.env.local`, `.env.$APP_ENV` and `.env.$APP_ENV.local`, and the real environment overrides them. The `.local` files are gitignored.
- **Changing a setting means a new build.** Only secrets can change at deploy time. That is the price of type checking, and it's what Symfony's compiled container does in prod too.
- **`debug:config` prints the resolved config** as JSON, with the values of secrets masked.

## The wiring is a package

`app.Build` builds the app from the config: it is `layer.Build(goncini.Framework, Services, Autoconfigured)`. A test builds the app with `webtest.Boot`, or with an injector of its own that swaps a provider, such as the clock. Keeping the wiring out of package `main` lets tests boot the app's real kernel, the way Symfony's `KernelTestCase` does, since Go can't import `main`. That leaves `main.ego` with a single line:

```go
// main.ego
package main

func main() { goncini.Main(config.Load, app.Build) }
```

`goncini.Main` reads `APP_ENV` and the `.env` files, calls `Load`, and fails with every missing secret at once. It then builds the kernel in `scope.Main` and runs a command: `serve` by default, `debug:router`, `debug:config`, `db:migrate`, or one of the app's own.

## What isn't there

- **No `cmd/`:** the app is a single binary, and it is also the console.
- **No `internal/`:** an app is imported by nothing, so `internal/` would only add a level of folders.
- **No `public/`, `templates/` or `assets/`:** goncini is for APIs.
- **No `tests/`:** tests sit next to the code they test. API tests boot the app through `app.Build`.
- **No separate Go modules:** every layout is one `go.mod`. A module per feature would bring `go.work` files, `replace` directives and versions between features, which only pay off for code that is released on its own.
