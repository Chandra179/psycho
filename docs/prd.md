# Psycho Product Requirements (PRD)

## Goal

A local-first system that extracts the psychological structure of a person from their writing and presents it with full auditability: every trait, cognitive label, and value assignment traceable to specific linguistic evidence, with explicit confidence levels. Zero data leaves the device.

## Non-goals

* Clinical diagnosis or mental health assessment
* Real‑time surveillance or monitoring
* Predicting future behavior
* Black‑box LLM inference (all claims are auditable)
* Multi‑modal input (audio, video); text only in this version
* Multi‑tenant SaaS platform; single‑user local app for now

## Numbers

* QPS: 1–10 analysis requests per minute (single‑user local app)
* Storage: \~10 MB per analyzed subject (raw text + feature vectors + profile)
* Latency target: <5 seconds for full analysis of a 5,000‑word corpus

## Constraints

* Only handle text input: direct paste, URL fetch, and directory import (.txt files from a local folder). Browser-style file upload is deferred to a later phase. No audio, video, or images.
* Single user. No authentication, no multi‑tenancy, no role‑based access.
* Only Big Five (OCEAN), Regulatory Focus (Higgins, 1997), Need for Cognition (Cacioppo & Petty, 1982), cognitive style, and Schwartz values. No MBTI, Enneagram, or custom frameworks in MVP.
* Dictionary‑based feature extraction only. LLM used optionally for narrative prose synthesis, never for core trait inference.
* Max 3 source types flagged per analysis (e.g., blog, chat, email). No unlimited source taxonomy.
* No real‑time collaboration or sharing. Export profile as JSON/PDF only.

***

## Core Features

### **Feature 1: Text Ingestion & Psychometric Analysis**

**What it does:** User submits text via direct paste, URL, or directory import. System normalises, extracts psycholinguistic features, and outputs Big Five trait scores, Regulatory Focus, Need for Cognition, cognitive style labels, and value orientations with confidence intervals.

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

## Core Feature Implementation Phase

**Phase 1: Text Ingestion & Basic Analysis**

* Build `ingest` module: paste handler, URL fetch, directory import. Normalise text, extract metadata.
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
