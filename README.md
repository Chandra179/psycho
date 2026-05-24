# Psyhco

## Goal

A local-first system that extracts the psychological structure of a person from their writing and presents it with full auditability—every trait, cognitive label, and value assignment traceable to specific linguistic evidence, with explicit confidence levels. Zero data leaves the device.

## Non-goals

* Clinical diagnosis or mental health assessment
* Real‑time surveillance or monitoring
* Predicting future behavior
* Black‑box LLM inference (all claims are auditable)
* Multi‑modal input (audio, video); text only in this version
* Multi‑tenant SaaS platform; single‑user local app for now

## Numbers

* QPS: 1–10 analysis requests per minute (single‑user local app)
* Storage: \~10 MB per analyzed subject (raw text + feature vectors + profile)
* Latency target: <5 seconds for full analysis of a 5,000‑word corpus

## Constraints

* Only handle text input: file upload, URL fetch, direct paste. No audio, video, or images.
* Single user. No authentication, no multi‑tenancy, no role‑based access.
* Only Big Five (OCEAN), Regulatory Focus (Higgins, 1997), Need for Cognition (Cacioppo & Petty, 1982), cognitive style, and Schwartz values. No MBTI, Enneagram, or custom frameworks in MVP.
* Dictionary‑based feature extraction only. LLM used optionally for narrative prose synthesis, never for core trait inference.
* Max 3 source types flagged per analysis (e.g., blog, chat, email). No unlimited source taxonomy.
* No real‑time collaboration or sharing. Export profile as JSON/PDF only.

***

## Core Features

### **Feature 1: Text Ingestion & Psychometric Analysis**

**What it does:** User submits text via paste, file upload, or URL. System normalises, extracts psycholinguistic features, and outputs Big Five trait scores, Regulatory Focus, Need for Cognition, cognitive style labels, and value orientations with confidence intervals.

**Risks we tolerate:**

* No authentication on the ingestion endpoint. Anyone with access to the local port can submit text.
* Analysis may be unreliable for texts <500 words. System warns but does not block submission.
* Single‑threaded processing. Texts >50,000 words may take >30 seconds. No progress indicator in MVP.

**Trusted sources:**

* LIWC2015 dictionary (Pennebaker et al., 2015) – validated mapping of words to psychological categories.
* Big Five language correlates (Yarkoni, 2010; Pennebaker & King, 1999) – Spearman correlations linking LIWC categories to personality traits, implemented in `coefficients.go`.
* Regulatory Focus (Higgins, 1997) – promotion/prevention word markers in `regfocus.go`.
* Need for Cognition (Cacioppo & Petty, 1982) – analytic/intuitive word markers in `needcog.go`.
* Schwartz Value Survey (Schwartz, 1992) – framework for value orientation, adapted for text co‑occurrence.

***

## Software Architecture

**Style:** Modular monolith — components share a single process and database but have clear interface boundaries. No network calls between modules.

**Core flow**

1. User submits text (paste, file, URL). The ingest module normalises whitespace, strips irrelevant markup, segments into sentences and paragraphs, and attaches source metadata (type, date).
2. The normalised text passes to the analyze module, which tokenises and compares against a psycholinguistic dictionary. It computes category percentages, stylometric features, and a coverage rate.
3. The feature vector is fed to trait inference (Big Five regression), Regulatory Focus, Need for Cognition, cognitive style classification, and value orientation mapping. Every output is stored with the feature evidence that produced it.
4. The profile module aggregates all scores, attaches confidence intervals, and generates structured output. Optionally, an external LLM call (user‑configurable, off by default) synthesises a narrative portrait from the structured scores.

### **Storage choice & why**

**Embedded SQLite** — Single‑user local app with modest data volumes. No server process needed. Provides queryability for cross‑subject comparison and temporal tracking that flat JSON files would make cumbersome. The database file is portable; a user can back up their entire analysis history by copying one file.

### **Directory Structure**

```
cmd/psycho/main.go      # entrypoint — starts HTTP server
modules/
  ingest/                  #   text ingestion module
    config.go              #     module-specific config struct
    dependencies.go        #     wire deps, load own config
    http.go                #     HTTP handlers + route registration
    normalizer.go          #     text normalization logic
  analyze/                 #   psycholinguistic analysis module
    config.go
    dependencies.go
    dictionary.go          #     dictionary lookup engine
    features.go            #     stylometric feature extraction
    inference.go           #     Big Five + cognitive style inference
  profile/                 #   profile generation module
    config.go
    dependencies.go
    synthesizer.go         #     aggregate scores, confidence intervals
    narrative.go           #     optional LLM narrative synthesis
middleware/                # shared: recovery, request ID, timeout, validation
config/                    # YAML loader + config.yaml
```

