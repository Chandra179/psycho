---
title: "Psycho"
description: "Psycho is a local-first personality profiler that extracts auditable psychological traits from text."
seoTitle: "Psycho: Local-First, Auditable Personality Profiling from Text"
seoDescription: "Psycho is a local-first personality profiler that extracts Big Five traits, motivations, and values from writing with a transparent, fully auditable method."
answerSummary: "Psycho is a local-first personality profiler that extracts auditable psychological traits from text."
tags: [system-design, nlp, psycholinguistics]
links:
  github: "https://github.com/Chandra179/psycho"
created: 2026-09-29
---

# Psycho: Local-First Personality Profiling with Full Auditability

Psycho is a local-first application that reads a sample of someone's writing
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

Everything runs on your own machine. The analyzed text and the resulting
profiles never need to leave the device, and no account is required. Psycho is
not a clinical instrument — diagnosis and mental-health assessment are explicit
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

You provide a sample of writing — for example saved blog posts, emails, or
chat logs. Psycho checks that the sample is long enough to say anything
meaningful, cleans it up, and preserves its structure: paragraphs, sentences,
and rhythm all survive intact.

### 2. Psycho reads it like a linguist

Every word is looked up in a psycholinguistic dictionary that maps words to
psychological categories. From those lookups Psycho counts how often each
category appears, measures stylistic habits, and records how much of the text
the dictionary actually covered — the main signal for how well the tool fits
the sample.

### 3. Read your profile

Each dimension is inferred from those counts and combined into a profile with
confidence intervals and short narrative prose. The profile is saved on your
device, so you can come back to it, and the evidence is kept alongside every
score so any result can be traced back to the words that produced it.

## Main features

- **Private by design** — text, profiles, and evidence stay on your own machine.
- **No black-box AI** — scores come from a transparent, dictionary-based method, so the same text always produces the same result and every score can be explained.
- **Big Five (OCEAN) traits** — openness, conscientiousness, extraversion, agreeableness, and neuroticism, estimated from word-use patterns.
- **Regulatory Focus** — whether the writing leans toward promotion (gains, aspirations) or prevention (safety, obligations).
- **Need for Cognition** — the writer's tendency toward effortful, analytic thinking.
- **Need for Closure** — comfort with definite answers versus ambiguity.
- **Cognitive style** — systematic versus intuitive processing markers.
- **Schwartz value orientations** — which values the writing emphasizes, from a cross-cultural framework.
- **Confidence intervals on every score** — the profile always states how much it should be trusted.
- **Full auditability** — every output is kept with the linguistic evidence that produced it.
- **Honest warnings** — short samples or poorly covered text widen the intervals and raise flags instead of failing silently.
- **Reports you can keep** — a PDF report to save or print, plus on-screen views in three styles.
- **Your whole history in one place** — past analyses persist on your device and can be re-read or re-exported anytime.

## Algorithms

### Text normalization

Formatting is stripped while paragraph structure is preserved, and the text is
segmented into sentences and paragraphs. Downstream stages work on a clean
document with reliable boundaries, so feature counts are not distorted by
formatting artifacts.

### Dictionary mapping

Each word is looked up in a LIWC-style psycholinguistic dictionary, and the
categories it belongs to are tallied into percentages over the whole corpus —
how much emotion language, cognitive language, social language, and so on. The
share of words that hit any category at all is the coverage rate: the main
signal for how well the dictionary fits the text.

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
percentiles relative to a reference sample.

## Under the hood

Text flows through one pipeline from submission to a finished profile: clean
up, count, infer, combine, save. Each stage does one job and hands its result
to the next, which is what keeps results reproducible and the reasoning
inspectable. None of this changes how Psycho is used — it explains why it
behaves the way it does.

### From sample to profile

```text
        your writing sample
             │
             ▼
   too short or too long? ─── yes ──▶ you are told the limits,
             │                         nothing is analyzed
             │ no
             ▼
   clean up: remove formatting, keep paragraph
   and sentence structure, count the words
             │
             ▼
   look up every word in the psycholinguistic dictionary
             │
             ▼
   category counts, style habits, and how much of
   the text the dictionary covered
             │
             ▼
   infer each dimension from those counts
             │
             ▼
   combine everything into a profile with
   confidence intervals and narrative prose
             │
             ▼
   saved on your device ──▶ ready to read or export
```

Each dimension is inferred directly from the same set of word counts, and the
inference is kept separate from the counting. That separation is why a score
can always be opened up: the evidence trail from your words to the final
number is stored with the profile, not discarded after the fact.

### Keeping scores honest

Low-quality input degrades results gradually rather than silently. A short
sample yields wide intervals and a low-confidence flag; jargon-heavy text the
dictionary barely recognizes yields the same. Nothing is blocked — the profile
is still produced — but the output always states how much it should be
trusted, and the evidence behind each score is kept alongside it.

### Reports you can keep

```text
   a finished analysis
       │
       ├── export ──▶ a PDF report you can save or print
       └── view ───▶ on-screen pages in three styles:
                     general, technical, and balanced
```

Reports are rendered from the saved analysis rather than by re-reading your
text, so a report generated today matches one generated months later. The
saved profile — scores, evidence, and prose — is the single source of truth.

### Private and simple

```text
 ┌──────────────────────────────────────────────────┐
 │ one app, on your machine                         │
 │                                                  │
 │ accounts:      none — no sign-up, no login       │
 │ network:       nothing is uploaded; analysis     │
 │                works offline                     │
 │ resilience:    the app recovers from unexpected  │
 │                errors instead of crashing        │
 │ long samples:  handled one at a time, in order   │
 └──────────────────────────────────────────────────┘
```

There are deliberately no accounts, no sharing, and no cloud: the app is
designed for one person on one machine. Whatever you analyze stays where you
put it.

## Current evidence

Psycho is early stage. The speed number below is measured by the automated
benchmark on a single development machine — a regression signal, not
production evidence — while the rest remain design targets.

| Area | Current status |
|---|---|
| Speed | a 5,000-word corpus analyzes in a median of **5 ms** (p95: **11 ms**) on the benchmark machine — far inside the **under 5 seconds** design target |
| Usage | designed for 1–10 analyses per minute — personal, single-user pacing |
| Storage | about **10 MB** per analyzed subject, including the text, the evidence, and the profile |
| Short samples | below 500 words results are flagged low-confidence, never blocked |
| Quality | every stage is covered by an automated test suite, including text fixtures with known linguistic profiles that pin exact feature counts, word-to-category placements, and the direction of every dimension |

An automated validation suite runs these known-profile text samples through
the full pipeline on every test run, so a change that flips a score's
direction or distorts a word count fails loudly; latency percentiles are
recorded alongside each run.

## Your data

Each analysis is a permanent record: the text you submitted, the word counts,
the scores, the evidence behind them, and the narrative — all kept together on
your device. Past analyses remain available across restarts, and your entire
history lives in one place that can be backed up by copying a single item.
Nothing in the analysis path needs an internet connection.

## What Psycho is designed for

Psycho is designed for private, personal profiling on one machine. It
prioritizes results you can interrogate — every score traceable to the words
that produced it — local data control, and a simple operating model: give it
text, get an honest, explainable profile back.

The deeper material is there if you want it: the
[product requirements](prd.md) describe what Psycho aims to do and explicitly
what it does not, and the [system design](system-design.md) explains how the
pieces fit together.
