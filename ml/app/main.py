import logging
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

from fastapi import Depends, FastAPI
from prometheus_fastapi_instrumentator import Instrumentator

from app import logging as applog
from app.config import Settings
from app.registry import Registry
from app.routers import health, models
from app.security import require_token

log = logging.getLogger("simpanugm.ml")


def create_app(settings: Settings | None = None) -> FastAPI:
    settings = settings or Settings()  # type: ignore[call-arg]
    applog.configure(settings.log_level, settings.service_version)
    registry = Registry.from_lock(settings.ml_models_lock)

    @asynccontextmanager
    async def lifespan(app: FastAPI) -> AsyncIterator[None]:
        registry.load_all()
        log.info("models loaded", extra={"components": len(registry.components)})
        yield

    app = FastAPI(title="SimpanUGM inference", version=settings.service_version, lifespan=lifespan)
    app.state.registry = registry
    app.state.settings = settings
    app.include_router(health.router)
    app.include_router(models.router, dependencies=[Depends(require_token(settings.ml_internal_token))])
    Instrumentator().instrument(app).expose(app, include_in_schema=False)
    return app
