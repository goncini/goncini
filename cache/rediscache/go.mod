module github.com/goncini/goncini/cache/rediscache

go 1.27

tool github.com/effect-go/effect-go/cmd/ego

replace github.com/goncini/goncini => ../..

require (
	github.com/effect-go/effect-go v0.4.0
	github.com/goncini/goncini v0.1.0
	github.com/redis/go-redis/v9 v9.23.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	go.uber.org/atomic v1.12.0 // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/tools v0.50.0 // indirect
)