### **Module boundaries**

* **ingest** — Owns text normalisation, segmentation, and source metadata. Exposes a clean document object to downstream modules. Does NOT know about dictionaries, traits, or profiles.
* **analyze** — Owns the psycholinguistic dictionary, feature extraction, and trait inference models. Depends on ingest for clean text. Does NOT know about temporal comparison or narrative synthesis.
* **profile** — Owns score aggregation, confidence computation, and narrative generation. Depends on analyze for trait/feature data. Does NOT know about ingestion logic.

### **Dependencies**

* **Go standard library:** `net/http`, `database/sql`, `encoding/json`, `text/template`
* **Open source:** `go-sqlite3` (embedded database), `empath` or equivalent open‑source psycholinguistic lexicon, optional LLM client package (Gemini/OpenAI, user‑configured)
* **Sidecar/optional:** A small LLM binary (e.g., Ollama) running locally if the user enables narrative synthesis. The app functions fully without it.

### **Abstraction Depth per Module**

**ingest** — No interfaces. Single implementation. Text normalisation is not swappable; the rules are the product.

**analyze**

* `Dictionary` interface — **Why abstracted:** Allows swapping between LIWC‑compatible lexicons without changing inference logic. Users may bring their own dictionary. The module exports `Lookup(word) → []Category` as the contract.
* `TraitModel` interface — **Why abstracted:** The regression model may be updated as new research publishes. The module exports `Infer(features) → BigFiveScores`.
* `FeatureExtractor` is NOT abstracted — single implementation. The features are dictated by the psycholinguistic literature, not user preference.

**profile**

* `NarrativeGenerator` interface — **Why abstracted:** Users may choose no LLM (template‑based), a local LLM (Ollama), or a cloud API (Gemini). The module exports `GenerateSynthesis(scores) → string`.
* `ScoreAggregator` is NOT abstracted — single implementation. The aggregation math is the product.

***

## Core Feature Implementation Phase

**Phase 1: Text Ingestion & Basic Analysis**

* Build `ingest` module: paste handler, file upload, URL fetch. Normalise text, extract metadata.
* Build `analyze` module: load dictionary, tokenise, compute category percentages and stylometrics.
* Implement Big Five inference using published regression coefficients (hardcoded for MVP).
* Write unit tests for normalizer, dictionary lookup, and trait inference.
* Write integration test: paste 1,000‑word sample → receive Big Five scores with confidence intervals.

**Checkpoint:** User pastes text. System returns Big Five scores with confidence intervals. No UI beyond JSON output.

**Phase 2: Extended Dimensions & Profile Synthesis**

* Add Regulatory Focus (Higgins, 1997) inference: promotion/prevention word markers, output score + label.
* Add Need for Cognition (Cacioppo & Petty, 1982) inference: analytic/intuitive word markers, output score + label.
* Build `profile` module: aggregate all scores, compute confidence intervals, generate structured output.
* Implement `NarrativeGenerator` with template‑based (no LLM) implementation.
* Add cognitive style and value orientation inference when word-lists are compiled.
* Unit tests for each new inference model + updated integration test for 7 dimensions.

**Checkpoint:** System returns Big Five + Regulatory Focus + Need for Cognition with confidence intervals. JSON output.

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

* "Submit 5,000‑word personal blog corpus → receive Big Five, Regulatory Focus, and Need for Cognition within 5 seconds. All 7 dimensions have valid scores and confidence intervals."
* "Submit text with 80% domain‑specific jargon → system returns low dictionary coverage warning and wide confidence intervals."
* Use test fixtures: pre‑prepared text samples with known linguistic profiles, embedded SQLite for test isolation.

***

## References

These are the published works, validated tools, and proven implementations that underpin the system. Every core inference is traceable to one of these sources.

#### Psycholinguistic Dictionary & Validation

* Pennebaker, J.W., Boyd, R.L., Jordan, K., & Blackburn, K. (2015). *The development and psychometric properties of LIWC2015*. University of Texas at Austin. – The standard dictionary for mapping words to psychological categories. Provides the category‑trait validation used in the `analyze` module.
* Fast, E., Chen, B., & Bernstein, M.S. (2016). *Empath: Understanding Topic Signals in Large‑Scale Text*. CHI 2016. – Open‑source alternative to LIWC with 194 categories, built on modern word embeddings. Used as the default dictionary if LIWC licence is unavailable.

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