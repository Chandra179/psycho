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
updated: 2026-10-10
---

# Psycho: Personality Profiling from Text

Psycho estimates Big Five traits and related language measures from text and shows the evidence behind each score.

It runs entirely in your browser, needs no account and no server, and makes no network calls during analysis. It is not a clinical or diagnostic tool.

## Algorithms and approach

```text
Writing -> Normalize -> Dictionary match -> Score (9 measures) -> Rank vs 2004 blog sample -> Report
                              |                                                                |
                    word categories + coverage                       score cards, evidence, fit notes
```

1. **Normalize:** formatting is removed; paragraph and sentence structure is kept.
2. **Map:** each word is matched against a LIWC-style dictionary. Words mostly used in another sense ("just", "will", "so") are left out of the scored lists. Coverage is the share of words scored.
3. **Score:** the Big Five use a correlation-weighted heuristic (Yarkoni, 2010), checked against the paper's Table 1; the Openness weights are scaled down so its spread matches the other traits. Cognitive Style is a function-word index (Pennebaker et al., 2014). Regulatory Focus, Need for Cognition and Need for Closure are project-defined proxies. Schwartz values are category counts with excerpts.
4. **Rank:** each score is ranked against 3,992 blog texts from 2004, with ties at the midpoint. This is not a percentage of people.
5. **Check quality:** under 500 words or under 45% coverage is low; under 1,000 words or under 60% coverage is medium. A share of lines that look like page numbers, captions or headers caps the flag at medium. Fit notes flag measures that mostly reflect style or topic.
6. **Report:** a notice under the header says the scores are rough and not for hiring or clinical use. Nine score cards each show a range ("another stretch of similar text would likely score N to M") and say "too close to call" when it crosses a band. Value excerpts and expandable calculation details follow, and the report can be saved as a PDF (through the browser's print dialog) or as one HTML file.

## Evidence

| Area | Result |
|---|---|
| Speed | 5,000 words in a median of 5 ms (p95 11 ms), one development machine, native build |
| Download | about 2.2 MB compressed, once |
| Coverage | about 61% of words scored in typical test samples |
| Accuracy | trait-ranking AUC 0.532 to 0.559 on 2,442 essays (0.5 is chance) |

**How to read this:**

- **Good:** analysis is fast and repeatable with the same model, dictionary and calibration.
- **Weak:** AUC is only slightly above chance, and the score ranges describe repeatability between halves of a text, not accuracy about a person. Most Big Five scores read "moderate".
- **Limit:** the 2004 blog sample is a poor reference for essays or formal prose. Sarcasm and word order are not read, and negation is handled only for value words and emotional tone, and ambiguous words are removed from trait lists, not read in context. Do not use scores for hiring, clinical or other decisions about a person. See the [offline supervised experiment](offline-supervised.md).

## Measured accuracy

`cmd/evaluate` scores a labeled essay set with the production path and compares each Big Five score with the questionnaire answers. The set is the local Essays CSV (2,467 rows, 2,442 kept at 200 or more words, yes/no labels). It is associated with Pennebaker & King (1999), but its exact version and label cutoffs have no provenance.

| Trait | Spearman rho | AUC | AUC 95% interval |
|---|---|---|---|
| Neuroticism | 0.099 | 0.556 | 0.533 to 0.578 |
| Agreeableness | 0.103 | 0.559 | 0.535 to 0.583 |
| Extraversion | 0.077 | 0.541 | 0.518 to 0.561 |
| Openness | 0.058 | 0.532 | 0.511 to 0.556 |
| Conscientiousness | 0.056 | 0.532 | 0.510 to 0.554 |

The Openness interval is from before the 2026-10-10 rescale, which moved its AUC from 0.533 to 0.532. These numbers are modestly above chance. They do not validate any single word category, the scale factors or the score ranges, and they apply only to this dictionary, corpus and labeling.

What did not help, tested on the same essays: more dictionary words (coverage rose, AUC did not move), letter-based word length and print cleanup (AUC moved by at most 0.004), the word-sense audit (mean AUC 0.542 to 0.545, inside the intervals) and a stricter word-sense filter (reverted, no gain). A second essay study (Mairesse et al., 2007) uses the same kind of essays, so it cannot serve as independent support. Fitted weights (`cmd/train`) scored 0.57 to 0.63 on 489 held-out authors against 0.52 to 0.58 for the heuristic, but every 99% interval includes zero, so they are not shipped. See the [offline supervised experiment](offline-supervised.md).

## How scores are calibrated

`cmd/calibrate` runs the production path over 3,992 reference blog posts and writes `config/calibration.json`: an offset that centres each score at 0.50 and 99 quantiles for ranking. A dictionary or weight change needs a recalibration, and a test fails if the two disagree. The score range comes from each measure's own sampling error, scaled by how far two halves of 375 reference posts differ. It shows repeatability within a text, not accuracy about a person.

## Your data

Nothing is uploaded. By default nothing is stored either. If you tick "keep this reading on this device", the scores, evidence and short text excerpts are saved in your browser only, and you can delete them from the same page.

## References

- Pennebaker, J.W., Boyd, R.L., Jordan, K., & Blackburn, K. (2015). [*The development and psychometric properties of LIWC2015*](https://www.liwc.net/).
- Tweedie, F.J., & Baayen, R.H. (1998). [*How variable may a constant be? Measures of lexical richness in perspective*](https://doi.org/10.1023/A:1001749303136).
- Yarkoni, T. (2010). [*Personality in 100,000 words*](https://doi.org/10.1016/j.jrp.2010.04.001).
- Pennebaker, J.W., Chung, C.K., Frazee, J., Lavergne, G.M., & Beaver, D.I. (2014). *When small words foretell academic success: The case of college admissions essays*. PLoS ONE, 9(12), e115844. The function-word index behind Cognitive Style.
- Mairesse, F., Walker, M.A., Mehl, M.R., & Moore, R.K. (2007). [*Using linguistic cues for the automatic recognition of personality in conversation and text*](https://jair.org/index.php/jair/article/view/10520). Read as a cross-check; it uses the same kind of essays as our test set.
- Pennebaker, J.W., & King, L.A. (1999). [*Linguistic styles: Language use as an individual difference*](https://doi.org/10.1037/0022-3514.77.6.1296).
- Higgins, E.T. (1997). [*Beyond pleasure and pain*](https://doi.org/10.1037/0003-066X.52.12.1280).
- Cacioppo, J.T., & Petty, R.E. (1982). [*The need for cognition*](https://doi.org/10.1037/0022-3514.42.1.116).
- Petty, R.E., & Cacioppo, J.T. (1986). *The elaboration likelihood model of persuasion*. Background for the earlier systematic and intuitive wording.
- Webster, D.M., & Kruglanski, A.W. (1994). [*Individual differences in need for cognitive closure*](https://doi.org/10.1037/0022-3514.67.6.1049).
- Schwartz, H.A., et al. (2013). [*Choosing the right words: Characterizing and reducing error of the word count approach*](https://aclanthology.org/S13-1042/). Motivation for the word-sense audit.
- Schwartz, S.H. (1992). [*Universals in the content and structure of values*](https://doi.org/10.1016/S0065-2601(08)60281-6).
- Schler, J., Koppel, M., Argamon, S., & Pennebaker, J.W. (2006). *Effects of age and gender on blogging*. The Blog Authorship Corpus used for calibration.
