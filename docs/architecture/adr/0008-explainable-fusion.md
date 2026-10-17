# ADR-0008 Explainable weighted fusion first, learned fusion only through the evaluation gate

Status: Accepted, 2026-10-17

## Context

The concept document requires that officers understand why a candidate is recommended
(Explainable Match View) and that AI remains decision support. Labelled data at launch is
limited to the team's evaluation set. Signals can be missing: many lost reports have no
photo, some have no place.

## Decision

Score pairs with a weighted mean of normalised components over the available signals,
multiplied by a category factor (matching-engine section 7), with every constant in one
versioned configuration document. A logistic regression on the same components may
replace the weighted mean only if it passes the gates of evaluation section 8 and beats
the baseline by the stated margin.

## Consequences

- Every displayed score decomposes into parts a person can read, and every stored score
  can be recomputed from its components and its configuration version.
- Missing signals are handled by renormalising weights instead of imputing values.
- The weighted mean leaves accuracy on the table compared with a learned ranker. The gate
  makes that trade visible with numbers instead of opinions.
- Even learned fusion stays a linear model on the same components, so explanations keep
  working.

## Alternatives

- **A learned ranker on raw features, such as gradient boosted trees.** Likely more
  accurate with enough labels, which we do not have, and much harder to explain per pair.
- **Image similarity only.** Fails exactly in the most common case, the owner without a
  photo.
