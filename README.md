# Psycho

<p align="center">
  <img src="docs/images/report.png" width="70%" alt="The Psycho report: score bars for each measure with a one-line meaning, value categories with matched words in context, and a collapsible calculation section">
</p>

A small Go program for exploring language patterns through a dictionary-based psychological profiling heuristic. Reports expose the formulas and matching-word evidence behind recorded scores.

Inference is dictionary-based (LIWC-style), no LLM in the core inference path. It is compiled to WebAssembly and runs entirely in the browser: there is no server, no account and no database, and the text never leaves the visitor's device.

## Features

* Psycholinguistic feature extraction against a bundled dictionary, with normalisation and segmentation
* Trait inference: Big Five (OCEAN), Regulatory Focus, Need for Cognition, Need for Closure, cognitive style, and Schwartz value orientations
* Project-defined rough score ranges, not validated confidence intervals
* One HTML report with a "Read this first" caveat, each score's evidence trail, fit notes for text that suits the method poorly, and a downloadable calculation record (JSON). Self-writing consent is required
* "Save as PDF" through the browser's print dialog, "Save report" as one self-contained HTML file, and an opt-in history kept in the browser's IndexedDB

## Getting started

Requires Go 1.27. Generated CSS is committed; building the site requires no Node runtime.

```sh
make wasm          # runs scripts/build-site.sh: writes dist/ with index.html and content-hashed files under dist/a/
make serve-wasm    # builds, then serves dist/ at http://localhost:8081
make test          # go test ./... -v
```

Open the site over `http://` or `https://`; browsers refuse to load the `.wasm` file from `file://`. The bundle is about 8.5 MB raw and roughly 2.2 MB gzipped. Scores match the earlier server build exactly (checked on a sample with identical scores and ranks).

`make wasm` embeds the dictionary, `config/calibration.json`, the report templates and the compiled CSS into the bundle, so run it again after changing any of them.

Rebuild the CSS after changing templates, report color mappings, `site/` or the Tailwind input:

```sh
npm ci --ignore-scripts
npm run build:assets
```

Tailwind is pinned to 3.4.17. It scans the report templates, the Go report mappings and `site/`. The pinned version depends on `braces` 3.0.3, which has a build-time stack-exhaustion advisory (GHSA-vfj7-8cjw-p6xm) and no patched compatible release, so build inputs must remain project-controlled. Node is a build-time tool only; nothing from `node_modules` is shipped.

## Hosting on Cloudflare Workers

`wrangler.jsonc` serves `dist/` as static assets, with no Worker code. `site/_headers` sets a Content-Security-Policy that lets the page load only its own files and blocks network requests to other hosts.

```sh
npx wrangler login   # once
make deploy          # builds dist/ and runs wrangler deploy
```

The app is served at `https://psycho.chan179.com`, set by the `routes` entry in `wrangler.jsonc`. Cloudflare creates the DNS record, so `chan179.com` must be a zone in the same Cloudflare account.

## Documentation

* [docs/overview.md](docs/overview.md): plain-language tour, measured accuracy and the research references
* [docs/prd.md](docs/prd.md): product requirements (goal, non-goals, constraints, core features)
* [docs/offline-supervised.md](docs/offline-supervised.md): the fixed protocol for the offline supervised experiment

## Scoring and input contract

Inputs are normalized once in the pipeline: repeated Unicode whitespace becomes a single space within paragraphs, blank-line paragraph breaks are preserved, and markup tags are stripped. Fewer than 10 normalized Unicode characters or no letters/numbers is rejected before scoring. Text is capped at 1 MiB (about 170,000 words): the page shows the size and blocks analysis over the limit, and the analyzer checks again. Loaded files must be .txt or .md, under the limit and free of binary data. Saved readings get a title from the first line of the text (renamable) and are limited to the newest 50. Worst-case analysis at the limit takes about 130 ms natively.

Big Five scores use a correlation-weighted heuristic with assumed scaling, not fitted regression coefficients. Other measures are project-defined proxies. The optional `calculation_details` object records exact counts, denominators, weights, baselines, contribution order, calibration, clamping/rounding, summary formulas, value percentages, rough ranges and percentile operations. 

Calibration requires the active dictionary hash and model fingerprint, all nine dimensions, and 99 finite sorted quantiles per dimension. Regenerate after a dictionary or model change:

```sh
go run ./cmd/calibrate -corpus corpus -out config/calibration.json
```

## Offline supervised experiment

Train and evaluate five regularized logistic classifiers using a local Essays CSV:

```sh
go run ./cmd/train -csv corpus-eval/essays.csv -out testresults/supervised
```

The Go-only workflow uses pinned Gonum 0.17.0, separate fitting/calibration/test authors, and 5,000 paired bootstrap samples. It writes aggregate JSON/Markdown reports and a local model artifact. The probability target is the CSV's positive questionnaire label, with local provenance marked unverified. This experiment does not change production inference.

See [offline method and release considerations](docs/offline-supervised.md). Essays, author-level data, and model artifacts stay gitignored; only aggregate findings are committed.
