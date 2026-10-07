# Psycho System Design

## Software Architecture

**Style:** Modular monolith: components share a single process and database but have clear interface boundaries. No network calls between modules.

**Core flow**

1. User submits text (paste, file, URL). The ingest module normalises whitespace, strips markup, preserves paragraph breaks and validates the normalized text. Word tokenization supplies both scoring tokens and offsets for contextual excerpts; sentence meaning is not interpreted.
2. The normalised text passes to the analyze module, which tokenises and compares against a psycholinguistic dictionary. It computes category percentages, stylometric features, and a coverage rate.
3. The feature vector is fed to trait inference (Big Five correlation-weighted heuristic), Regulatory Focus, Need for Cognition, cognitive style classification, and value orientation mapping. Every output is stored with the feature evidence that produced it.
4. The profile module aggregates all scores, attaches rough score ranges, and generates structured output. Optionally, an external LLM call (user‑configurable, off by default) synthesises a narrative portrait from the structured scores.

### **Storage choice & why**

**Embedded SQLite**. Single‑user app with modest data volumes. No server process needed. Provides queryability for cross‑subject comparison and temporal tracking that flat JSON files would make cumbersome. The database file is portable; a user can back up their entire analysis history by copying one file.

### **Directory Structure**

```
cmd/
  psycho/main.go         # entrypoint, starts HTTP server
  rendertemplates/       # renders HTML report previews from an analysis JSON
  calibrate/             # derives config/calibration.json from a reference corpus
modules/
  ingest/                  # text ingestion module
    config.go              #   module-specific config struct
    dependencies.go        #   wire deps, load own config
    handler.go             #   POST /analyze-dir handler
    normalizer.go          #   text normalization logic
  analyze/                 # psycholinguistic analysis module
    config.go / dependencies.go
    dictionary.go          #   dictionary lookup engine (dictionary.json)
    features.go            #   feature extraction + project-defined summary proxies
    bigfive.go             #   Big Five correlation-weighted heuristic model
    coefficients.go        #   Yarkoni-inspired heuristic weights
    calibration.go         #   reference distribution: offsets + percentile quantiles
    regfocus.go            #   Regulatory Focus inference
    needcog.go             #   Need for Cognition inference
    need_closure.go        #   Need for Closure inference
    cognitive_style.go     #   cognitive style inference
    values.go              #   Schwartz value scores
    handler.go             #   POST /analyze handler
  profile/                 # profile generation module
    config.go / dependencies.go
    synthesizer.go         #   aggregation, rough score ranges, percentiles
    narrative.go           #   template-based narrative synthesis
    storage.go             #   SQLite persistence (modernc.org/sqlite)
    pdf.go / pdf_maroto.go #   PDF report generation
    handler.go             #   GET /analysis/{id} and GET /analysis/{id}/pdf handlers
  report/                  # the single HTML report
    report.go              #   view builder + rendering (fragment for HTMX, full page otherwise)
    form.go                #   POST /report handler (analyze + return the report inline)
  server/                  # composes all modules, registers routes
pipeline/                  # the analysis flow: normalize → extract → infer → persist
middleware/                # shared: recovery, request ID, timeout, validation
config/                    # YAML loader + config.yaml
samples/                   # .txt demo corpus read by /analyze-dir
templates/                 # index.html (Tailwind + HTMX upload page) and the report templates
test/                      # integration + known-profile validation tests
```

### **Module boundaries**

* **ingest**: Owns text normalisation, segmentation, and source metadata. Exposes a clean document object to downstream modules. Does NOT know about dictionaries, traits, or profiles.
* **analyze**: Owns the psycholinguistic dictionary, feature extraction, and trait inference models. Depends on ingest for clean text. Does NOT know about temporal comparison or narrative synthesis. Also owns the evidence trail: per-category contribution math (`evidence.go`) and the matched-word samples the extractor keeps. Owns the calibration reference (`calibration.go`): raw scores are centered and ranked against the distribution measured by `cmd/calibrate` over a reference corpus (`config/calibration.json`, committed; nil calibration falls back to the fixed 0.50 intercepts and the normal approximation).
* **profile**: Owns score aggregation, confidence computation, evidence attachment, and narrative generation. Depends on analyze for trait/feature data. Does NOT know about ingestion logic.
* **pipeline**: Owns stage ordering: normalizes, extracts, infers, aggregates, narrates, and persists in one `Run`. Depends on all three modules; exists so neither the HTTP server nor the tests duplicate the orchestration. The server and tests hand it to the handlers through the `ingest.AnalyzeFunc` seam.

### **Dependencies**

