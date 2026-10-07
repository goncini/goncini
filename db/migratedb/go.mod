module github.com/goncini/goncini/db/migratedb

go 1.27

tool github.com/effect-go/effect-go/cmd/ego

require (
	github.com/golang-migrate/migrate/v4 v4.20.1
	github.com/goncini/goncini v0.0.0
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/effect-go/effect-go v0.3.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/tools v0.50.0 // indirect
	modernc.org/libc v1.77.1 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
	modernc.org/sqlite v1.60.1 // indirect
)

replace github.com/goncini/goncini => ../..
