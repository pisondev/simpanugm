"""Validate the shared API/ML fixture bundle without application code."""

from __future__ import annotations

import json
import math
import re
from pathlib import Path
from typing import Any

from rapidfuzz.fuzz import token_set_ratio


ROOT = Path(__file__).resolve().parents[2]
FIXTURE_DIR = ROOT / "data" / "fixtures"
FUSION_BANDS = {
    "image": (0.60, 0.95),
    "cross_modal": (0.18, 0.33),
    "e5": (0.78, 0.93),
}
FUSION_WEIGHTS = {"image": 0.35, "cross_modal": 0.25, "text": 0.35, "location": 0.15, "time": 0.15}
# digit runs may be grouped by single spaces or hyphens, as printed on cards
DIGIT_RUN = re.compile(r"\d(?:[ -]?\d){12,18}")


def load_json(path: Path) -> dict[str, Any]:
    try:
        with path.open(encoding="utf-8") as handle:
            value = json.load(handle)
    except (OSError, json.JSONDecodeError, UnicodeError) as exc:
        raise ValueError(f"{path}: cannot read JSON") from exc
    if not isinstance(value, dict):
        raise ValueError(f"{path}: root must be an object")
    return value


def normalize(value: float, low: float, high: float) -> float:
    return min(1.0, max(0.0, (value - low) / (high - low)))


def luhn_valid(value: str) -> bool:
    total = 0
    for index, character in enumerate(reversed(value)):
        digit = int(character)
        if index % 2:
            digit *= 2
            if digit > 9:
                digit -= 9
        total += digit
    return total % 10 == 0


def mask_visible_text(value: str) -> str:
    def replace(match: re.Match[str]) -> str:
        run = match.group(0)
        digits = re.sub(r"\D", "", run)
        # BR-PRV-03: every 16 digit run, which covers NIK, plus any 13 to 19 digit run that passes Luhn
        if len(digits) == 16 or luhn_valid(digits):
            return "*" * (len(digits) - 4) + digits[-4:]
        return run

    return DIGIT_RUN.sub(replace, value)


def _score_attribute(kind: str, value: str) -> float | None:
    if value == "missing":
        return None
    if kind == "colour":
        return {"shared": 1.0, "adjacent": 0.5, "different": 0.0}[value]
    if kind == "brand":
        return {"same": 1.0, "different": 0.0}[value]
    raise ValueError(f"unknown attribute kind: {kind}")


def _time_score(time_case: dict[str, Any]) -> float:
    relation = time_case.get("relation")
    if relation == "in_window":
        return 1.0
    if relation == "before_window":
        if time_case.get("hours_before_start") is None:
            raise ValueError("before_window requires hours_before_start")
        return 0.0
    if relation == "after_window":
        hours = time_case.get("hours_after_window")
        if not isinstance(hours, (int, float)) or hours < 0:
            raise ValueError("after_window requires non-negative hours_after_window")
        return 0.5 ** (hours / 72.0)
    raise ValueError(f"unknown time relation: {relation}")


def expected_fusion(case: dict[str, Any]) -> dict[str, float | str | None]:
    mode = case["mode"]
    if mode not in {"image", "cross_modal", "none"}:
        raise ValueError(f"unknown visual mode: {mode}")

    visual_input = case.get("visual")
    visual_score: float | None = None
    if mode != "none":
        if not isinstance(visual_input, dict):
            raise ValueError("a visual input is required for image and cross_modal modes")
        key = "image_cosine" if mode == "image" else "cross_modal_cosine"
        cosine = visual_input.get(key)
        if not isinstance(cosine, (int, float)):
            raise ValueError(f"visual input requires {key}")
        visual_score = normalize(float(cosine), *FUSION_BANDS[mode])
    elif visual_input is not None:
        raise ValueError("none mode cannot have a visual input")

    text_input = case["text"]
    semantic = normalize(float(text_input["semantic_cosine"]), *FUSION_BANDS["e5"])
    attributes = [
        score
        for kind in ("colour", "brand")
        if (score := _score_attribute(kind, text_input[kind])) is not None
    ]
    attribute_score = sum(attributes) / len(attributes) if attributes else None
    text_score = 0.75 * semantic + 0.25 * attribute_score if attribute_score is not None else semantic

    location_input = case.get("location")
    location_score = None
    if location_input is not None:
        distance = location_input.get("distance_m")
        if not isinstance(distance, (int, float)) or distance < 0:
            raise ValueError("location distance must be non-negative")
        location_score = math.exp(-float(distance) / 400.0)

    time_score = _time_score(case["time"])
    components: list[tuple[float, float]] = [(FUSION_WEIGHTS["text"], text_score)]
    if visual_score is not None:
        components.insert(0, (FUSION_WEIGHTS[mode], visual_score))
    if location_score is not None:
        components.append((FUSION_WEIGHTS["location"], location_score))
    components.append((FUSION_WEIGHTS["time"], time_score))
    fused = float(case["category_factor"]) * sum(weight * score for weight, score in components) / sum(
        weight for weight, _ in components
    )
    tier = "high" if fused >= 0.72 else "medium" if fused >= 0.55 else "low" if fused >= 0.40 else "not_stored"
    return {
        "visual": visual_score,
        "semantic": semantic,
        "attributes": attribute_score,
        "text": text_score,
        "location": location_score,
        "time": time_score,
        "fused": fused,
        "stored_score": round(fused, 4),
        "tier": tier,
    }


