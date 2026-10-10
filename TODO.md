# TODO

Findings from seven simulated reader personas (diary writer, angry reviewer,
psychometrician, HR manager, Gen Z user, historian, mobile/ESL reader) who read
real rendered reports before and after the "At a glance" round. Satisfaction
went from 3-5/10 to 3-7/10. The items below are what they still flagged.

## Round 2: report layer (no recalibration)

Done on 2026-10-10: a "Read this first" notice under the header carries the
hiring/clinical caveat, the reasons the confidence is not high, the formal-prose
and sparse-emotion notes and the support note (items 1 and 4 of the old list);
the formal-prose note says every score is rough and now covers Extraversion
(the formal samples score 0.44 to 0.45 against 0.53 for the diary entry;
Agreeableness and Need for Cognition showed no consistent shift, so they are
not flagged); "Reading quality" is "Confidence in this reading" with a line
saying it does not rate the writing; the calculation JSON is a download link
instead of an inline block; scores use `role="meter"`; the low-fit
note is 14px; the PDF shows the caveat and the capitalized confidence word;
CI rebuilds the CSS and fails on a stale `assets/app.css`.

Dropped because the rank sentences and "At a glance" were removed: saturated
ranks, the support-note trigger on its own, the emotion-count line wording
(the Emotional tone card says "Based on N negative-feeling and M
positive-feeling dictionary words").

Still open:

1. **Uncertainty on the page.** Show the bounds beside scores, or "can't tell"
   when they cross bands; show the tie width for scores that sit on a tie
   (Regulatory Focus 50 spans the 37th to 77th percentile).
2. **Jargon and labels.** Renamed on 2026-10-10 (display names only; keys are
   unchanged): Clout is "Confident wording", Regulatory Focus is "Goals: gain
   vs. safety", Need for Closure is "Preference for certainty". Still open:
   plain words for "signal" and a consistent band vocabulary.
3. **Mobile and accessibility.** A tap cue on band chips; heading levels in the
   evidence section.
4. **Optional:** a personal headline for short or shareable reports, and a "not
   enough text" view for short samples instead of nine full-size scores.
5. **PDF** does not show the per-measure fit notes.

## Phase 2: scoring layer

Done on 2026-10-09 (dictionary and rules changed, `config/calibration.json`
regenerated; AUC moved by at most 0.004 for the first fixes, then to 0.532 to 0.559 after the word-sense audit):

- Value lists no longer match everyday words ("just", "kind", "content",
  "natural", "care", "support", "rule", "order", "control" and others); value
  matches after a negator are not counted; excerpts pick the richest sentences.
- Long words and average length count letters, not bytes, and no score uses long words any more. Cognitive Style is now the Pennebaker et al. (2014) function-word index (new dictionary categories: personal and impersonal pronouns, auxiliary verbs, adverbs, conjunctions) and Authenticity lost its long-word term.
- `Normalize()` drops page numbers, captions and running headers and rejoins
  hyphenated words.
- Emotional tone flips negated emotion words, uses tighter bands, and the
  negative list gained "cried", "scream", "scam", "garbage", "disaster" and others.
- Bounds are per measure, from the sampling error of each measure's own weights,
  scaled by a split-half check on 375 reference posts.
- Ranks use unrounded calibrated scores (73 to 99 distinct quantiles instead of 7 to 41).
- Low dictionary coverage (under 45%) now gives a "low" reading quality.
- `TestAnalyzeDirWithDataSamples` counts the files in `samples/`.

Still open (any dictionary edit changes `dictionary_sha256` and needs
`go run ./cmd/calibrate -corpus corpus/`; do not tune against the final-test labels):

1. **Trait lists are audited for ambiguity, not read in context.** Done on
   2026-10-09: a concordance audit (20 contexts per word, one annotator) removed 18
   word entries ("just", "will", "so", "as", "up", "right" ...); see
   `modules/analyze/dictionary_exclusions.json`. The Schwartz et al. (2013) top-sense
   filter (theta 0.50, WordNet 3.0 tag counts) was also built and measured, then
   reverted: it removed 418 more entries, cost 2.5 points of coverage and gave
   no accuracy gain (mean AUC 0.540 against 0.545), and it removes many in-sense
   words. Residual: occurrences are still counted without disambiguation. The
   production Big Five weights are `rho * 0.06`, not fitted, so a word-list change
   needs `cmd/calibrate`, not a `cmd/train` refit.
2. **First-person pronoun category.** Done on 2026-10-10: `first_person_singular`
   (i, me, my, mine, myself and contractions) is counted but has no score
   weight. It separates formal samples (0.0% to 0.04%) from personal ones (7.7%
   to 12%) and now guards the formal-prose note. Scores and AUC are unchanged.
3. **Quality flag.** Done on 2026-10-10: a noise share of 5% or more of lines
   (page numbers, captions, headers) caps the flag at medium. The threshold is a
   judgement: no reference post lost a line.
4. **Reference corpus.** 2004 blog posts are a poor comparison for essays,
   abstracts and book chapters. Consider genre-specific references.
5. **Rename or demote** Authenticity and Clout (read as character verdicts), and
   "high/low signal" (read as reliability).
6. **Weights vs the paper.** Checked on 2026-10-10 against Table 1 of Yarkoni
   (2010) (open manuscript PMC2885844): all 32 weights equal rho * 0.06; the two
   resting on non-significant correlations (pronouns with Extraversion and
   Neuroticism) were removed. Still unvalidated: the 0.06 scale (assumed SDs
   of 0.15 and 2.5 points), and the project's `pronoun` category includes
   demonstratives that the paper's total-pronoun row does not.
7. **Accuracy is near chance.** AUC 0.532 to 0.559 on 2,442 essays. The supervised
   model (`cmd/train`, rerun 2026-10-10) reaches 0.57 to 0.63 on 489 held-out
   authors, ahead of the heuristic on every trait, but no 99% paired interval
   excludes zero and it is untested outside student essays.
