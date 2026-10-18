from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse

router = APIRouter()


@router.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}


@router.get("/readyz")
def readyz(request: Request) -> JSONResponse:
    registry = request.app.state.registry
    if not registry.ready or not hasattr(request.app.state, "inference"):
        return JSONResponse({"status": "loading"}, status_code=503)
    return JSONResponse({"status": "ready", "components": len(registry.components)})
