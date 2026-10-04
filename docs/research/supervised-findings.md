# Offline experiment findings, 2026-10-04

All five regularized logistic models and independent sigmoid calibrators converged. The experiment retained 2,442 of 2,467 essays, using 1,465 fitting authors, 488 calibration authors, and 489 final-test authors. Production scores and APIs are unchanged.

| Trait | Calibrated AUC | Heuristic AUC on the same authors | AUC difference, 99% paired interval | Calibrated Brier | Prevalence-baseline Brier |
|---|---:|---:|---|---:|---:|
| extraversion | 0.5843 | 0.5386 | [-0.0327, 0.1194] | 0.2453 | 0.2499 |
| neuroticism | 0.6202 | 0.5802 | [-0.0232, 0.1028] | 0.2426 | 0.2502 |
| agreeableness | 0.5487 | 0.5642 | [-0.0687, 0.0358] | 0.2486 | 0.2484 |
| conscientiousness | 0.5842 | 0.5346 | [-0.0265, 0.1310] | 0.2451 | 0.2502 |
| openness | 0.6255 | 0.5598 | [-0.0187, 0.1484] | 0.2378 | 0.2497 |

Every 99% paired AUC interval includes zero. This experiment therefore does not establish a ranking improvement over the heuristic at that interval level. Point estimates improve for four traits and decrease for agreeableness.

For neuroticism, conscientiousness, and openness, the 95% paired Brier and log-loss difference intervals are below zero. These are descriptive findings across multiple reported metrics, rather than an automatic release decision. Extraversion intervals include zero; agreeableness slightly worsens at the point estimate. Calibration slopes and reliability bins remain part of the evidence review.

The local dataset provenance, precise questionnaire cutoffs, and redistribution terms remain unverified. Essay-only results do not establish performance for general text or continuous personality scores. No trained model or participant rows are committed.

[Complete aggregate Markdown](supervised-essays.md) and [full-precision JSON](supervised-essays.json) record all settings, hashes, fitting statuses, reliability bins, diagnostic calibration fits, bootstrap intervals, and limitations. See [the protocol and later release requirements](../offline-supervised.md).

## Verification

- `go test ./...` passed, including existing API, storage, pipeline, PDF, and HTML report tests. Integration tests required loopback access beyond the default sandbox.
- `go build ./...` passed.
- Two complete corpus reruns produced byte-identical model artifacts and aggregate reports on Go 1.27.1.
- Legacy heuristic evaluation and dictionary-ablation commands completed with the shared strict reader.
- Supervised package test coverage: 87.8%; train command: 72.7%.
- Staged-file checks exclude local corpora, participant rows, and model artifacts.
