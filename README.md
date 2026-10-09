# Psycho

<p align="center">
  <img src="docs/images/report.png" width="70%" alt="The Psycho report: score bars for each measure with a one-line meaning, value categories with matched words in context, and a collapsible calculation section">
</p>

A small Go service for exploring language patterns through a dictionary-based psychological profiling heuristic. Reports expose the formulas and matching-word evidence behind recorded scores.

Inference is dictionary-based (LIWC-style), no LLM in the core inference path. Single-user, no auth, everything runs in one process against an embedded SQLite database.

## Features

* Text ingestion from a directory, with normalisation and segmentation
* Psycholinguistic feature extraction against a bundled dictionary
* Trait inference: Big Five (OCEAN), Regulatory Focus, Need for Cognition, Need for Closure, cognitive style, and Schwartz value orientations
* Project-defined rough score ranges, not validated confidence intervals
* Structured JSON output and PDF report export (`GET /analysis/{id}/pdf`)
* Single-report browser flow (Tailwind + HTMX): paste text at the root URL, the report swaps in on the same page, download it as PDF. The report opens with an "At a glance" summary and flags measures that fit the text poorly. Every score ships with its evidence trail, and self-writing consent is required
* Config file path overridable via `PSYCHO_CONFIG`

## Getting started

Requires Go 1.27. Generated browser assets are committed; serving and exporting reports requires no Node runtime.

Rebuild browser assets after changing templates, report color mappings, CSS or JavaScript:

```sh
npm ci --ignore-scripts
npm run build:assets
```

Tailwind is pinned to 3.4.17 and HTMX to 2.0.4 in the lockfile. Scripts and styles are served locally with CSP; standalone CLI HTML exports embed the compiled CSS. Tailwind scans the Go report mappings as well as HTML/JavaScript.

The pinned Tailwind version depends on `braces` 3.0.3, which has a build-time stack-exhaustion advisory (GHSA-vfj7-8cjw-p6xm) and no patched compatible release. Build inputs must remain project-controlled. Node dependencies are excluded from the production image; the deployed app serves only generated CSS and scripts.

```sh
make build   # go build ./...
make run     # go run ./cmd/psycho/ (serves on :8080)
make test    # go test ./... -v
```

Config lives in `config/config.yaml` (port, dictionary path, DB path, sample dir). With the server running:

```sh
make test-curl            # POSTs the configured samples dir to /analyze-dir
make pdf ID=<analysis_id> # downloads the PDF report for a saved analysis
```

## Container (Podman)

```sh
make up     # podman build + run on :8080
make down   # stop and remove the container
```

## Documentation

* [docs/overview.md](docs/overview.md): plain-language tour of what Psycho does and how to read a report
* [docs/prd.md](docs/prd.md): product requirements (goal, non-goals, constraints, core features)
* [docs/system-design.md](docs/system-design.md): architecture, storage, module boundaries, and the research references each inference is based on

## Scoring and input contract

Inputs are normalized once in the pipeline: repeated Unicode whitespace becomes a single space within paragraphs, blank-line paragraph breaks are preserved, and markup tags are stripped. Fewer than 10 normalized Unicode characters or no letters/numbers returns HTTP 400 before scoring or saving. Transport byte limits still apply.

Big Five scores use a correlation-weighted heuristic with assumed scaling, not fitted regression coefficients. Other measures are project-defined proxies. The optional `calculation_details` object records exact counts, denominators, weights, baselines, contribution order, calibration, clamping/rounding, summary formulas, value percentages, rough ranges and percentile operations. Old records without it show recorded results with an explicit notice.

Calibration requires the active dictionary hash and model fingerprint, all nine dimensions, and 99 finite sorted quantiles per dimension. Regenerate after a dictionary or model change:

```sh
go run ./cmd/calibrate -corpus corpus -out config/calibration.json
```

## Offline supervised experiment

Train and evaluate five regularized logistic classifiers using a local Essays CSV:

```sh
go run ./cmd/train -csv corpus-eval/essays.csv -out testresults/supervised
```

The Go-only workflow uses pinned Gonum 0.17.0, separate fitting/calibration/test authors, and 5,000 paired bootstrap samples. It writes aggregate JSON/Markdown reports and a local model artifact. The probability target is the CSV's positive questionnaire label, with local provenance marked unverified. This experiment does not change production inference or APIs.

See [offline method and release considerations](docs/offline-supervised.md) and the [measured aggregate findings](docs/research/supervised-findings.md). Essays, author-level data, and model artifacts stay gitignored; only aggregate findings are committed.