def assert_close(actual: Any, expected: Any, label: str) -> None:
    if expected is None:
        if actual is not None:
            raise ValueError(f"{label}: expected null, got {actual!r}")
        return
    if not isinstance(actual, (int, float)) or not math.isclose(float(actual), float(expected), abs_tol=1e-6):
        raise ValueError(f"{label}: expected {expected!r}, got {actual!r}")


def validate_fusion(path: Path) -> list[str]:
    document = load_json(path)
    errors: list[str] = []
    if document.get("schema_version") != 1:
        errors.append("fusion_cases.json: unsupported schema_version")
    if document.get("fusion_version") != "fusion-linear-2026.11.0":
        errors.append("fusion_cases.json: unexpected fusion_version")
    cases = document.get("cases")
    if not isinstance(cases, list) or not cases:
        return errors + ["fusion_cases.json: cases must be a non-empty list"]
    ids: set[str] = set()
    for index, case in enumerate(cases, start=1):
        if not isinstance(case, dict):
            errors.append(f"fusion case {index}: expected an object")
            continue
        case_id = case.get("id", f"<case-{index}>")
        if not isinstance(case_id, str) or not case_id.strip():
            errors.append(f"fusion case {index}: id must be a non-empty string")
            continue
        if case_id in ids:
            errors.append(f"fusion case {case_id}: duplicate id")
        ids.add(case_id)
        try:
            calculated = expected_fusion(case)
            expected = case["expected"]
            for field in ("visual", "semantic", "attributes", "text", "location", "time", "fused", "stored_score"):
                assert_close(calculated[field], expected[field], f"fusion case {case_id} {field}")
            if calculated["tier"] != expected["tier"]:
                raise ValueError(f"tier expected {expected['tier']!r}, got {calculated['tier']!r}")
        except (KeyError, TypeError, ValueError, ZeroDivisionError) as error:
            errors.append(f"fusion case {case_id}: {error}")
    return errors


def validate_masking(path: Path) -> list[str]:
    document = load_json(path)
    errors: list[str] = []
    if document.get("schema_version") != 1 or document.get("rule") != "BR-PRV-03":
        errors.append("masking_cases.json: unsupported schema or rule")
    cases = document.get("cases")
    if not isinstance(cases, list) or not cases:
        return errors + ["masking_cases.json: cases must be a non-empty list"]
    for index, case in enumerate(cases, start=1):
        if not isinstance(case, dict):
            errors.append(f"masking case {index}: expected an object")
            continue
        case_id = case.get("id", f"<case-{index}>")
        input_value = case.get("input", "")
        if not isinstance(input_value, str):
            errors.append(f"masking case {case_id}: input must be a string")
            continue
        actual = mask_visible_text(input_value)
        if actual != case.get("expected"):
            errors.append(f"masking case {case_id}: expected {case.get('expected')!r}, got {actual!r}")
    return errors


def validate_fuzzy(path: Path) -> list[str]:
    document = load_json(path)
    errors: list[str] = []
    threshold = document.get("identity_name_threshold")
    if document.get("schema_version") != 1 or document.get("metric") != "rapidfuzz.fuzz.token_set_ratio":
        errors.append("fuzzy_cases.json: unsupported schema or metric")
    if not isinstance(threshold, (int, float)):
        return errors + ["fuzzy_cases.json: identity_name_threshold must be numeric"]
    cases = document.get("cases")
    if not isinstance(cases, list) or not cases:
        return errors + ["fuzzy_cases.json: cases must be a non-empty list"]
    for index, case in enumerate(cases, start=1):
        if not isinstance(case, dict):
            errors.append(f"fuzzy case {index}: expected an object")
            continue
        case_id = case.get("id", f"<case-{index}>")
        if not isinstance(case.get("left"), str) or not isinstance(case.get("right"), str):
            errors.append(f"fuzzy case {case_id}: left and right must be strings")
            continue
        expected_ratio = case.get("expected_ratio")
        if not isinstance(expected_ratio, (int, float)):
            errors.append(f"fuzzy case {case_id}: expected_ratio must be numeric")
            continue
        ratio = token_set_ratio(case["left"], case["right"])
        if not math.isclose(ratio, float(expected_ratio), abs_tol=1e-6):
            errors.append(f"fuzzy case {case_id}: expected {expected_ratio}, got {ratio}")
        expected_hit = ratio >= threshold
        if expected_hit != case.get("expected_name_hit"):
            errors.append(f"fuzzy case {case_id}: name-hit expectation is inconsistent")
    return errors


def validate_fixture_dir(fixture_dir: Path = FIXTURE_DIR) -> list[str]:
    errors: list[str] = []
    for filename, validator in (
        ("fusion_cases.json", validate_fusion),
        ("masking_cases.json", validate_masking),
        ("fuzzy_cases.json", validate_fuzzy),
    ):
        try:
            errors.extend(validator(fixture_dir / filename))
        except (OSError, ValueError, TypeError) as exc:
            errors.append(str(exc))
    return errors


def main() -> int:
    errors = validate_fixture_dir()
    if errors:
        for error in errors:
            print(f"ERROR: {error}")
        return 1
    print("Fixture bundle: OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
