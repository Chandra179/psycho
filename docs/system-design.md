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
* "Dictionary coverage below 60% lowers reading quality to medium, and the report says why."
* "High promotion_focus and low prevention_focus percentages map to elevated Regulatory Focus score."

**Integration Tests**

**What:** Module interactions. Full pipeline from text input to profile output.

**Examples:**

* "Submit 5,000‑word personal blog corpus → receive Big Five, Regulatory Focus, and Need for Cognition within 5 seconds. All 7 dimensions have valid scores and rough score ranges."
* "Submit text with 80% domain‑specific jargon → system returns low dictionary coverage warning and wide rough score ranges."
* Use test fixtures: pre‑prepared text samples with known linguistic profiles, embedded SQLite for test isolation.

**Current state:** All of the above exists. Beyond the unit and integration tests, `test/validation_test.go` runs text fixtures with known linguistic profiles through the pipeline (asserting exact category percentages, word-to-category placements, and the direction of every dimension) and records latency percentiles per corpus size. `test/calibration_test.go` pins the committed calibration: quantile lookup is monotonic and clamped, a corpus-average score maps to the 50th percentile, the JSON round-trips, and directionally opposite texts rank correctly through the calibrated pipeline. GitHub Actions runs gofmt, vet, build, and the full suite on every push.

### **Score calibration**

