from fastapi import FastAPI, Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse
from starlette.exceptions import HTTPException as StarletteHTTPException


class ProblemError(Exception):
    """An error with a stable code, rendered as problem details (matching-engine section 9)."""

    def __init__(self, status: int, code: str, detail: str) -> None:
        super().__init__(detail)
        self.status = status
        self.code = code
        self.detail = detail


def problem(status: int, code: str, detail: str) -> JSONResponse:
    body = {
        "type": f"https://simpanugm.my.id/problems/ml/{code}",
        "title": code.replace("_", " "),
        "status": status,
        "code": code,
        "detail": detail,
    }
    return JSONResponse(body, status_code=status, media_type="application/problem+json")


def install(app: FastAPI) -> None:
    @app.exception_handler(ProblemError)
    async def _problem(_: Request, exc: ProblemError) -> JSONResponse:
        return problem(exc.status, exc.code, exc.detail)

    @app.exception_handler(StarletteHTTPException)
    async def _http(_: Request, exc: StarletteHTTPException) -> JSONResponse:
        code = {401: "unauthenticated", 404: "not_found", 405: "method_not_allowed"}.get(exc.status_code, "error")
        return problem(exc.status_code, code, str(exc.detail))

    @app.exception_handler(RequestValidationError)
    async def _validation(_: Request, exc: RequestValidationError) -> JSONResponse:
        first = exc.errors()[0] if exc.errors() else {}
        where = ".".join(str(p) for p in first.get("loc", ()))
        return problem(422, "invalid_request", f"{where}: {first.get('msg', 'invalid')}")
