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
* Only Big Five (OCEAN), cognitive style (abstract/concrete, systematic/intuitive, need for closure), and Schwartz values. No MBTI, Enneagram, or custom frameworks in MVP.
* Dictionary‑based feature extraction only. LLM used optionally for narrative prose synthesis, never for core trait inference.
* Max 3 source types flagged per analysis (e.g., blog, chat, email). No unlimited source taxonomy.
* No real‑time collaboration or sharing. Export profile as JSON/PDF only.

***

## Core Features

### **Feature 1: Text Ingestion & Psychometric Analysis**

**What it does:** User submits text via paste, file upload, or URL. System normalises, extracts psycholinguistic features, and outputs Big Five trait scores, cognitive style labels, and value orientations with confidence intervals.

**Risks we tolerate:**

* No authentication on the ingestion endpoint. Anyone with access to the local port can submit text.
* Analysis may be unreliable for texts <500 words. System warns but does not block submission.
* Single‑threaded processing. Texts >50,000 words may take >30 seconds. No progress indicator in MVP.

**Trusted sources:**

* LIWC2015 dictionary (Pennebaker et al., 2015) – validated mapping of words to psychological categories.
* Big Five language correlates (Yarkoni, 2010; Pennebaker & King, 1999) – regression coefficients linking LIWC features to personality.
* Schwartz Value Survey (Schwartz, 1992) – framework for value orientation, adapted for text co‑occurrence.

### **Feature 2: Audit Trail**

**What it does:** Every trait score, cognitive label, and value assignment is clickable. The user can trace any output back to the specific word‑frequency percentages, dictionary categories, and statistical features that produced it.

**Risks we tolerate:**

* Audit links may break if the underlying dictionary version changes between analyses. MVP ships with a single dictionary version.
* Audit trail data stored alongside profile; for very large corpora this may double storage per subject.
* No diff view between audit trails of different subjects in MVP.

**Trusted sources:**

* Explainable AI design principles from the DARPA XAI program (Gunning et al., 2019) – ensuring traceability of model decisions.

### **Feature 3: Temporal Comparison**

**What it does:** If the user provides writing from two or more distinct time periods, the system detects statistically significant shifts in traits, cognitive style, and emotional tone. Outputs a timeline of psychological change.

**Risks we tolerate:**

* Statistical significance requires ≥1,000 words per time period. Below that, shifts are flagged as "low confidence" but still displayed.
* Comparing across different source types (e.g., work email vs. personal blog) may produce misleading shifts. System warns but does not enforce source-type consistency.
* No automatic period detection. User must explicitly label time periods.

**Trusted sources:**

* Cohen's d effect size (Cohen, 1988) for detecting meaningful trait shifts between time points.
* Longitudinal language‑personality studies (Mehl et al., 2006) – baseline expectations for stability vs. change.

***

## Software Architecture

**Style:** Modular monolith — components share a single process and database but have clear interface boundaries. No network calls between modules.

**Core flow**

1. User submits text (paste, file, URL). The ingest module normalises whitespace, strips irrelevant markup, segments into sentences and paragraphs, and attaches source metadata (type, date).
2. The normalised text passes to the analyze module, which tokenises and compares against a psycholinguistic dictionary. It computes category percentages, stylometric features, and a coverage rate.
3. The feature vector is fed to trait inference (Big Five regression), cognitive style classification, and value orientation mapping. Every output is stored with the evidence chain that produced it.
4. The profile module aggregates all scores, attaches confidence intervals, and generates structured output. Optionally, an external LLM call (user‑configurable, off by default) synthesises a narrative portrait from the structured scores.
5. If multiple time‑labelled corpora exist for the same subject, the temporal module compares baseline profiles across periods and outputs a change timeline.

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
  temporal/                #   temporal comparison module
    config.go
    dependencies.go
    comparator.go          #     cross-period shift detection
