---
title: "Psycho"
description: "Psycho is a personality profiler that extracts auditable psychological traits from text."
seoTitle: "Psycho: Auditable Personality Profiling from Text"
seoDescription: "Psycho is a personality profiler that extracts Big Five traits, motivations, and values from writing with a transparent, fully auditable method."
answerSummary: "Psycho is a personality profiler that extracts auditable psychological traits from text."
tags: [system-design, nlp, psycholinguistics]
links:
  github: "https://github.com/Chandra179/psycho"
created: 2026-09-29
---

# Psycho: Personality Profiling with Full Auditability

Psycho is an application that reads a sample of someone's writing
and describes the psychological structure behind it: Big Five trait scores,
motivational and cognitive tendencies, and value orientations. Every score
comes with an honest confidence level and the linguistic evidence that
produced it, so nothing is a verdict handed down by a black box.

It is useful for:

- journaling and self-reflection over your own writing;
- seeing how style and emphasis shift across a corpus of blogs, emails, or chat logs;
- research on the relationship between language use and personality;
- writing-sample analysis where the reasoning behind a score matters; and
- any text a person owns and wants to understand through a psycholinguistic lens.

Everything runs in one small application: one process, one embedded database,
no accounts, and no external calls in the analysis pipeline. Psycho is
not a clinical instrument; diagnosis and mental-health assessment are explicit
non-goals.

## How it works

```text
your writing → clean and organize → read word by word → infer tendencies → your profile
                                            ↑                 │
                              a psycholinguistic        the evidence behind
                                  dictionary            every score
```

You give Psycho text, and it gives you back a written profile: scores with
confidence intervals, the evidence behind each one, and a short narrative
summary. The profile can be read on screen or exported as a PDF report.

### 1. Add your writing

You provide a sample of writing, for example saved blog posts, emails, or
chat logs. Psycho checks that the sample is long enough to say anything
meaningful, cleans it up, and preserves its structure: paragraphs, sentences,
and rhythm all survive intact.

### 2. Psycho reads it like a linguist

Every word is looked up in a psycholinguistic dictionary that maps words to
psychological categories. From those lookups Psycho counts how often each
category appears, measures stylistic habits, and records how much of the text
the dictionary actually covered, the main signal for how well the tool fits
the sample.

### 3. Read your profile

Each dimension is inferred from those counts and combined into a profile with
confidence intervals and short narrative prose. The profile is saved, so you can come back to it, and the evidence is kept
alongside every score so any result can be traced back to the words that
produced it.

## Main features

- **Private by design**: no accounts, and the analysis pipeline makes no external calls.
- **No black-box AI**: scores come from a transparent, dictionary-based method, so the same text always produces the same result and every score can be explained.
- **Big Five (OCEAN) traits**: openness, conscientiousness, extraversion, agreeableness, and neuroticism, estimated from word-use patterns.
- **Regulatory Focus**: whether the writing leans toward promotion (gains, aspirations) or prevention (safety, obligations).
- **Need for Cognition**: the writer's tendency toward effortful, analytic thinking.
- **Need for Closure**: comfort with definite answers versus ambiguity.
- **Cognitive style**: systematic versus intuitive processing markers.
- **Schwartz value orientations**: which values the writing emphasizes, from a cross-cultural framework.
- **Confidence intervals on every score**: the profile always states how much it should be trusted.
- **Full auditability**: every output is kept with the linguistic evidence that produced it.
- **Honest warnings**: short samples or poorly covered text widen the intervals and raise flags instead of failing silently.
- **Reports you can keep**: a PDF report to save or print, plus the single on-screen report.
- **Your whole history in one place**: past analyses persist and can be re-read or re-exported anytime.

## Algorithms

### Text normalization

Formatting is stripped while paragraph structure is preserved, and the text is
segmented into sentences and paragraphs. Downstream stages work on a clean
document with reliable boundaries, so feature counts are not distorted by
formatting artifacts.

### Dictionary mapping

Each word is looked up in a LIWC-style psycholinguistic dictionary, and the
categories it belongs to are tallied into percentages over the whole corpus:
how much emotion language, cognitive language, social language, and so on. The
share of words that hit any category at all is the coverage rate: the main
signal for how well the dictionary fits the text. Alongside the category
counts, simple stylometric measures (lexical diversity, word length) feed the
same feature vector.

### Big Five regression

The category percentages are weighted by published correlations between word
use and personality (Yarkoni, 2010; Pennebaker & King, 1999). Because the
weights come from large-scale studies rather than opinion, the same text
always produces the same, reproducible scores.

### Regulatory Focus

Words marking gains, aspirations, and advancement are counted separately from
words marking safety, duty, and loss avoidance (Higgins, 1997). The balance
between the two produces a promotion-versus-prevention score and label.

### Need for Cognition

Markers of analytic and intuitive processing (Cacioppo & Petty, 1982) are
tallied into a score for the writer's tendency toward effortful thinking.

