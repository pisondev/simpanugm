# Shared fixtures

These files are the small, deterministic contract between the future `api` and `ml`
implementations. They contain synthetic values only: no participant names, photographs,
identity documents, or collected descriptions belong here.

## Files

| File | Contract |
|---|---|
| `fusion_cases.json` | Raw component inputs and expected component, fused, stored-score, and tier values from `matching-engine.md` sections 6 and 7. |
| `masking_cases.json` | BR-PRV-03 examples, including valid 13-, 16-, and 19-digit runs and non-matching runs. |
| `fuzzy_cases.json` | `rapidfuzz.fuzz.token_set_ratio` outputs and the 85-point identity-name decision. |
| `images/` | Reserved for small synthetic media fixtures. Binary media is added only when the media contract is implemented. |

Validate the complete bundle from the repository root:

```text
python3 data/scripts/check_fixtures.py
```

The validator is intentionally independent of the application services. It catches drift
inside the shared data before the Go and Python contract tests consume it.
