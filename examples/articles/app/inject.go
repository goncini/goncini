//go:build egolayers

package app

import (
	"context"
	"time"

	"github.com/effect-go/effect-go/layer"
	"github.com/effect-go/effect-go/scope"
	"github.com/goncini/goncini"

	"github.com/goncini/goncini/examples/articles/config"
)

// Build builds the app for cfg.
func Build(ctx context.Context, s *scope.Scope, cfg config.Config) (*goncini.App, error) {
	panic(layer.Build(Framework, Services, Autoconfigured))
}

// BuildTest builds the app with a test's clock.
func BuildTest(ctx context.Context, s *scope.Scope, cfg config.Config, now func() time.Time) (*goncini.App, error) {
	panic(layer.Build(Framework, Services, Autoconfigured))
}