With calibration enabled, `cmd/calibrate` runs the production inference path over reference texts and writes `config/calibration.json`: per-dimension offsets that center the corpus mean at 0.50, plus 99 quantiles of adjusted scores. Offsets and quantiles come from the unrounded model scores, and the pipeline ranks the unrounded calibrated score, so texts whose displayed scores match can still land on different ranks (the quantile tables hold 73 to 99 distinct values per dimension instead of 7 to 41). Startup validates dictionary and model fingerprints. The pipeline applies offsets before aggregation; percentile lookup interpolates quantiles and uses midpoints for ties, producing approximate ranks rather than a strict percentage of texts below a score. The committed artifact retains 3,992 texts from a 4,010-post Blog Authorship Corpus sample. Without calibration, percentiles use the documented normal-approximation assumptions. Heuristic bounds are per measure: each score is baseline plus the mean over words of a per-word contribution (100 times the summed weights of the word's categories), so `FeatureExtractor` records its sampling standard error `sd(contribution) / sqrt(words)` (`analyze/sampling.go`). The half-width is 1.96 standard errors, limited to 0.01 to 0.25. Independent words understate the real spread, so each standard error is multiplied by a measured factor (1.05 to 1.75 by dimension, `seInflation` in `analyze/sampling.go`): the ratio of the actual score difference between the first and second halves of the 375 reference posts of at least 800 words to the predicted one. This checks repeatability within a text, not accuracy against a person's true trait, and is not a confidence interval. Records without the recorded spread fall back to the older shared length rule.

### Scoring rules worth knowing

* **Long words** (seven letters or more) are counted in letters, not UTF-8 bytes, and only feed the formal-prose note. No score uses them.
* **Cognitive Style** is the Pennebaker et al. (2014) function-word index: article and preposition percentages up, personal pronouns, impersonal pronouns, auxiliary verbs, adverbs, conjunctions and negations down, each at weight 0.01 per point with baseline 0.70 (the index averages about -19.7 on the reference corpus). The weight scale is a project assumption. The labels stay "systematic" (categorical) and "intuitive" (dynamic).
* **Authenticity** no longer subtracts long words, and its numerator is centered by subtracting 20.7, the reference median, because the long-word term had been what kept it off the ceiling.
* **Negation:** a Schwartz value word within three tokens after a negator ("did not care", "wasn't generous") is not counted toward that value, and the excerpt picker applies the same rule (`analyze/matching.go`). Trait categories keep plain counts of the words that remain, because their weights come from plain-count correlations; ambiguous words are removed from the lists instead (Word-sense audit below). For Emotional tone, a negated emotion word moves to the opposite side ("not happy" is negative, "not bad" is positive), and the tone band is positive from 0.55 and negative up to 0.45.
* **Excerpts** are the two sentences with the most distinct matched words per value, not the first two.
* **Print noise:** `Normalize` drops bare page numbers, figure and table captions and running page headers (a 20-character or longer line without end punctuation repeated three or more times), and rejoins words hyphenated across long lines.
* **Dictionary:** value lists no longer include everyday words such as "just", "kind", "content", "natural", "care", "support", "rule", "order" or "control"; the negative-emotion list gained words such as "cried", "scream", "scam", "garbage" and "disaster".
* **Word-sense audit** removes 18 word entries from the scored content-word lists ("just", "will", "so", "as", "up", "over", "right", "world", "through", "way", "take", "kind", "all", "sense" and a few more) because most of their uses in the reference posts are not the category's meaning. For each ambiguous word that makes up at least 4% of its category, 20 random contexts were judged by one annotator and the word was removed when fewer than 10 used the category's sense. Words that passed ("only", "may", "go", "since", "good", "know" and others) are listed as audited in `analyze/dictionary_exclusions.json`; words under 4% of their category or judged unambiguous by definition were not sampled. The record is word by word and a test fails if the dictionary regains an excluded word or loses an audited one. Function-word categories (Cognitive Style), pronouns and the value lists are not audited. The audit lowers mean dictionary coverage on the reference posts from 66.5% to 65.5% and the matches of "cause" by 73%, "space" by 49%, "exclusive" by 30%. Coverage therefore means the share of words that were scored.
* **Rescaled by the audit:** Authenticity's centering constant (19.2 to 20.7, its new reference median). The bounds scaling factors were re-measured and stayed within 4% of their old values, the 45% low-coverage line is still about the 2nd percentile (47.5%), and the 60% line sits at about the 19th percentile instead of the 13th.
* **Quality flag:** under 500 words or under 45% dictionary coverage is low (45% is below the 2nd percentile of the reference corpus); under 1,000 words or under 60% coverage is medium.

### Reading report and contextual evidence

The HTML report no longer shows an "At a glance" section (removed on 2026-10-09); `report.BuildGlance` still builds the same recorded fields ( size and dictionary coverage, the reasons the reading
quality is not high (`analyze.QualityReasons`, which shares its thresholds with
`profile.computeConfidenceFlag` through `analyze.QualityFlag`), recorded
emotion-word counts, fit notes, and one plain caveat (`analyze.ReadingCaveat`)) for the fit notes on the cards.
Each measure is a compact 0–100 row with canonical band legends,
native disclosure tooltips, and a one-line meaning (`analyze.MeasureSummary`),
with no rank sentence (removed on 2026-10-09; the percentile is still stored and shown in the calculation details). Calibrated scores for most dimensions cluster within
about 0.44 to 0.56 in the reference sample, while the 35/65 bands are fixed on
the score scale. `report.FitNotes` flags measures that mostly reflect register or
topic (formal prose: Authenticity at or below 0.30, Cognitive Style at or above 0.60 and at least 25% long words; this flags 1.9% of reference posts)
or lack signal (emotion words under 0.5% of the text); thresholds were set from
the nine samples plus 25 random corpus posts. Heuristic bounds and complete
recorded calculations stay collapsed. Below 768px the evidence table is replaced
by a stacked list. All nine dimensions and four summary proxies remain available
across full-page, HTMX and standalone HTML rendering. Narrative and PDF use the
same simplified score wording but do not yet include the glance block.

The optional `value_excerpts` field flows through analysis responses, profile JSON,
saved-analysis retrieval and report decoding without a database migration. Each
category has at most two distinct excerpts of up to 240 Unicode characters,
represented as `segments` of `{text, matched}`. Matching uses the existing
tokenizer and dictionary; excerpts cannot affect feature counts or scores. Go HTML
escaping handles every segment, and templates supply only the highlighting markup.
Value counts come from recorded calculation details. Legacy results identify
missing counts or excerpts rather than rebuilding them with today's dictionary.

### **Measured accuracy** (`cmd/evaluate`)

`cmd/evaluate` scores a labeled corpus with the production inference path (raw scores; calibration is monotone and cannot change ranking) and reports the Spearman rank correlation and AUC of each Big Five score against ground truth. On the local Essays CSV (2,467 source rows; 2,442 retained at a minimum of 200 normalized tokens; binary questionnaire labels; measured 2026-10-09 with the dictionary and rules of that date). The dataset family is associated with Pennebaker & King (1999), but the exact local version and label cutoffs lack accompanying provenance:

| trait | Spearman ρ | AUC | AUC 95% CI (bootstrap) |
|---|---|---|---|
| neuroticism | 0.103 | 0.559 | 0.537 – 0.581 |
| agreeableness | 0.103 | 0.559 | 0.535 – 0.583 |
| extraversion | 0.075 | 0.542 | 0.521 – 0.566 |
| openness | 0.058 | 0.533 | 0.511 – 0.556 |
| conscientiousness | 0.056 | 0.532 | 0.510 – 0.554 |

The aggregate AUCs in this evaluation are modestly above chance. This does not validate individual category associations, prediction formulas, or rough score ranges. The scores are a correlation-weighted heuristic, not a fitted regression model. The evaluation applies only to this dictionary, corpus and labeling scheme.

The 2026-10-09 scoring fixes (letter-based word length, dictionary pruning and additions, print-noise cleanup) moved the AUCs by at most 0.004, so they improve correctness and legibility, not predictive accuracy.

The word-sense audit of the same day (the table above) was compared with the earlier lists on the same essays, with the word lists fixed before any accuracy was computed: mean AUC 0.542 before and 0.545 after, every change inside the bootstrap intervals. It is justified by what the counted words mean, not by accuracy.

A second step was tried and reverted: the Schwartz et al. (2013) top-sense probability filter (theta 0.50, WordNet 3.0 tag counts) removed 418 more word entries, cut mean dictionary coverage from 65.5% to 63.0% and moved mean AUC to 0.540 (extraversion 0.536, neuroticism 0.561, agreeableness 0.551, conscientiousness 0.533, openness 0.520), again inside the intervals. It also removed many clearly in-sense words ("excited", "pain", "fear"), because it penalises any word with several meanings, and its recall cost suits corpora of millions of posts more than one 500 to 1,000 word text. It is not in the product.

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

* Pennebaker, J.W., Chung, C.K., Frazee, J., Lavergne, G.M., & Beaver, D.I. (2014). *When small words foretell academic success: The case of college admissions essays*. PLoS ONE, 9(12), e115844. – The categorical-versus-dynamic function-word index behind Cognitive Style (articles and prepositions up; personal and impersonal pronouns, auxiliary verbs, adverbs, conjunctions and negations down). Validated against college grades, not personality.
* Schwartz, H.A., Eichstaedt, J., Dziurzynski, L., Kern, M.L., Blanco, E., Kosinski, M., Ungar, L., & Seligman, M.E.P. (2013). *Choosing the right words: Characterizing and reducing error of the word count approach*. \*SEM 2013. – Measured 70.4% precision for LIWC emotion words in Facebook posts, with lexical ambiguity the most common error; the reason value lists were pruned of everyday words; its top-sense probability filter was tried on the trait lists and not kept (see Measured accuracy).
* WordNet 3.0 Copyright 2006 by Princeton University. All rights reserved. – used only in the reverted top-sense experiment; no WordNet data or derived scores are stored in this repository.
* Petty, R.E., & Cacioppo, J.T. (1986). *The Elaboration Likelihood Model of persuasion*. Advances in Experimental Social Psychology. – Background for the earlier systematic vs. intuitive wording of the labels.
* Webster, D.M., & Kruglanski, A.W. (1994). *Individual differences in need for cognitive closure*. Journal of Personality and Social Psychology. – Need for closure operationalised via certainty/tentative word ratios.
* Higgins, E.T. (1997). *Beyond pleasure and pain*. American Psychologist, 52(12), 1280‑1300. – Regulatory Focus Theory (promotion vs. prevention), implemented in `regfocus.go`.
* Cacioppo, J.T. & Petty, R.E. (1982). *The need for cognition*. Journal of Personality and Social Psychology, 42(1), 116‑131. – Need for Cognition scale, adapted for text markers in `needcog.go`.
