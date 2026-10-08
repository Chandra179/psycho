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
15. **Known test failure:** `TestAnalyzeDirWithDataSamples` expects 8 files in
    `samples/`; the untracked `samples/ancient.txt` makes it 9. Either commit the
    file and change the expectation, or keep it out of `samples/`.

## Phase 2: scoring layer (needs `go run ./cmd/calibrate` against local `corpus/`)

Any dictionary edit changes `dictionary_sha256` and forces recalibration
(`TestCalibrationMatchesDictionary`). Do not tune against the final-test labels.

1. **Dictionary false matches.** Polysemous words count as values: "just"
   (Universalism and exclusive), "kind", "content", "natural", "care", "support",
   "rule", "independent". Add context or sense rules, a minimum match count, and
   split endorsing a value from rejecting it ("did not care"). Apply the same rule in
   `features.go` and `excerpts.go`.
2. **Value excerpts.** Pick the strongest or most representative matches, not the
   first two; show which word matched; say value counts describe subject matter.
3. **Normalize noise.** `Normalize()` should strip running page headers, bare page
   numbers, figure captions and map legends, and rejoin hyphenation. Bump
   `model_fingerprint.go` when it changes.
4. **Long-word term.** It counts bytes, not letters (contractions and accents count),
   and dominates Cognitive Style and Authenticity on formal text. Count runes and
   rebalance.
5. **Emotional tone.** The 0.35-0.65 "neutral" band is too wide, there is no negation
   handling, and the negative list is short (misses "cried", "scream", "scam",
   "garbage", "disaster"). Relabel when negatives outnumber positives.
6. **Trait-specific bounds.** One width of about +-0.22 applies to all nine measures
   and ignores the reference spread (SD 0.01-0.04 for most). Scale per measure.
7. **Percentile quality.** Calibrated scores are rounded to two decimals, so the
   table has only 7-41 distinct values and many ties. Use finer resolution.
8. **First-person pronoun category.** The single `pronoun` category mixes "that",
   "which", "these" with "I" and "my". A real first-person share would enable a better
   genre signal.
9. **Quality flag.** Low dictionary coverage can never produce "low". Let it, and add
   an extraction-noise component.
10. **Reference corpus.** 2004 blog posts are a poor comparison for essays, abstracts
    and book chapters. Consider genre-specific references.
11. **Rename or demote** Authenticity and Clout (read as character verdicts), and
    "high/low signal" (read as reliability).
