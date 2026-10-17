# Twenty-item trial checklist

The trial is a process rehearsal before the 150-item collection. It is complete only
when the labels, descriptions, images, and consent records pass the checks below.

## Intake targets

- 20 physical items with IDs `I001` through `I020`.
- Coverage across ordinary, distinctive, and visually similar objects.
- At least one mock identity-card prop; no real identity documents.
- At least two finder photos per item.
- Owner photo for a subset, taken in a different place and light from finder photos.
- One owner description from memory and one finder description while looking at the item.
- Independent category and distinctiveness labels from two people, with disagreements
  recorded for review.

## Collection order

1. Verify consent and assign an item code before taking photographs.
2. Photograph the item in a campus-like location using default phone settings.
3. Have the owner write the owner description without looking at the item.
4. Have a different contributor write the finder description while looking at it.
5. Capture labels in `trial-intake-template.csv` without private details.
6. Strip EXIF metadata and review every image for people, reflections, screens, and real
   documents.
7. Run the label validator, compare the two private label sheets with
   `data/scripts/check_agreement.py`, then complete the two-person review.

## Exit review

The trial reviewer records pass/fail for each item, the reason for every retake, category
disagreements, missing photos, consent status, and the final Drive path. The review log
and all raw material stay outside git. Record the aggregate Cohen's kappa values in that
private review log; create it with `create_trial_review_workspace.py` and do not publish
the annotation sheets or completed log.
