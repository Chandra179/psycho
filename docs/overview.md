---
title: "Psycho"
description: "Psycho estimates psychological traits from text and shows the evidence behind each score."
seoTitle: "Psycho: Personality Profiling from Text"
seoDescription: "Psycho estimates Big Five traits, motivations, and values from writing using a dictionary-based method."
answerSummary: "Psycho estimates psychological traits from text and shows the evidence behind each score."
tags: [system-design, nlp, psycholinguistics]
links:
  github: "https://github.com/Chandra179/psycho"
created: 2026-09-29
---

# Psycho: Personality Profiling from Text

Psycho estimates Big Five traits, motivational and cognitive tendencies, and
value-related language from writing. Reports show one estimated text score per
measure, an overall reading-quality flag, and the evidence behind each score.

It is useful for:

- reflecting on your writing;
- comparing style across blogs, emails, or chats;
- studying language and personality; and
- reviewing your own text and the evidence behind its scores.

Psycho runs as one process, stores data in an embedded database, requires no
account, and makes no external calls during analysis. It is not a clinical or
diagnostic tool and does not assess mental health.

## How it works

```text
your writing → clean and organize → read word by word → infer tendencies → your profile
                                            ↑                 │
                              a psycholinguistic        the evidence behind
                                  dictionary            every score
```

Reports include scores, evidence, and a short summary. Expand calculation
details for approximate percentiles, unvalidated bounds, and recorded formulas.
View reports online or export them as PDFs.

### 1. Add your writing

Submit text such as blog posts, emails, or chat logs. Psycho checks its length,
normalizes it, and preserves paragraph and sentence structure.

### 2. Analyze the text

Psycho maps words to dictionary categories, counts them, measures style, and
reports dictionary coverage, the share of the sample represented.

### 3. Read your profile

Category counts produce one estimated text score per measure. The responsive
**Text-based measures** section lists all nine, with the Big Five first and four
project-defined language proxies after them. A shared legend explains Big Five
bands; each proxy has its own explanation. Explanatory text fills the report
width and wraps on narrow screens. Expand calculation details for approximate
percentiles, unvalidated bounds, and recorded formulas.

Value-related language shows each category's count, share of all words, and up
to two sampled excerpts with matching words highlighted. Excerpts are limited
to 240 Unicode characters, stored as plain text, and escaped when rendered.
Older results show available word samples and flag missing excerpts or counts.
Mentions and rejections both count toward percentages.

## Main features

- **Local analysis**: runs in one process, stores data in an embedded database, and needs no account or external calls.
- **Dictionary-based scores**: repeatable with the same model, dictionary, and calibration; reports show matching word categories.
- **Big Five (OCEAN)**: openness, conscientiousness, extraversion, agreeableness, and neuroticism, estimated from word patterns.
- **Regulatory Focus**: promotion- versus prevention-related language.
- **Need for Cognition**: analytic and intuitive word patterns used as a proxy for effortful thinking.
- **Need for Closure**: certainty and tentative language used as a proxy for ambiguity tolerance.
- **Cognitive style**: systematic and intuitive processing markers.
- **Schwartz values**: value categories in the writing, based on a cross-cultural framework.
- **Rough score bounds**: diagnostics based on text length and dictionary coverage, not validated confidence intervals or probabilities.
- **Quality flags**: short samples and low dictionary coverage can trigger warnings and widen bounds.
- **Reports**: view results online or export them as a PDF.
- **Saved analyses**: view or export past results.

## Algorithms

### Text normalization

Formatting is removed, and paragraph structure is preserved. Scoring counts
normalized words by category. Sentence and paragraph boundaries select
excerpts; the scorer does not interpret sentence meaning, word order, sarcasm,
or negation in context.

### Dictionary mapping

Each word is matched against a LIWC-style dictionary. Category counts become
percentages of the full sample for features such as emotion, cognition, and
social language. Coverage is the share of words matched by at least one
category. Features also include lexical diversity and word length.

### Big Five correlation-weighted heuristic

The Big Five scorer applies a correlation-weighted heuristic inspired by
language associations (Yarkoni, 2010). It scales correlations using assumed
trait and category standard deviations, not fitted regression coefficients.
Fixed category ordering makes results reproducible with the same model,
dictionary, and calibration. Other measures are project-defined proxies, not
official LIWC algorithms or validated personality tests.

### Regulatory Focus

Words associated with gains and aspirations are counted separately from words
associated with safety and duty (Higgins, 1997). Project-defined weights turn
these counts into a text proxy and label. Higgins (1997) supports the construct,
not this word-list scoring formula.

### Need for Cognition

