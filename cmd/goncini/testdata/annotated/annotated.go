// Package annotated is an API whose endpoints goncini generate annotates,
// for its tests.
package annotated

import (
	"context"
	"net/http"

	"github.com/goncini/goncini/httpkernel"
	"github.com/goncini/goncini/routing"
)

// Thing is a thing.
type Thing struct {
	Name string `json:"name"` // its name
}

// Show returns a thing.
//
// Anyone may.
func Show(ctx context.Context, in struct{}) (Thing, error) {
	return Thing{}, httpkernel.Problem{Status: http.StatusConflict}
}

// Generic returns a T.
func Generic[T any](ctx context.Context, in struct{}) (T, error) {
	var t T
	return t, nil
}

// Things serves things.
type Things struct{}

// List lists the things.
func (t *Things) List(ctx context.Context, in struct{}) ([]Thing, error) {
	return nil, nil
}

// Routes routes the endpoints.
func Routes(r *routing.Router) {
	r.Get("/things/{id}", httpkernel.Endpoint(Show))
	r.Get("/generic", httpkernel.Endpoint(Generic[Thing]))
	r.Get("/things", httpkernel.Endpoint((&Things{}).List))
}
