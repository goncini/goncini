// Package app is an app whose injector decorates a service and builds
// another lazily, for debug:container's tests.
package app

import "github.com/effect-go/effect-go/layer"

type DB struct{}

func NewDB() *DB { return &DB{} }

type Repo interface{ Count() int }

type pgRepo struct{}

func (pgRepo) Count() int { return 0 }

func NewPgRepo(db *DB) Repo { return pgRepo{} }

type counted struct{ Repo }

// NewCounted decorates the Repo.
func NewCounted(inner Repo) Repo { return counted{inner} }

type SMTP struct{}

func NewSMTP() (*SMTP, error) { return &SMTP{}, nil }

type Mailer struct{}

func NewMailer(s *SMTP, db *DB) *Mailer { return &Mailer{} }

type Reports struct{}

// NewReports needs the Mailer only to send a report.
func NewReports(r Repo, m layer.Lazy[*Mailer]) *Reports { return &Reports{} }
