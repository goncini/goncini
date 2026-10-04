# Changelog

## Unreleased

goncini requires Go 1.27, and effect-go v0.2.1, which pins the library and
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
- **Startup checks:** conflicting routes, duplicate names, and paths ServeMux
  can't route panic at startup, saying where each route was registered.
- **Listing:** `List`, `Match` and `WriteTable` list the routes for
  `debug:router`; `Endpoint` handlers name their function there.
- **Cost:** serving through a `Router` costs what the ServeMux does; building
  a URL takes about 500 ns.

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