### Need for Closure

The ratio of certainty words to tentative words (Webster & Kruglanski, 1994)
estimates the writer's preference for definite answers over ambiguity.

### Schwartz values

Value keywords are grouped by the Schwartz Value Survey framework (Schwartz,
1992), a cross-cultural model of human values, and adapted for how values
co-occur in text.

### Confidence intervals

Every score carries an interval that widens or narrows with the evidence:
longer samples produce tighter intervals, and text the dictionary barely
recognizes produces wider ones. Very short samples always receive a
low-confidence flag rather than being rejected. Scores are reported as
percentiles relative to a measured reference population, a sample of about
4,000 blog posts (the Blog Authorship Corpus), so "60th percentile" means
"higher than 60% of comparable texts in that reference sample."

## Current evidence

Psycho is early stage. The speed number below is measured by the automated
benchmark on a single development machine, a regression signal, not
production evidence, while the rest remain design targets.

| Area | Current status |
|---|---|
| Speed | a 5,000-word corpus analyzes in a median of **5 ms** (p95: **11 ms**) on the benchmark machine, far inside the **under 5 seconds** design target |
| Usage | designed for 1–10 analyses per minute, personal, single-user pacing |
| Storage | about **10 MB** per analyzed subject, including the text, the evidence, and the profile |
| Short samples | below 500 words results are flagged low-confidence, never blocked |
| Quality | every stage is covered by an automated test suite, including text fixtures with known linguistic profiles that pin exact feature counts, word-to-category placements, and the direction of every dimension; the dictionary recognizes about **58%** of words in typical test samples (2,155 words across 36 categories) |
| Measured accuracy | scored against a public corpus of **2,442 essays** with ground-truth personality ratings, all five Big Five dimensions rank people **above chance** (AUC 0.52–0.56, each confidence interval excluding coin-flip); the right direction everywhere, with honest, modest effect sizes that dictionary growth is expected to improve |

An automated validation suite runs these known-profile text samples through
the full pipeline on every test run, so a change that flips a score's
direction or distorts a word count fails loudly; latency percentiles are
recorded alongside each run.

## Your data

Each analysis is a permanent record: the text you submitted, the word counts,
the scores, the evidence behind them, and the narrative, all kept together in
one database file. Past analyses remain available across restarts, and your
entire history can be backed up by copying a single file. The analysis itself
calls no external service.

## References

- Pennebaker, J.W., Boyd, R.L., Jordan, K., & Blackburn, K. (2015). [*The development and psychometric properties of LIWC2015*](https://www.liwc.net/). University of Texas at Austin. The dictionary model behind the word-to-category mapping, and the word-length summary variables stylized here.
- Tweedie, F.J., & Baayen, R.H. (1998). [*How variable may a constant be? Measures of lexical richness in perspective*](https://doi.org/10.1023/A:1001749303136). Computers and the Humanities, 32(5), 323-352. The type-token ratio and related lexical diversity measures.
- Yarkoni, T. (2010). [*Personality in 100,000 words: A large-scale analysis of personality and word use among bloggers*](https://doi.org/10.1016/j.jrp.2010.04.001). Journal of Research in Personality, 44(3), 363-373. The word-category to trait weights for the Big Five.
- Pennebaker, J.W., & King, L.A. (1999). [*Linguistic styles: Language use as an individual difference*](https://doi.org/10.1037/0022-3514.77.6.1296). Journal of Personality and Social Psychology, 77(6), 1296-1312. The finding that function words carry stable personality signals.
- Higgins, E.T. (1997). [*Beyond pleasure and pain*](https://doi.org/10.1037/0003-066X.52.12.1280). American Psychologist, 52(12), 1280-1300. Regulatory Focus Theory, promotion versus prevention.
- Cacioppo, J.T., & Petty, R.E. (1982). [*The need for cognition*](https://doi.org/10.1037/0022-3514.42.1.116). Journal of Personality and Social Psychology, 42(1), 116-131. The need for cognition construct.
- Webster, D.M., & Kruglanski, A.W. (1994). [*Individual differences in need for cognitive closure*](https://doi.org/10.1037/0022-3514.67.6.1049). Journal of Personality and Social Psychology, 67(6), 1049-1062. Need for closure, read from certainty versus tentative language.
- Schwartz, S.H. (1992). [*Universals in the content and structure of values: Theoretical advances and empirical tests in 20 countries*](https://doi.org/10.1016/S0065-2601(08)60281-6). Advances in Experimental Social Psychology, 25, 1-65. The Schwartz Value Survey behind the value orientations.
- Schler, J., Koppel, M., Argamon, S., & Pennebaker, J.W. (2006). *Effects of age and gender on blogging*. AAAI Spring Symposium on Computational Approaches to Analyzing Weblogs. The Blog Authorship Corpus used for percentile calibration.