The project weights analytic and intuitive word categories as a proxy for
effortful thinking. Cacioppo and Petty (1982) support the construct, not these
word lists or weights.

### Need for Closure

The project weights certainty language positively and tentative language
negatively as a text proxy. Webster and Kruglanski (1994) describe the construct,
not this scoring formula.

### Schwartz values

Value keywords use Schwartz's cross-cultural framework (1992), adapted to show
how value categories co-occur in text.

### Rough score ranges

Score bounds reflect text length and dictionary coverage: longer samples tend
to have narrower bounds, while low coverage widens them. Very short samples
receive a low-confidence flag. Samples with fewer than 10 normalized Unicode
characters or no letters or numbers are rejected. These bounds use project
assumptions; they are not validated confidence intervals. Percentiles are
approximate ranks among 3,992 retained blog texts, with ties assigned their
midpoint. They do not represent percentages of people. The main report shows
the 0–100 text score; calculation details include percentiles and bounds.

## Current evidence

Psycho is early stage. Benchmark speed comes from one development machine and
tracks regressions; it is not a production estimate. Other figures are design
targets.

| Area | Current status |
|---|---|
| Speed | a 5,000-word corpus analyzes in a median of **5 ms** (p95: **11 ms**) on the benchmark machine; the design target is **under 5 seconds** |
| Usage | designed for 1–10 analyses per minute, personal, single-user pacing |
| Storage | about **10 MB** per subject, including text, evidence, and profile |
| Short samples | samples below 500 words are flagged low-confidence; invalid text is rejected |
| Quality | pipeline tests check known profiles, feature counts, category mappings, and score directions; the dictionary matches about **58%** of words in typical test samples (2,155 words across 36 categories) |
| Measured accuracy | evaluated on **2,442 essays** from a local CSV of 2,467 rows with binary questionnaire labels; provenance is unverified. Trait-ranking AUC was **0.524–0.554** (0.5 is chance). This does not establish individual accuracy or show that a larger dictionary would improve results. |

The separate [offline supervised experiment](offline-supervised.md) evaluates learned models on held-out essay authors.

## Your data

Each analysis stores the submitted text, word counts, scores, evidence, and
summary in one database file. Analyses remain available after restarts; copy
the file to back them up.

## References

- Pennebaker, J.W., Boyd, R.L., Jordan, K., & Blackburn, K. (2015). [*The development and psychometric properties of LIWC2015*](https://www.liwc.net/). University of Texas at Austin. The dictionary model behind the word-to-category mapping, and the word-length summary variables stylized here.
- Tweedie, F.J., & Baayen, R.H. (1998). [*How variable may a constant be? Measures of lexical richness in perspective*](https://doi.org/10.1023/A:1001749303136). Computers and the Humanities, 32(5), 323-352. The type-token ratio and related lexical diversity measures.
- Yarkoni, T. (2010). [*Personality in 100,000 words: A large-scale analysis of personality and word use among bloggers*](https://doi.org/10.1016/j.jrp.2010.04.001). Journal of Research in Personality, 44(3), 363-373. Language-category associations inspiring the Big Five heuristic, not fitted weights.
- Pennebaker, J.W., & King, L.A. (1999). [*Linguistic styles: Language use as an individual difference*](https://doi.org/10.1037/0022-3514.77.6.1296). Journal of Personality and Social Psychology, 77(6), 1296-1312. The finding that function words carry stable personality signals.
- Higgins, E.T. (1997). [*Beyond pleasure and pain*](https://doi.org/10.1037/0003-066X.52.12.1280). American Psychologist, 52(12), 1280-1300. Regulatory Focus Theory, promotion versus prevention.
- Cacioppo, J.T., & Petty, R.E. (1982). [*The need for cognition*](https://doi.org/10.1037/0022-3514.42.1.116). Journal of Personality and Social Psychology, 42(1), 116-131. The need for cognition construct.
- Webster, D.M., & Kruglanski, A.W. (1994). [*Individual differences in need for cognitive closure*](https://doi.org/10.1037/0022-3514.67.6.1049). Journal of Personality and Social Psychology, 67(6), 1049-1062. The need for closure construct; the project's text mapping is a proxy.
- Schwartz, S.H. (1992). [*Universals in the content and structure of values: Theoretical advances and empirical tests in 20 countries*](https://doi.org/10.1016/S0065-2601(08)60281-6). Advances in Experimental Social Psychology, 25, 1-65. The Schwartz Value Survey behind the value orientations.
- Schler, J., Koppel, M., Argamon, S., & Pennebaker, J.W. (2006). *Effects of age and gender on blogging*. AAAI Spring Symposium on Computational Approaches to Analyzing Weblogs. The Blog Authorship Corpus used for percentile calibration.
