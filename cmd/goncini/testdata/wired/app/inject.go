//go:build egolayers

package app

import (
	"context"

	"github.com/effect-go/effect-go/layer"
	"github.com/effect-go/effect-go/scope"
)

func Build(ctx context.Context, s *scope.Scope) (*Reports, error) {
	panic(layer.Build(NewDB, NewPgRepo, layer.Decorate(NewCounted), NewSMTP, NewMailer, NewReports))
}
