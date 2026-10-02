# Cycle 1 — Interview Kit (report-reading sessions)

Companion to the [four-risks assessment](2026-10-02-four-risks.md), Cycle 1.
Goal: test value and usability with 5–8 people who write regularly about
themselves, against bars fixed in advance. Everything here is designed so the
session runs itself — print this and go.

## The bars (fixed in advance)

- **Usability PASS:** ≥ 4 of 5 participants correctly explain the percentile
  AND the confidence interval in their own words, unaided, AND trace at least
  one score back to its evidence words without help.
- **Value PASS:** ≥ 3 of 5 return unsolicited with a second writing sample
  within 7 days.
- Anything else is iterate or kill — decision rules at the bottom.

## Recruiting (start now — this is calendar time)

Target: 5–8 completed sessions; schedule 8 to net 5. Not friends, not family,
not household. Screen for: writes regularly about themselves (journal, blog,
newsletter, personal essays), can bring a real sample of 500+ words, has 45
minutes.

Recruiting message (adapt, don't embellish):

> I'm building a small tool that reads a piece of your own writing — journal
> entries, blog posts — and shows you the evidence behind a personality-style
> profile it produces. I'm looking for people who write regularly to try it
> while I watch and take notes. 45 minutes, your text never leaves my laptop,
> nothing is published. Interested?

Rules: promise nothing about accuracy; describe nothing about the report
before the session.

## Session script (45 minutes)

### 0–5 — Setup

Confirm consent for notes/recording. No tool talk yet.

### 5–20 — Past behavior (do not mention Psycho)

1. "Tell me about the last time you tried to understand your own patterns —
   what did you do?"
2. "What started that?"
3. "What did you do with what you learned?"
4. "Have you taken personality tests before? What did you do afterwards?"

Listen for frequency, pain, and what they actually did — workarounds included.
Capture quotes verbatim. If they ask what the tool does: "you'll see in a
minute."

### 20–40 — The report (think-aloud)

They paste a real sample (500+ words), you run the analysis, they read the
general-template report aloud. Your only words: "what are you thinking here?"
and "what does that mean to you?" Never explain the interface — silence is
data.

Watch for and log verbatim:

- attempts to find the evidence behind a score (feeds U3);
- how they think the percentile works (feeds U1);
- how they explain the confidence interval (feeds U2);
- over-trust ("so I *am* 90th-percentile neurotic") and under-trust;
- Barnum reactions ("that could describe anyone");
- feature requests — note, don't promise, move on.

### 40–45 — Wrap

- "What would you do with this?"
- "If you kept it, when would you use it again?"
- **Do not ask them to come back.** The value bar is unsolicited return
  within 7 days — asking for it destroys the signal.
- Thank them; say you may email in a week or two.

## Scoring sheet (one per participant)

| Check | Pass? | Quote / evidence |
|---|---|---|
| U1 — percentile explained correctly, unaided | | |
| U2 — confidence interval explained correctly, unaided | | |
| U3 — traced a score to evidence words without help | | |
| V1 — returned with a second sample within 7 days | | date: |

Notes — confusion moments, over/under-trust, requested features:

## Day before: dry run

Run one sample from `samples/` through the app (`go run ./cmd/psycho`, then
import the directory via `POST /analyze-dir`) and render the general report
(`cmd/rendertemplates`), so session day has no surprises.

## Decision rules (from the discovery doc)

- **Usability fail** → rework the report's explanation layer (glossary,
  worked example) and re-run with 3 new readers.
- **Value fail** → build nothing; drop to problem interviews before touching
  the roadmap.
- **No second samples and no evidence-trail curiosity** → the auditability
  bet is dead as framed: kill or pivot the audience. Killing at this stage
  costs one week and is a success.
