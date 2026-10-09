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
updated: 2026-10-09
---

# Psycho: Personality Profiling from Text

Psycho estimates Big Five traits and related language measures from text and shows the evidence behind each score.

It runs as one process, needs no account, and makes no external calls during analysis. It is not a clinical or diagnostic tool.

## Algorithms and approach

```text
Writing -> Normalize -> Dictionary match -> Score (9 measures) -> Rank vs 2004 blog sample -> Report
                              |                                                                |
                    word categories + coverage                       score cards, evidence, fit notes
```

1. **Normalize:** formatting is removed; paragraph and sentence structure is kept.
2. **Map:** each word is matched against a LIWC-style dictionary. Words mostly used in another sense ("just", "will", "so") are left out of the scored lists. Coverage is the share of words scored.
3. **Score:** the Big Five use a correlation-weighted heuristic (Yarkoni, 2010). Regulatory Focus, Need for Cognition, Need for Closure and Cognitive Style are project-defined proxies. Schwartz values are category counts with excerpts.
4. **Rank:** each score is ranked against 3,992 blog texts from 2004, with ties at the midpoint. This is not a percentage of people.
5. **Check quality:** under 500 words or under 45% coverage is low; under 1,000 words or under 60% coverage is medium. Fit notes flag measures that mostly reflect style or topic.
6. **Report:** nine score cards, value excerpts, expandable calculation details, and PDF export.

## Evidence

| Area | Result |
|---|---|
| Speed | 5,000 words in a median of 5 ms (p95 11 ms), one development machine |
| Storage | about 10 MB per subject |
| Coverage | about 61% of words scored in typical test samples |
| Accuracy | trait-ranking AUC 0.532 to 0.559 on 2,442 essays (0.5 is chance) |

**How to read this:**

- **Good:** analysis is fast and repeatable with the same model, dictionary and calibration.
- **Weak:** AUC is only slightly above chance, and the score bounds describe repeatability between halves of a text, not accuracy about a person.
- **Limit:** the 2004 blog sample is a poor reference for essays or formal prose. Sarcasm and word order are not read, and negation is handled only for value words and emotional tone, and ambiguous words are removed from trait lists, not read in context. Do not use scores for hiring, clinical or other decisions about a person. See the [offline supervised experiment](offline-supervised.md).

## Your data

Each analysis stores its text, scores and evidence in one database file. Copy the file to back it up.

## References

- Pennebaker, J.W., Boyd, R.L., Jordan, K., & Blackburn, K. (2015). [*The development and psychometric properties of LIWC2015*](https://www.liwc.net/).
- Tweedie, F.J., & Baayen, R.H. (1998). [*How variable may a constant be? Measures of lexical richness in perspective*](https://doi.org/10.1023/A:1001749303136).
- Yarkoni, T. (2010). [*Personality in 100,000 words*](https://doi.org/10.1016/j.jrp.2010.04.001).
- Pennebaker, J.W., & King, L.A. (1999). [*Linguistic styles: Language use as an individual difference*](https://doi.org/10.1037/0022-3514.77.6.1296).
- Higgins, E.T. (1997). [*Beyond pleasure and pain*](https://doi.org/10.1037/0003-066X.52.12.1280).
- Cacioppo, J.T., & Petty, R.E. (1982). [*The need for cognition*](https://doi.org/10.1037/0022-3514.42.1.116).
- Webster, D.M., & Kruglanski, A.W. (1994). [*Individual differences in need for cognitive closure*](https://doi.org/10.1037/0022-3514.67.6.1049).
- Schwartz, H.A., et al. (2013). [*Choosing the right words: Characterizing and reducing error of the word count approach*](https://aclanthology.org/S13-1042/). Motivation for the word-sense audit.
- Schwartz, S.H. (1992). [*Universals in the content and structure of values*](https://doi.org/10.1016/S0065-2601(08)60281-6).
- Schler, J., Koppel, M., Argamon, S., & Pennebaker, J.W. (2006). *Effects of age and gender on blogging*. The Blog Authorship Corpus used for calibration.