* **Go standard library:** `net/http`, `database/sql`, `encoding/json`, `text/template`
* **Open source:** `modernc.org/sqlite` (embedded database, pure Go, no CGO), `go.uber.org/zap` (logging), `go-playground/validator` (request validation), `johnfercher/maroto/v2` (PDF generation), `google/uuid` (analysis IDs), `google.golang.org/grpc` (gRPC request-ID interceptor in middleware)
* **Sidecar/optional:** A small LLM binary (e.g., Ollama) running locally if the user enables narrative synthesis. The app functions fully without it.

### **Abstraction Depth per Module**

**ingest**: No interfaces. Single implementation. Text normalisation is not swappable; the rules are the product.

**analyze**

* `Dictionary` interface. **Why abstracted:** Allows swapping between LIWC‑compatible lexicons without changing inference logic. Users may bring their own dictionary. The module exports `Lookup(word) → []Category` as the contract.
* `TraitModel` interface. **Why abstracted:** The heuristic model may be updated as new research publishes. The module exports `Infer(features) → BigFiveScores`.
* `FeatureExtractor` is NOT abstracted: single implementation. The features are dictated by the psycholinguistic literature, not user preference.

**profile**

* `NarrativeGenerator` interface. **Why abstracted:** Users may choose no LLM (template‑based), a local LLM (Ollama), or a cloud API (Gemini). The module exports `GenerateSynthesis(scores) → string`.
* `ScoreAggregator` is NOT abstracted: single implementation. The aggregation math is the product.

***

## Testing Strategy

Tests run after each phase completes. The system is decomposed so each module is testable independently without waiting for the full app.

**Unit Tests**

**What:** Domain logic. Normalisation rules. Dictionary lookup. Feature computation. Trait inference math.

**Examples:**

* "Text with 5.2% positive emotion words and 8.7% cognitive process words maps to predicted Openness percentile within expected range."
* "Corpus with <500 words returns low‑confidence flag regardless of feature values."
* "Normaliser strips HTML tags but preserves paragraph boundaries."
* "Dictionary coverage below 60% triggers warning flag."
* "High promotion_focus and low prevention_focus percentages map to elevated Regulatory Focus score."

**Integration Tests**

**What:** Module interactions. Full pipeline from text input to profile output.

**Examples:**

* "Submit 5,000‑word personal blog corpus → receive Big Five, Regulatory Focus, and Need for Cognition within 5 seconds. All 7 dimensions have valid scores and rough score ranges."
* "Submit text with 80% domain‑specific jargon → system returns low dictionary coverage warning and wide rough score ranges."
* Use test fixtures: pre‑prepared text samples with known linguistic profiles, embedded SQLite for test isolation.

**Current state:** All of the above exists. Beyond the unit and integration tests, `test/validation_test.go` runs text fixtures with known linguistic profiles through the pipeline (asserting exact category percentages, word-to-category placements, and the direction of every dimension) and records latency percentiles per corpus size. `test/calibration_test.go` pins the committed calibration: quantile lookup is monotonic and clamped, a corpus-average score maps to the 50th percentile, the JSON round-trips, and directionally opposite texts rank correctly through the calibrated pipeline. GitHub Actions runs gofmt, vet, build, and the full suite on every push.

### **Score calibration**

With calibration enabled, `cmd/calibrate` runs the production inference path over reference texts and writes `config/calibration.json`: per-dimension offsets that center the corpus mean at 0.50, plus 99 quantiles of adjusted scores. Startup validates dictionary and model fingerprints. The pipeline applies offsets before aggregation; percentile lookup interpolates quantiles and uses midpoints for ties, producing approximate ranks rather than a strict percentage of texts below a score. The committed artifact retains 3,992 texts from a 4,010-post Blog Authorship Corpus sample. Without calibration, percentiles use the documented normal-approximation assumptions. Heuristic bounds use assumed text-length and coverage rules; they are unvalidated diagnostics.

### Reading report and contextual evidence

The HTML report shows one 0–100 score per measure in compact rows, with canonical
band legends and native disclosure tooltips. Percentiles, heuristic bounds and
complete recorded calculations are collapsed into details. All nine dimensions
and four summary proxies remain available across full-page, HTMX and standalone
HTML rendering. Narrative and PDF use the same simplified score wording.

The optional `value_excerpts` field flows through analysis responses, profile JSON,
saved-analysis retrieval and report decoding without a database migration. Each
category has at most two distinct excerpts of up to 240 Unicode characters,
represented as `segments` of `{text, matched}`. Matching uses the existing
tokenizer and dictionary; excerpts cannot affect feature counts or scores. Go HTML
escaping handles every segment, and templates supply only the highlighting markup.
Value counts come from recorded calculation details. Legacy results identify
missing counts or excerpts rather than rebuilding them with today's dictionary.

### **Measured accuracy** (`cmd/evaluate`)

