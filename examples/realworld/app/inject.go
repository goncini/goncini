//go:build egolayers

package app

import (
	"context"

	"github.com/effect-go/effect-go/layer"
	"github.com/effect-go/effect-go/scope"
	"github.com/goncini/goncini"

	"github.com/goncini/goncini/examples/realworld/conduit"
	"github.com/goncini/goncini/examples/realworld/config"
)

// Build builds the API for cfg. RealWorld has an error format and
// validation messages of its own, and authenticates every request.
func Build(ctx context.Context, s *scope.Scope, cfg config.Config) (*goncini.App, error) {
	panic(layer.Build(goncini.Framework, Services, Autoconfigured, conduit.NewRenderer, conduit.NewValidator, middleware))
}
