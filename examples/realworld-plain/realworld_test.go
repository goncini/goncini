package main

import (
	"log/slog"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestRealWorldSpec runs RealWorld's official Hurl suite against the API.
func TestRealWorldSpec(t *testing.T) {
	hurl, err := exec.LookPath("hurl")
	if err != nil {
		t.Skip("hurl is not on PATH")
	}
	files, err := filepath.Glob("../realworld/testdata/hurl/*.hurl")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no .hurl files in ../realworld/testdata/hurl")
	}

	db, err := openDB(t.Context(), filepath.Join(t.TempDir(), "realworld.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})

	logger := slog.New(slog.NewTextHandler(t.Output(), nil))
	srv := httptest.NewServer(newServer(db, []byte(strings.Repeat("k", 32)), logger))
	t.Cleanup(srv.Close)

	args := []string{
		"--test", "--jobs", "1",
		"--variable", "host=" + srv.URL,
		"--variable", "uid=" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	out, err := exec.CommandContext(t.Context(), hurl, append(args, files...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("hurl: %v\n%s", err, out)
	}
	t.Logf("%s", out)
}
