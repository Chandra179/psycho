# Four-Risks Discovery Assessment — 2026-10-02

Status: open. Cycles 0 and 1 are pending; outcomes are logged at the bottom of
this document as they conclude.

This document records a product discovery assessment of Psycho as built, run
with the INSPIRED method (Marty Cagan): before committing further engineering
resources, test the four product risks — value, usability, feasibility, and
business viability — in small, cheap cycles with pass/fail criteria fixed in
advance. Discovery work is not tracked in the PRD; a change graduates into the
PRD only when a cycle concludes and proves it.

## Evidence base

What is known today, from the code and the docs:

- The pipeline (ingest → analyze → profile) works end to end, is deterministic,
  and is pinned by automated tests including known-profile text fixtures.
- A 5,000-word corpus analyzes in a median of 5 ms against a 5-second target.
- PDF export and all three report templates (general, technical, balanced) are
  implemented.
- Measured accuracy against a 2,442-essay labeled corpus: all five Big Five
  dimensions rank above chance (AUC 0.52–0.56, confidence intervals excluding
  coin-flip). Dictionary coverage is about 58%.
- The overview names five use cases (journaling/self-reflection, corpus
  comparison, research, writing-sample analysis, personal archives) and
  commits to none of them.
- No user has ever been observed reading a report.

## The four risks

| Risk | Verdict | Evidence |
|------|---------|----------|
| Feasibility | Low — mostly retired | Pipeline works end to end; latency 1000× inside target; deterministic and pinned by tests; PDF and templates are real code. One residual: whether the accuracy strategy itself has headroom (Cycle 0). |
| Value | High — the killer risk | Five use cases, none chosen. AUC 0.52–0.56 is honest but modest, and the free alternative — a 15-minute validated Big Five questionnaire — has far better psychometrics. The untested bet is that auditability + your own words + privacy carries the value where raw accuracy cannot. Nobody has been observed choosing this product. |
| Usability | Medium-high — untested | The report is the product, and it leans on two cognitively demanding devices: percentile-vs-blog-corpus framing and confidence intervals. Failure modes cut both ways: confusion, or the Barnum effect (over-trusting vague narrative prose). No user has been observed reading a report. |
| Business viability | Low — one flag | Local-first single-user deliberately scopes away sales, support, and legal machinery. Residual: the tool can analyze third-party text with no consent gate — acceptable on one machine, a real ethical/reputational (and GDPR / EU AI Act profiling) issue the day it is distributed. See the [P3] issue. |

## Ranked assumptions

Written as testable hypotheses, most uncertain first.

### 1. Value — writers will choose this over validated tests

We believe people who write regularly (journalers, bloggers) are curious about
their own psychological patterns but distrust opaque quiz-style tests. We
believe an auditable profile of their own writing will earn that trust. We'll
know it's true when at least 3 of 5 test users return unsolicited with a second
writing sample within 7 days. Behavior only — stated intent is not evidence.

### 2. Usability — readers hold correct beliefs after one read

We believe the general-template report communicates what each score does and
does not claim. We'll know it's true when at least 4 of 5 readers explain
confidence intervals and percentiles correctly in their own words, unaided, and
can trace at least one score back to its evidence words without help.

### 3. Feasibility of quality — dictionary growth lifts accuracy

We believe growing the dictionary is the path to useful accuracy, as the
overview claims. We'll know it's true when AUC rises meaningfully with
dictionary size — projected at or above 0.60 at 2× — rather than plateauing
near the current 0.52–0.56.

### 4. Business viability — third-party text without consent

The tool analyzes any text; nothing prevents profiling a person who never
consented. Fine on a single machine; unacceptable by default the moment Psycho
is published or distributed. A decision item, not a discovery cycle.

## Cycle 0 — quality-ceiling spike

Run first. Costs about a day and zero users, and its result changes the value
pitch shown in Cycle 1 — never oversell accuracy in an interview before knowing
the ceiling.

**Method.** A technical spike using what exists: `cmd/evaluate` and the
2,442-essay corpus. Rescore the corpus with the dictionary ablated at 50%,
75%, and 100% (by category; optionally by word) and read the AUC-vs-size
curve. Optionally split style/function-word categories from content categories
to locate where the signal actually lives.

**Pass.** The curve rises with size — 0.60+ AUC projected at 2× dictionary.
Iterate: dictionary growth is an evidenced quality strategy; revisit issue #19
(discriminative vocabulary selection) with data.

**Fail.** Flat curve (under 0.01 gain from 50% to 100%). Pivot the strategy:
new feature families (function words, syntax, readability) or reframe value
around auditability and reflection rather than accuracy. Either way, stop
grinding the dictionary.

## Cycle 1 — value + usability batch

Five to eight sessions in one week. Live-data prototype: the real product.

**Segment.** People who already write regularly about themselves — frequency,
existing behavior, and they own the text. Not friends or family. The research
segment is deliberately cut (see below).

**Protocol.** Per 45-minute session: 15 minutes on past behavior ("tell me
about the last time you tried to understand your own patterns — what did you
do?"), no pitching. Then the participant pastes a real writing sample, reads
their general-template report thinking aloud, and the facilitator says nothing.

**Usability bar.** At least 4 of 5 correctly explain the percentile and the
confidence interval in their own words, unaided, and trace at least one score
to its evidence words.

**Value bar.** At least 3 of 5 return unsolicited with a second writing sample
within 7 days.

**Iterate.** Comprehension failure → rework the report's explanation layer
(glossary, worked example) and re-run with 3 new readers. Value failure →
build nothing; drop to problem interviews before touching the roadmap.

## Kill criterion for the whole product

If no test user returns a second sample and none connects the evidence trail
to their own curiosity, the auditability bet is dead as currently framed.
Kill it or pivot the audience. Killing at this stage costs one week and is a
success, not a failure.

## Deliberate cut: the research segment

Dropped from the MVP segments. Researchers need validated instruments, and
AUC 0.55 will never clear that bar. Keeping it on the list muddies who Psycho
is being built for.

## Sequencing rule

Run Cycle 0 before Cycle 1: its result determines what may honestly be said
about accuracy in the Cycle 1 sessions.

## Outcomes log

| Date | Cycle | Result | Decision |
|------|-------|--------|----------|
| — | — | — | — |
