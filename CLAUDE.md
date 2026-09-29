# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Psycho is a local-first Go service that extracts a psychological profile (Big Five/OCEAN, Regulatory Focus, Need for Cognition, cognitive style, Need for Closure, Schwartz values) from submitted text, using dictionary-based (LIWC-style) feature extraction — no LLM in the core inference path. Single-user, no auth, everything runs in one process against a local SQLite DB. See `docs/prd.md` for the full product spec (goals, non-goals) and `docs/system-design.md` for the references to the psychology literature each inference is based on.

## Build & run

```
go build ./...          # build
make run                 # go run ./cmd/psycho/
make test                 # go test ./... -v
go test ./modules/analyze/ -run TestName -v   # single test
make vendor               # go mod tidy && go mod vendor (after adding a dependency)
```

Config is loaded from `config/config.yaml` (see `config/config.go` for the schema). The server reads `cfg.App.HTTP.Port` etc. at startup in `cmd/psycho/main.go`.

Two manual end-to-end targets talk to a running server (`make run` first):
```
make test-curl            # POSTs modules/ingest's DirPath contents to /analyze-dir
make pdf ID=<analysis_id> # downloads the PDF for a saved analysis
```

## Architecture

**Modular monolith.** Each module under `modules/<name>/` is a flat Go package (no `internal/`) with a consistent file shape:
- `config.go` — YAML-tagged config struct for that module
- `dependencies.go` — `NewDependencies(cfg, logger) (*Dependencies, error)`: wires the module's own services (loads dictionaries, opens DB, runs migrations, etc.)
- `handler.go` — HTTP transport; handlers are constructed as closures via `MakeHandleX(...)` factory functions that take dependencies/callback functions as arguments, not a receiver struct
- one file per domain concern (e.g. `bigfive.go`, `regfocus.go`, `storage.go`, `narrative.go`)

Wiring happens one level up in `modules/server/http_server.go`: `NewHandler` builds each module's `Dependencies`, then registers routes on a stdlib `http.ServeMux`, threading cross-module glue through closures passed into `MakeHandleX` (e.g. the `/analyze` handler's callback takes `analyze` output and calls into `profileDeps.Aggregator`/`Storage`/`NarrativeGenerator` — modules never import each other's handler package directly for business logic, only `server` composes them).

**Request flow** (`POST /analyze-dir`, the primary path — reads `.txt` files from a configured directory rather than accepting arbitrary uploads):
1. `ingest` reads and concatenates files (`ReadDir`), enforces min/max size.
2. `ingest.Normalizer` strips markup, segments text, produces a `Document` with word count.
3. `analyze.FeatureExtractor` tokenizes against the loaded dictionary (`dictionary.json`, loaded once at startup in `analyze.NewDependencies`) to build a `FeatureVector` + coverage %.
4. `analyze.TraitModel.Infer` (Big Five, `bigfive.go`/`coefficients.go`) plus standalone `Compute*` functions for Regulatory Focus, Need for Cognition, Cognitive Style, Need for Closure, Schwartz Values — each is a pure function over the `FeatureVector`, independently testable.
5. `profile.ScoreAggregator.Aggregate` combines everything into a `Profile` with confidence flags; `profile.NarrativeGenerator` (template-based, no LLM) fills in prose.
6. `profile.Storage` (SQLite via `modernc.org/sqlite`, pure-Go/no CGO) persists the analysis as JSON blobs, keyed by a generated analysis ID.
7. `profile.PDFGenerator` (`pdf_maroto.go`, backend selected by `cfg.PDFBackend` in `profile.NewDependencies`) renders a stored analysis to PDF on demand via `GET /analysis/{id}/pdf`.

Middleware (`middleware/chain.go`, applied outermost-first): `Recovery` → `RequestID` → `Timeout`. Request bodies are decoded and validated together via `middleware.DecodeAndValidate[T](r)`, which uses `go-playground/validator` struct tags (see `ingest.AnalyzeDirRequest` for the pattern).

Every inference function is meant to be traceable to a cited source — check the References section in `docs/system-design.md` and the comment at the top of the relevant file (`coefficients.go`, `regfocus.go`, `needcog.go`, etc.) before changing scoring logic.

## Conventions

- No `internal/` packages; no global state — dependencies are passed explicitly via structs/closures.
- Config is YAML, loaded once per module in that module's `dependencies.go`.
- Logger: `zlogger.New(level)` (`"dev"` → debug, else info; wraps `zap`). Call sites: `logger.Info(ctx, msg, zlogger.Field{Key: "k", Value: v}, ...)`.
- `vendor/` is gitignored; run `make vendor` after editing `go.mod`.
- `scripts/rename-module.sh` has the old module name (`brook`) hardcoded — update it before reuse if the module is ever renamed again.
