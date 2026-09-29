# Psycho — agent instructions

## Project

Go module `psycho` (Go 1.27) — psychological profiling from text. Early stage.

## Build & verify

```
go build ./...
```

Run `go test ./...` for the unit and integration tests (under `modules/` and `test/`).

## Project layout

```
cmd/psycho/main.go         # entrypoint — starts HTTP server
modules/<name>/            # one flat Go package per domain module
  config.go                # YAML config structs
  dependencies.go          # wire deps, load config, construct services
  http.go                  # transport layer (could be grpc.go, cron.go, etc.)
  <action>.go              # one file per handler/operation
middleware/                # stdlib middleware stack (http.Handler adapter)
  chain.go                 # middleware.Chain(handler, Recovery, RequestID, ...)
  request_id.go            # RequestID, GetRequestID, RequestIDUnaryInterceptor (gRPC)
  timeout.go               # Timeout(TimeoutConfig{Duration})
  recovery.go              # d.Recovery() — depends on *zlogger.Logger
  request_validation.go    # DecodeAndValidate[T](r) — go-playground/validator tags
  dependencies.go          # Dependencies struct (holds logger)
config/
  config.go                # top-level Load(path string) (*Config, error)
  config.yaml              # default config
zlogger/
  zlogger.go               # wrapper around go.uber.org/zap
scripts/
  rename-module.sh         # renames Go module (old="brook" is hardcoded — update if used)
```

## Key conventions

- **No `internal/` packages** — modules stay flat.
- **No global state** — deps passed via closure or struct field.
- **Config** is YAML (`gopkg.in/yaml.v3`), loaded once per module in `dependencies.go` or the entrypoint file.
- **Logger**: `zlogger.New(level)` — `"dev"` = debug level, anything else = info. Console encoding. Usage: `logger.Info(ctx, msg, zlogger.Field{Key: "k", Value: "v"}, ...)`.
- **Middleware order** (outermost first): Recovery → RequestID → Timeout → Logger → Auth → RateLimit (see `middleware/chain.go`).
- **Validation**: `middleware.DecodeAndValidate[T](r)` — call inside handlers, uses `validate:"required,min=3"` struct tags.

## Container (Podman)

Multi-stage `Containerfile`, `CGO_ENABLED=0`, Alpine runtime. Run: `make up` (podman build + run) or `podman build .`

## Infrastructure

- `.env` is gitignored.
- `vendor/` is gitignored. Use `make vendor` when adding deps.
- Single commit so far.
