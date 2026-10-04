# Offline supervised Big Five experiment

This milestone tests whether learned dictionary-category weights improve on
the current heuristic. The server, JSON APIs, stored profiles, HTML reports,
and PDF exports continue using their existing scoring path.

## Reproduce

```sh
go run ./cmd/train -csv corpus-eval/essays.csv -out testresults/supervised
# Equivalent default command:
make train
```

Go 1.27 and the pinned Gonum 0.17.0 dependency are sufficient. No Python or
browser asset build is involved. Optional flags are `-dictionary`, `-min-words`,
`-seed`, and `-resamples`. Changed settings define a different experiment;
do not search seeds or settings against final-test performance.

The output directory contains `report.json`, `report.md`, and, when all model
fits succeed, `model.json`. Failed fits retain explicit errors and null metrics
in aggregate reports. Missing or malformed inputs return an error. A failed
input/run clears the previous model artifact in that output directory, so it
cannot be mistaken for the failed run's result.

## Data and target

The local CSV has 2,467 unique-author essays. At the default minimum of 200
normalized tokens, 2,442 remain and 25 are excluded. These counts describe the
local file, whose SHA-256 is recorded in each report; they are not universal
dataset sizes. The previous heuristic evaluation also used the retained 2,442.

Required columns are `AUTHID` (optionally prefixed by `#`), `TEXT`, `cEXT`,
`cNEU`, `cAGR`, `cCON`, and `cOPN`. Labels must be `y` or `n`, ignoring surrounding
space and case. The reader enforces CSV widths, rejects duplicate author IDs
and normalized essays, and validates text before any length filtering. Valid
UTF-8 takes precedence over Windows-1252 fallback; the detected encoding is
recorded. Input validation errors expose row numbers and column names, not text
or participant identifiers.

The target is **the CSV's positive questionnaire label**. The Essays dataset
family has been described as median-split questionnaire labels in the
[2013 shared-task paper](https://cdn.aaai.org/ocs/6190/6190-30315-1-PB.pdf).
This local file has no accompanying provenance or redistribution record, so
its precise version and label cutoffs remain unverified. Outputs therefore
avoid asserting an above-median target. Raw essays, author-level records, and
trained artifacts stay in gitignored directories while those terms remain
unverified. Committed findings contain only aggregate statistics.

## Fixed protocol

Authors are ordered by SHA-256 of `seed:author`; the first floor(60%) fit the
models, the next floor(20%) calibrate probabilities, and the remainder form the
final test set. The default seed is 42. The retained local corpus yields
1,465 fitting authors, 488 calibration authors, and 489 test authors. The
partition fingerprint is saved before tuning. Each observation has one unique
author, eliminating repeated-author leakage and repeated-prediction pooling.

Each trait uses all 36 existing category percentages in sorted feature order.
Categories overlap and need not sum to 100. Means and population standard
deviations are fitted only on the relevant training rows. Constant features
become zero, including when an unseen row has a different value.

Five-fold stratified validation inside the fitting partition selects lambda
from `{0.0001, 0.001, 0.01, 0.1, 1, 10}` by mean fold log loss. Numerical ties
within `1e-12` choose stronger regularization. Every fold fits its own scaling.
The objective is mean logistic loss plus `lambda/2 * sum(weight²)`; the
intercept is unpenalized. Deterministic Gonum L-BFGS uses one worker, an infinity
gradient tolerance of `1e-8`, and a 2,000-iteration limit. Only converged fits
are eligible; every fold's objective, gradient norm, and status are recorded.

Independent calibration fits `sigmoid(a*raw_logit+c)` using Platt's softened
targets, following [Niculescu-Mizil and Caruana, 2005](https://icml.cc/Conferences/2005/proceedings/papers/079_GoodProbabilities_NiculescuMizilCaruana.pdf).
Constant logits or single-class calibration are explicitly degenerate and
provide no calibrated prediction; optimizer failures never fall back silently.
The model is not refitted using calibration or test labels.

## Evaluation and interpretation

The final test reports original and calibrated AUC, Brier score, log loss,
calibration intercept/slope, and ten reliability bins including empty-bin
reasons and counts. Calibration intercept/slope are diagnostic test-set fits;
they never adjust test predictions. Log loss uses logits directly and remains
stable for extreme probabilities. Baseline probabilities use fitting-partition
prevalence. Current heuristic AUC is measured on the identical test authors;
heuristic scores are not treated as probabilities for Brier/log-loss comparisons.
AUC ranks predicted logits to preserve ordering when floating-point sigmoid
values saturate at zero or one.

All methods share 5,000 author bootstrap draws for paired differences. The
report provides 95% descriptive intervals and 99% paired AUC intervals for
each trait. The latter concern the five comparisons within a model variant;
they do not establish joint coverage of every statistic and both variants.
Intervals condition on the fixed trained models, assume independent test
authors, and exclude training and population-selection uncertainty. Undefined
statistics are null with reasons, not replacement values.

There is no automatic release decision. Slightly exceeding chance or a weak
heuristic does not establish useful individual predictions. Results on this
essay sample establish no accuracy claim for chats, emails, web pages, other
languages or populations. [Berggren et al., 2024](https://doi.org/10.1016/j.paid.2023.112465)
demonstrate the importance of testing personality models across text domains.
The cited papers support methodology, not this project's particular dictionary
or learned models. Learned contribution terms describe log odds and do not
provide causal explanations or psychometric validation.

## Artifact and later product transition

Local artifacts include version, target, provenance status, feature order,
standardization, learned coefficients, calibration parameters, fitting status,
protocol, and corpus/partition/dictionary/preprocessing/training fingerprints.
The offline loader rejects unknown fields, trailing JSON, missing traits,
invalid dimensions, incompatible fingerprints, and nonfinite parameters.
Prediction replay exposes each standardized category's logit contribution;
probabilities retain full precision without score rounding.

A later release requires evidence review, practical quality requirements,
verified target/provenance, applicable text domains and lengths, and permission
to distribute the intended artifacts. Production integration must update all
analysis endpoints, saved-profile decoding, narratives, PDF exports, browser
reports and standalone HTML together. Bump the profile JSON version and use
the existing `profile_version` column; legacy results retain their recorded
heuristic meaning and are never rescored. Model availability needs explicit
reasons, configured corrupt artifacts must fail startup, and every prediction
must record its inputs and calculation trace.

## Verification requirements

```yaml
- id: supervised-reproducibility
  type: outcome
  statement: Identical corpus, dictionary, settings and toolchain reproduce models and aggregate reports.
  fitness_function: Workflow and CLI rerun equality tests plus two complete local corpus runs.
- id: supervised-isolation
  type: invariant
  statement: Held-out labels and features cannot change fitting, tuning or fold scaling.
  fitness_function: Tests perturb calibration/test inputs and compare fitted models and tuning records.
- id: production-compatibility
  type: invariant
  statement: Existing analysis and report contracts retain current behavior.
  fitness_function: Existing pipeline, API, storage, PDF and report tests in go test ./....
- id: supervised-local-data
  type: constraint
  statement: Committed findings contain no participant rows or trained artifact.
  fitness_function: Aggregate-output leakage tests and staged-file inspection before commit.
```
