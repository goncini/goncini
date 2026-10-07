# Logging

goncini logs through `log/slog`, Go's standard logging API since Go 1.21:
what PSR-3 is to PHP. Code logs through a `*slog.Logger`; a `slog.Handler`
decides where records go and in what format. Every popular logging library
provides a handler, so an app picks its library by providing that handler.

goncini's `Framework` has two providers:

- `NewLogHandler`, the default handler: text records on stderr, or JSON
  with `Log.JSON`, from `Log.Level` up;
- `NewLogger`, the app's `*slog.Logger` on that handler. It adds the trace
  and span IDs of the context a record is logged with, so the records of a
  request carry its trace.

An app replaces the handler by passing its own provider to `layer.Build`,
and keeps the trace IDs:

```go
// app/inject.go
func Build(ctx context.Context, s *scope.Scope, cfg config.Config) (*goncini.App, error) {
	panic(layer.Build(goncini.Framework, Services, Autoconfigured, logHandler))
}
```

The kernel's access log and errors, the server's, `goncini.Main`'s and
`slog.Default()` then all go through it.

## Recipes

Each one compiles with the library's current version.

### zap and zerolog: adapter modules

`log/zaplog` and `log/zerologlog` are modules of their own, so that apps
that don't use those libraries don't depend on them. Each has a provider of
the library's logger, built from the app's `goncini.Log` section (the
level, and JSON or lines for people), and `NewHandler`, which replaces
goncini's handler when it is passed to `layer.Build`:

```go
var Services = layer.Set(zaplog.NewLogger, …) // a *zap.Logger, synced when the app stops

func Build(ctx context.Context, s *scope.Scope, cfg config.Config) (*goncini.App, error) {
	panic(layer.Build(goncini.Framework, Services, Autoconfigured, zaplog.NewHandler))
}
```

The services that take a `*zap.Logger` share the one the handler writes to.
[examples/articles](../examples/articles) logs this way. The recipes below
do the same by hand, for other setups.

### zap

```go
import (
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

func logHandler() (slog.Handler, func() error, error) {
	z, err := zap.NewProduction()
	if err != nil {
		return nil, nil, err
	}
	return zapslog.NewHandler(z.Core()), z.Sync, nil // flushed when the app stops
}
```

### zerolog

```go
import (
	"github.com/rs/zerolog"
	slogzerolog "github.com/samber/slog-zerolog/v2"
)

func logHandler() slog.Handler {
	z := zerolog.New(os.Stderr) // the handler adds the time
	return slogzerolog.Option{Logger: &z, Level: slog.LevelDebug}.NewZerologHandler()
}
```

### logrus

```go
import (
	sloglogrus "github.com/samber/slog-logrus/v2"
	"github.com/sirupsen/logrus"
)

func logHandler() slog.Handler {
	return sloglogrus.Option{Logger: logrus.StandardLogger()}.NewLogrusHandler()
}
```

### logr (Kubernetes' klog, controller-runtime, …)

```go
import "github.com/go-logr/logr"

func logHandler(l logr.Logger) slog.Handler {
	return logr.ToSlogHandler(l)
}
```

### OpenTelemetry logs

```go
import "go.opentelemetry.io/contrib/bridges/otelslog"

func logHandler() slog.Handler {
	return otelslog.NewHandler("github.com/acme/conduit") // to the global LoggerProvider
}
```

### Several at once

`slog.NewMultiHandler`, in the standard library, sends records to several
handlers, such as goncini's default one and OpenTelemetry's.

## Libraries that want their own logger

A library that takes a `*zap.Logger`, a `zerolog.Logger` or a `logr.Logger`
gets it from the app's container, like any other service: provide the
library's logger once, and build both the handler and the services that
need it from that provider.
