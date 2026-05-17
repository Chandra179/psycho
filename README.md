# Brook

Go modular monolith skeleton. One binary, domain modules as Go packages. Split to microservices later — not before.

## Layout

```
cmd/example/main.go   # entrypoint — starts HTTP + gRPC
modules/              # domain modules
  example/            #   example module
    config.go         #     module-specific config struct
    dependencies.go   #     wire deps, load own config
    http.go           #     HTTP handlers + route registration
middleware/           # shared: recovery, request ID, timeout, validation
config/               # YAML loader + config.yaml
```

## Module pattern

Each module under `modules/<name>/` is flat Go package. Owns domain logic, transport (HTTP/gRPC), DI, and config.

Module defines own `Config` struct in `config.go`. Config loaded via `loadConfig(path)` in `dependencies.go` — opens YAML, unmarshals module's section. Zero import of shared `config/`.

Shared `config/config.yaml` nests module sections under module key:

```yaml
example:
  http:
    port: "8080"
    ...
middleware:
  timeout_in_second: 30
```

No inter-module coupling — modules call shared infra (`middleware/`), not each other.

## Config decoupling

Each module owns its config shape. Means 3 lines of YAML load boilerplate per module. Tradeoff: migration = copy module dir, zero dependency on shared config types.

## Adding a new module

1. **Copy the example skeleton**:
   ```bash
   cp -r modules/example modules/<name>
   ```

2. **Rename the package** inside all three files (`config.go`, `dependencies.go`, `http.go`):
   ```go
   package example  →  package <name>
   ```

3. **Write your own config** in `config.go` — replace the placeholder structs.

4. **Wire deps** in `dependencies.go` — replace `NewDependencies` with your own constructor.

5. **Add handlers** in `http.go` — register routes on the mux:
   ```go
   mux.HandleFunc("POST /<name>", HandleCreate)
   ```

6. **Create an entrypoint** at `cmd/<name>/main.go`:
   ```go
   package main

   import "brook/modules/<name>"

   func main() {
       <name>.RunHttpServer()
   }
   ```

7. **Add config section** in `config/config.yaml`:
   ```yaml
   <name>:
     http:
       port: "8080"
       read_timeout_in_second: 5
       write_timeout_in_second: 35
       idle_timeout_in_second: 120

   middleware:
     timeout_in_second: 30
     logger:
       level: dev  # dev | prod
   ```

Module owns its config. No shared config types. Zero coupling to other modules.

## Renaming the project

This skeleton uses `brook` as the Go module name. When cloning for a new project,
rename it with:

```bash
scripts/rename-module.sh <new-module-name>
```

Example:

```bash
scripts/rename-module.sh github.com/myorg/myproject
```

Updates `go.mod` and all import paths (e.g. `"brook/middleware"` → `"github.com/myorg/myproject/middleware"`).
Run `go build ./...` after to verify.

## Commands

```bash
make vendor          # go mod tidy && go mod vendor
go run cmd/example/main.go
go build ./...
go test ./modules/...
```

## State

Mid-restructure. Single module (`example`). One entrypoint binary. Basic test coverage on config loading + DI wiring.

## Design choices

- Validation via `middleware.DecodeAndValidate[T](r)` inside handlers
- No `internal/` sub-packages inside modules
- Config struct per module, YAML section per module key
- No global state — deps injected via closure or struct field
