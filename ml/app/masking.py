import re

# BR-PRV-03, the same rule as api/internal/privacy and data/fixtures/masking_cases.json
_DIGIT_RUN = re.compile(r"\d(?:[ -]?\d){12,18}")


def _luhn(digits: str) -> bool:
    if not 13 <= len(digits) <= 19:
        return False
    total = 0
    double = False
    for ch in reversed(digits):
        d = int(ch)
        if double:
            d *= 2
            if d > 9:
                d -= 9
        total += d
        double = not double
    return total % 10 == 0


def _mask(match: re.Match[str]) -> str:
    run = match.group(0)
    digits = "".join(c for c in run if c.isdigit())
    if len(digits) == 16 or _luhn(digits):
        return "*" * (len(digits) - 4) + digits[-4:]
    return run


def mask_numbers(text: str) -> str:
    return _DIGIT_RUN.sub(_mask, text)
