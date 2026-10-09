# TODO

Findings from seven simulated reader personas (diary writer, angry reviewer,
psychometrician, HR manager, Gen Z user, historian, mobile/ESL reader) who read
real rendered reports before and after the "At a glance" round. Satisfaction
went from 3-5/10 to 3-7/10. The items below are what they still flagged.

## Round 2: report layer (no recalibration)

1. **Lead with the finding and the warning.** The hiring/clinical warning is the
   last line of a grey block and was skipped by a skimmer. Put the plain finding
   first, make the warning bold and visible, and move the support note up.
2. **Fix "moderate" next to a high or low rank.** The clustering clause did not
   help. Either give each card a plainer headline that matches the rank, or drop
   the bands for the narrow-spread measures.
3. **Name the reference sample on the main page.** Say "2004 blog posts", not
   "reference texts". Replace saturated ranks ("99 of 100", "1 of 100") with
   "above/below every reference text" when the score is outside the reference range.
4. **Tighten the support note.** It fires on a product review. Require a stronger
   trigger, or soften the wording.
5. **Reword the emotion-count line.** "Emotion words found" overclaims. Use "the
   dictionary matched N negative-feeling and M positive-feeling words" and say the
   list is limited.
6. **Fix the formal-prose note.** It names four measures, so readers assume the rest
   are trustworthy. Say all scores are rough and these are especially so.
7. **Extend fit notes** to Agreeableness, Extraversion and Need for Cognition on
   formal prose (driven by "space", "time", "group" and reasoning words).
8. **Show uncertainty on the page.** Show the bounds beside scores, or "can't
   tell" when they cross bands. Show the tie width for scores that sit on a tie
   (Regulatory Focus 50 spans the 37th to 77th percentile).
9. **Explain "Reading quality" beside the chip**, and consider renaming it ("Confidence
   in this reading"). It was read as the quality of the writing or the English.
10. **Jargon and labels.** Plain words for Clout, Regulatory Focus, Need for
    Closure, "signal"; consistent band vocabulary.
11. **Mobile and accessibility polish.**
    - Replace the ~27,000px inline JSON block with a download or copy button.
    - Larger caveat text (rank and "Low fit" are 12px).
    - Show a tap cue on band chips.
    - `role=meter` instead of `progressbar` for scores.
    - Heading levels in the evidence section.
    - Band chip `aria-label` overrides the visible word.
12. **Optional: personal headline** for short or shareable reports, and a "not
    enough text" view for short samples instead of nine full-size scores.
13. **PDF** (`modules/profile/pdf_maroto.go`, `narrative.go`) does not get the glance
    block or fit notes yet, and prints the raw lowercase quality flag.
14. **CI:** add a step that rebuilds assets and runs `git diff --exit-code assets/`,
    since nothing catches stale CSS.

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
2. **First-person pronoun category.** (`personal_pronoun` now exists; a first-person-singular-only list is still missing.) The single `pronoun` category mixes "that",
   "which", "these" with "I" and "my". A real first-person share would give a
   better genre signal.
3. **Quality flag.** Add an extraction-noise component (share of removed lines).
4. **Reference corpus.** 2004 blog posts are a poor comparison for essays,
   abstracts and book chapters. Consider genre-specific references.
5. **Rename or demote** Authenticity and Clout (read as character verdicts), and
   "high/low signal" (read as reliability).
6. **Weights are unverified against the paper.** The Big Five weights are
   consistent with `rho * 0.06` for every correlation quoted in
   `coefficients.go`, but those correlations were not re-checked against
   Yarkoni (2010) Table 1, and the SD assumptions are unvalidated.
7. **Accuracy is near chance.** AUC 0.532 to 0.559 on 2,442 essays. Only a
   discriminative vocabulary or the supervised model (`cmd/train`) can change that.