middleware/                # shared: recovery, request ID, timeout, validation
config/                    # YAML loader + config.yaml
```

### **Module boundaries**

* **ingest** — Owns text normalisation, segmentation, and source metadata. Exposes a clean document object to downstream modules. Does NOT know about dictionaries, traits, or profiles.
* **analyze** — Owns the psycholinguistic dictionary, feature extraction, and trait inference models. Depends on ingest for clean text. Does NOT know about temporal comparison or narrative synthesis.
* **profile** — Owns score aggregation, confidence computation, and narrative generation. Depends on analyze for trait/feature data. Does NOT know about ingestion or temporal logic.
* **temporal** — Owns cross‑period comparison and shift detection. Depends on profile for baseline profiles. Does NOT know about raw text or dictionary internals.

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

**temporal**

* No interfaces. Single implementation. Shift detection uses a fixed statistical method (Cohen's d on trait scores). If the method changes, it changes for everyone.

***

## Core Feature Implementation Phase

**Phase 1: Text Ingestion & Basic Analysis**

* Build `ingest` module: paste handler, file upload, URL fetch. Normalise text, extract metadata.
* Build `analyze` module: load dictionary, tokenise, compute category percentages and stylometrics.
* Implement Big Five inference using published regression coefficients (hardcoded for MVP).
* No cognitive style or values yet. No audit trail. No narrative synthesis.
* Write unit tests for normalizer, dictionary lookup, and trait inference.
* Write integration test: paste 1,000‑word sample → receive Big Five scores with confidence intervals.

**Checkpoint:** User pastes text. System returns raw Big Five scores with confidence intervals. No UI beyond JSON output.

**Phase 2: Audit Trail & Profile Synthesis**

* Add evidence chain to every trait score: store the word‑frequency percentages and dictionary categories that produced it.
* Build `profile` module: aggregate scores, compute confidence intervals, generate structured output.
* Implement `NarrativeGenerator` with template‑based (no LLM) implementation. Flag where LLM integration point exists.
* Add cognitive style and value orientation inference to `analyze` module.
* Integration test: click on any trait score → see the linguistic evidence. Change dictionary → audit link warns of version mismatch.

**Checkpoint:** Full structured profile output. Every claim auditable. No LLM dependency. JSON + basic HTML report.

**Phase 3: Temporal Comparison**

* Build `temporal` module: accept multiple time‑labelled corpora for same subject.
* Compute baseline profiles per period. Detect shifts using Cohen's d on Big Five scores.
* Flag low‑confidence shifts (insufficient words per period).
* Integration test: two time‑separated text sets → detect significant shifts, annotate low‑confidence ones.

**Checkpoint:** User uploads writing from 2023 and 2024. System returns a change timeline showing what shifted with confidence levels.

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
* "Two profiles with Cohen's d < 0.2 on Extraversion are flagged as 'no significant shift'."

**Integration Tests**

**What:** Module interactions. Full pipeline from text input to profile output. Temporal comparison with real data.

**Examples:**

* "Submit 5,000‑word personal blog corpus → receive Big Five, cognitive style, and values within 5 seconds. All scores have evidence links."
* "Audit link for Openness score: click it → receive list of cognitive process word percentages, tentative word count, and type‑token ratio that contributed to the score."
* "Submit two time‑labelled corpora (2023 and 2024) from the same subject → temporal module detects significant Openness increase and flags low‑confidence Agreeableness shift."
* "Submit text with 80% domain‑specific jargon → system returns low dictionary coverage warning and wide confidence intervals."
* "Change dictionary version → existing profiles warn of version mismatch on audit links."
* Use test fixtures: pre‑prepared text samples with known linguistic profiles, embedded SQLite for test isolation.

***

## References

These are the published works, validated tools, and proven implementations that underpin the system. Every core inference is traceable to one of these sources.

#### Psycholinguistic Dictionary & Validation

* Pennebaker, J.W., Boyd, R.L., Jordan, K., & Blackburn, K. (2015). *The development and psychometric properties of LIWC2015*. University of Texas at Austin. – The standard dictionary for mapping words to psychological categories. Provides the category‑trait validation used in the `analyze` module.
* Fast, E., Chen, B., & Bernstein, M.S. (2016). *Empath: Understanding Topic Signals in Large‑Scale Text*. CHI 2016. – Open‑source alternative to LIWC with 194 categories, built on modern word embeddings. Used as the default dictionary if LIWC licence is unavailable.

#### Big Five & Language

* Yarkoni, T. (2010). *Personality in 100,000 words: A large‑scale analysis of personality and word use among bloggers*. Journal of Research in Personality. – Provides the regression coefficients linking LIWC features to Big Five traits. Directly implemented in `inference.go`.
* Pennebaker, J.W., & King, L.A. (1999). *Linguistic styles: Language use as an individual difference*. Journal of Personality and Social Psychology. – Foundational work establishing that function words (pronouns, articles) carry reliable personality signals.

#### Value Frameworks

* Schwartz, S.H. (1992). *Universals in the content and structure of values: Theoretical advances and empirical tests in 20 countries*. Advances in Experimental Social Psychology. – The Schwartz Value Survey, adapted for keyword co‑occurrence in `analyze/values.go`.

#### Cognitive Style

* Petty, R.E., & Cacioppo, J.T. (1986). *The Elaboration Likelihood Model of persuasion*. Advances in Experimental Social Psychology. – Basis for systematic vs. intuitive processing markers.
* Webster, D.M., & Kruglanski, A.W. (1994). *Individual differences in need for cognitive closure*. Journal of Personality and Social Psychology. – Need for closure operationalised via certainty/tentative word ratios.

#### Temporal Comparison & Effect Sizes

* Cohen, J. (1988). *Statistical Power Analysis for the Behavioral Sciences* (2nd ed.). – Cohen's d used for shift detection in `temporal/comparator.go`.
* Mehl, M.R., Gosling, S.D., & Pennebaker, J.W. (2006). *Personality in its natural habitat: Manifestations and implicit folk theories of personality in daily life*. Journal of Personality and Social Psychology. – Longitudinal stability expectations for language‑based personality measures.

#### Explainability

* Gunning, D., Stefik, M., Choi, J., Miller, T., Stumpf, S., & Yang, G.Z. (2019). *XAI—Explainable artificial intelligence*. Science Robotics. – Design principles for the audit trail and evidence‑chain architecture.


---

# Agent Instructions: Querying This Documentation

If you need additional information that is not directly available in this page, you can query the documentation dynamically by asking a question.

Perform an HTTP GET request on the current page URL with the `ask` query parameter:

```
GET https://nothin.gitbook.io/computing/psyhco.md?ask=<question>
```

The question should be specific, self-contained, and written in natural language.
The response will contain a direct answer to the question and relevant excerpts and sources from the documentation.

Use this mechanism when the answer is not explicitly present in the current page, you need clarification or additional context, or you want to retrieve related documentation sections.