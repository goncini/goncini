# Dogfooding goncini

goncini's examples were written alongside the framework. To find what an
app's author runs into, [examples/heartbeat](../examples/heartbeat) is built
the way a user would build one: with `goncini new`, the makers and
AGENTS.md, then by hand. It is a cron-job monitoring API, after
Healthchecks.io: jobs ping it, and it alerts when a ping is late.

Each friction found is fixed in goncini, or in effect-go, or recorded here
as a decision.

## The app

- **Projects** own checks and alert channels; a project's API key
  authenticates its calls to `/api/v1`.
- **Checks** expect a ping every period, with a grace time. Their status
  is `new`, `up`, `down` or `paused`, moved by a workflow.
- **Pings** are public: `/ping/{id}`, `/ping/{id}/start` and
  `/ping/{id}/fail`, with the job's output as their body, rate limited.
- **A scheduled task** finds the checks whose ping is late, and marks them
  down. A status change between `up` and `down` is a flip.
- **Alerts:** each flip queues a message, whose handler notifies the
  project's channels: email, and webhooks.
- **Badges:** a public, cached status of a project's checks.

## Friction, and what was done

| # | Found | Done |
|---|---|---|
| 1 | `goncini new -replace ../..` wrote the checkout's absolute path in `go.mod`, which can't be committed | A relative checkout stays relative, from the app |
| 2 | The new app's `.gitignore` didn't ignore the binary that `go build` writes, which the skeleton's docs tell users to run | It does: `/<name>` |
| 3 | `security.Verifier.Verify` took no `ctx`, so an OIDC provider's keys couldn't be fetched; and an API key that the app keeps, by its hash, had no verifier | `Verify(ctx, token)`; and a firewall without `Tokens` hands the token to `Load`, which looks the key up |
| 4 | Endpoints only took JSON bodies; a ping's body is a job's output, of any type | A `Body []byte` or `Body string` field is the raw body, up to the kernel's limit; OpenAPI describes it as `*/*`, and the contract check matches media ranges |
| 5 | `make:feature checks` named its variables `check`, as in `check, ok := check c.store.Get(id)`: it compiles, but `check` is ego's keyword; and its docs said "a item" | The makers keep ego's keywords out of their names too, and their docs name the type: "a check", "an article" |
| 6 | Importing a goncini package whose dependencies the app's `go.mod` lacked, such as `security`'s JWT module, failed with "could not import … invalid package name" | The makers and `goncini generate` say that `goncini tidy` adds them |
| 7 | A `validate` tag on an `httpkernel.Optional` field panicked in go-playground: a 500 | The tags check the member's value, when it is present |
| 8 | A test couldn't run a scheduled task: `scheduler:run` runs until stopped | `scheduler:trigger <task>` runs it now, once, with now as its tick |
| 9 | `make:message` put goncini's `messenger` import in the app's group, in an app whose module goncini's owner owns too | An added import joins the first group that shares the most of its path |
| 10 | A `testing/synctest` bubble's time doesn't move while the app talks to `mailertest`'s or `httptest`'s servers: the alerts test, which waits for a retry, hung | Kept: that test runs in real time, as AGENTS.md now says. The webhooks step brings a receiver for tests that needs no socket |
| 11 | `make:command projects:create` refused a package that didn't exist yet | Kept: a maker doesn't make a package of a typo, and a command works on what its package has |

## Decisions in the app

- **SQLite's transactions take the write lock first**, with `_txlock=immediate`
  in `DATABASE_URL`: a ping reads its check, then writes it, in one
  transaction, which a deferred transaction can't upgrade while another
  writes.
- **The badge is invalidated after the transaction** that changes a
  status, as RealWorld's articles are: invalidated inside it, a badge
  computed before the commit would be cached with the old status.
- **A flip's alert is a message dispatched in the flip's transaction**, by
  a subscriber of `checks.Flipped`: the alert is queued only if the flip is
  saved, and the checks don't import the channels.
- **Deliveries make an alert idempotent:** a retried alert skips the
  channels it reached. Its webhook's `Idempotency-Key`, the flip's ID, is
  the same for each try, which lets the HTTP client retry the POST, and
  the receiver tell a retry.
