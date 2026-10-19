import time
from collections.abc import Generator
from contextlib import contextmanager

from prometheus_client import Gauge, Histogram

MODELS_LOADED = Gauge(
    "ml_models_loaded",
    "Loaded machine learning models and revisions",
    ["component", "revision"],
)

INFERENCE_SECONDS = Histogram(
    "ml_inference_seconds",
    "Inference latency per model component in seconds",
    ["component"],
    buckets=[0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0],
)

REQUEST_DURATION_SECONDS = Histogram(
    "ml_request_duration_seconds",
    "ML HTTP request duration in seconds by handler and status",
    ["handler", "status"],
    buckets=[0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0],
)


@contextmanager
def measure_inference(component: str) -> Generator[None, None, None]:
    start = time.perf_counter()
    try:
        yield
    finally:
        INFERENCE_SECONDS.labels(component=component).observe(time.perf_counter() - start)
