//go:build egolayers

package app

import (
	"context"

	"github.com/effect-go/effect-go/layer"
	"github.com/effect-go/effect-go/scope"
	"github.com/goncini/goncini"

	"github.com/goncini/goncini/examples/heartbeat/config"
)

// Build builds the app for cfg: goncini's services, the app's own, and
// those that goncini generate finds by their types.
func Build(ctx context.Context, s *scope.Scope, cfg config.Config) (*goncini.App, error) {
	panic(layer.Build(goncini.Framework, Services, Autoconfigured))
}
