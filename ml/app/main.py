import asyncio
import logging
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

from fastapi import Depends, FastAPI
from prometheus_fastapi_instrumentator import Instrumentator

from app import errors
from app import logging as applog
from app.backend import Backend, TorchBackend
from app.config import Settings
from app.inference import Inference
from app.registry import Registry
from app.routers import health, inference, models
from app.security import require_token

log = logging.getLogger("simpanugm.ml")


def create_app(settings: Settings | None = None, backend: Backend | None = None) -> FastAPI:
    """The backend is injected by tests, the service builds the PyTorch one from the lock."""
    settings = settings or Settings()  # type: ignore[call-arg]
    applog.configure(settings.log_level, settings.service_version)
    registry = Registry.from_lock(settings.ml_models_lock)
    chosen: Backend = backend or TorchBackend(registry, settings.hf_home, settings.ml_torch_threads)

    @asynccontextmanager
    async def lifespan(app: FastAPI) -> AsyncIterator[None]:
        # loading takes minutes on a first start, /healthz answers meanwhile and /readyz reports loading
        task = asyncio.create_task(asyncio.to_thread(_load, chosen, registry, app, settings))
        yield
        task.cancel()

    app = FastAPI(title="SimpanUGM inference", version=settings.service_version, lifespan=lifespan)
    app.state.registry = registry
    app.state.settings = settings
    errors.install(app)
    app.include_router(health.router)
    guard = [Depends(require_token(settings.ml_internal_token))]
    app.include_router(models.router, dependencies=guard)
    app.include_router(inference.router, dependencies=guard)
    Instrumentator().instrument(app).expose(app, include_in_schema=False)
    return app


def _load(backend: Backend, registry: Registry, app: FastAPI, settings: Settings) -> None:
    try:
        backend.load()
        if not registry.ready:
            registry.load_all()
        app.state.inference = Inference(backend, registry, settings.ml_seed_dir, settings.ml_concurrency)
        log.info("models loaded", extra={"components": len(registry.components)})
    except Exception:
        # a half loaded service must never report ready, the container restart policy tries again
        log.exception("model loading failed")
        raise