`cmd/evaluate` scores a labeled corpus with the production inference path (raw scores; calibration is monotone and cannot change ranking) and reports the Spearman rank correlation and AUC of each Big Five score against ground truth. On the local Essays CSV (2,467 source rows; 2,442 retained at a minimum of 200 normalized tokens; binary questionnaire labels; measured 2026-10-01, dictionary at 2,155 words / 36 categories). The dataset family is associated with Pennebaker & King (1999), but the exact local version and label cutoffs lack accompanying provenance:

| trait | Spearman ρ | AUC | AUC 95% CI (bootstrap) |
|---|---|---|---|
| neuroticism | 0.095 | 0.554 | 0.532 – 0.576 |
| agreeableness | 0.090 | 0.552 | 0.528 – 0.574 |
| extraversion | 0.074 | 0.542 | 0.520 – 0.565 |
| openness | 0.055 | 0.532 | 0.509 – 0.555 |
| conscientiousness | 0.042 | 0.524 | 0.503 – 0.547 |

The aggregate AUCs in this evaluation are modestly above chance. This does not validate individual category associations, prediction formulas, or rough score ranges. The scores are a correlation-weighted heuristic, not a fitted regression model. The evaluation applies only to this dictionary, corpus and labeling scheme.

**Breadth experiment (2026-10-01):** growing the dictionary from 1,471 to 2,155 words raised sample coverage from 55.8% to 57.8% but left the AUCs unchanged (all deltas inside overlapping bootstrap CIs). The discriminating signal in this corpus sits in closed-class function words (articles, prepositions, pronouns), which were already near-complete; generic content-word additions add coverage and evidence richness but not rank accuracy. The next lever is *discriminative* vocabulary (words selected because their usage varies with the traits, as LIWC's lists were), not more breadth for its own sake.

### Offline supervised experiment (`cmd/train`)

The Go-only experiment fits five regularized logistic classifiers with independent fitting, probability-calibration, and test authors. It preserves production scoring and records provenance as unverified. See [the fixed protocol and transition requirements](offline-supervised.md) and [the aggregate held-out findings](research/supervised-findings.md). Historical whole-corpus heuristic metrics above and the smaller supervised test sample must not be compared directly; the experiment measures both methods on identical test authors.

***

## References

These sources support language associations or psychological constructs. They do not validate this project's scoring formulas, assumed scales, dictionary, or uncertainty ranges. Big Five weights adapt zero-order correlations with assumed SD_trait=0.15 and SD_category=2.5 percentage points. Other text measures are project-defined proxies; summary sigmoid divisors and rough-range constants are also project assumptions.

#### Psycholinguistic Dictionary & Validation

* Pennebaker, J.W., Boyd, R.L., Jordan, K., & Blackburn, K. (2015). *The development and psychometric properties of LIWC2015*. University of Texas at Austin. – Research on dictionary-based language categories; the project uses its own lexicon and does not implement official LIWC summary algorithms.
* Fast, E., Chen, B., & Bernstein, M.S. (2016). *Empath: Understanding Topic Signals in Large‑Scale Text*. CHI 2016. – Open‑source alternative to LIWC with 194 categories, built on modern word embeddings. Background research only; the current default is the committed project dictionary.

#### Big Five & Language

* Yarkoni, T. (2010). *Personality in 100,000 words: A large‑scale analysis of personality and word use among bloggers*. Journal of Research in Personality. – Provides the Spearman correlations linking LIWC categories to Big Five traits, converted to per‑percentage‑point weights in `coefficients.go`.
* Pennebaker, J.W., & King, L.A. (1999). *Linguistic styles: Language use as an individual difference*. Journal of Personality and Social Psychology. – Foundational work establishing that function words (pronouns, articles) carry reliable personality signals.

#### Value Frameworks

* Schwartz, S.H. (1992). *Universals in the content and structure of values: Theoretical advances and empirical tests in 20 countries*. Advances in Experimental Social Psychology. – The Schwartz Value Survey, adapted for keyword co‑occurrence.

#### Cognitive Style & Motivation

* Petty, R.E., & Cacioppo, J.T. (1986). *The Elaboration Likelihood Model of persuasion*. Advances in Experimental Social Psychology. – Basis for systematic vs. intuitive processing markers.
* Webster, D.M., & Kruglanski, A.W. (1994). *Individual differences in need for cognitive closure*. Journal of Personality and Social Psychology. – Need for closure operationalised via certainty/tentative word ratios.
* Higgins, E.T. (1997). *Beyond pleasure and pain*. American Psychologist, 52(12), 1280‑1300. – Regulatory Focus Theory (promotion vs. prevention), implemented in `regfocus.go`.
* Cacioppo, J.T. & Petty, R.E. (1982). *The need for cognition*. Journal of Personality and Social Psychology, 42(1), 116‑131. – Need for Cognition scale, adapted for text markers in `needcog.go`.
