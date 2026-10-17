import hmac
from collections.abc import Callable

from fastapi import Header, HTTPException, status


def require_token(expected: str) -> Callable[[str | None], None]:
    def check(authorization: str | None = Header(default=None)) -> None:
        scheme, _, token = (authorization or "").partition(" ")
        if scheme.lower() != "bearer" or not hmac.compare_digest(token.encode(), expected.encode()):
            raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="unauthenticated")

    return check
