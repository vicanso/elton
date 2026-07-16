# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Elton is a Go web framework (module `github.com/vicanso/elton/v2`, requires Go 1.24+) inspired by koa/echo. Two packages: the core framework at the repo root, and `middleware/` with all built-in middleware. Documentation is primarily in Chinese; commit messages follow conventional commits in lowercase English (`feat:`, `fix:`, `refactor!:` for breaking changes).

## Commands

```bash
make test                      # go test -race -cover ./...
go test -race -run TestProxy ./middleware/   # single test
make bench                     # benchmarks
make lint                      # golangci-lint run
go test -fuzz=FuzzNewCacheResponse -fuzztime 10s ./middleware/  # fuzz a binary decoder
```

Always run tests with `-race` (CI does). Verify `gofmt -l .` is clean before finishing.

## Architecture

### Onion middleware model (do not change the .Next chaining logic)

Everything is a `Handler func(*Context) error`. At route registration, global middleware (`e.Use` at that moment) is snapshotted with route handlers into a fixed chain; each handler calls `c.Next()` (stable `boundNext` → `chainNext`, no per-request closure). Request flows inward, response outward. Errors return up the chain instead of writing responses directly. `Context.Committed` short-circuits: once true, remaining processing is skipped and the response is considered written. Call `Use` before registering routes that need those middlewares.

Response contract: handlers set `c.Body` (any) and/or `c.StatusCode`; a responder middleware (e.g. `middleware.NewDefaultResponder`) converts `Body` to `c.BodyBuffer` (*bytes.Buffer), which the framework writes out in `elton.go` after the chain returns. If `Body` is an `io.Reader` it is streamed via Pipe; if it implements `io.Closer` and is *not* piped (error path, or `BodyBuffer` set), the framework closes it (`Context.closeReaderBody`).

Key core files: `elton.go` (Elton instance, route registration, lifecycle `ListenAndServe/Shutdown/GracefulClose`, events `OnBefore/OnDone/OnError`, Context pooling via `sync.Pool`), `context.go` (Context; generic `GetContextValue[T]`; signed cookies via keygrip), `route.go` (path normalize, param names, `edgeWriter` for single ServeMux match + custom 404/405), `fresh.go` (zero-alloc HTTP freshness check). Routing uses stdlib `http.ServeMux` (Go 1.22+ patterns: `{name}`, `{name...}`, method match); `c.Param` / `RouteParams` are filled from `Request.PathValue`. `ServeHTTP` calls `mux.ServeHTTP` once; elton routes `markHandled` so mux-internal 404/405 can be replaced without a second match.

### Middleware conventions

- Constructor pattern: `New<Name>(Config) elton.Handler`; `NewDefault<Name>()` means zero-config, `NewFS...`/other prefixes mean a specific variant. Config structs embed a `Skipper elton.Skipper`; use `getSkipper` from `middleware/helper.go`.
- Middleware config validation errors panic at construction time (routes are registered before serving); request-time failures return errors.
- Errors use `github.com/vicanso/hes` v1.0.0+. Treat `*hes.Error` as **immutable**: never mutate fields after `hes.Wrap(err)` (without opts it returns the original pointer from the chain); use options (`hes.WithStatus/WithCategory/WithException/WithCause`) or `wrapAsHesError(err, category)` from `helper.go`. `hes.Wrap` defaults non-hes errors to 500 + `Exception=true`; expected errors (e.g. auth failure) should use `hes.New` instead.
- Naming trap: `Context.GetHeader/SetHeader` operate on the **response**; use `GetRequestHeader/SetRequestHeader` for the request (opposite of gin).

### Hand-written binary formats (fuzz-covered)

`middleware/cache.go` encodes cached responses as `[status(1)][createdAt(4)][statusCode(2)][headerLen(4)][headers][compression(1)][body]` with a fetch/hit/hit-for-pass state machine; `middleware/http_header.go` compact-encodes headers using a short-header index dictionary. Decoders must tolerate corrupted/truncated input without panicking — extend `middleware/fuzz_test.go` when touching them.

### Tests

Table-driven with `testify/assert`, using `httptest`. Several tests assert exact `hes.Error` string formats (`statusCode=..., category=..., message=..., exception=true`), so error-construction changes usually require test updates.

## Breaking changes policy

Any breaking API change must be recorded in `docs/migration-v2.md` (rename tables, behavior changes, v1 defects fixed). `docs/introduction.md` quotes middleware source code — keep those snippets in sync when changing `responder.go`/`error.go`.
