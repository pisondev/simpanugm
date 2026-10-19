import numpy as np
import sklearn
from fastapi import APIRouter, Request
from pydantic import BaseModel, Field
from sklearn.cluster import HDBSCAN
from starlette.concurrency import run_in_threadpool

from app.errors import ProblemError
from app.metrics import measure_inference

router = APIRouter(prefix="/v1")

MAX_CLUSTER_ITEMS = 5000


class ClusterItem(BaseModel):
    id: str
    vector: list[float] = Field(min_length=1)


class ClusterRequest(BaseModel):
    items: list[ClusterItem]
    min_cluster_size: int = Field(default=2, ge=2)
    min_samples: int = Field(default=1, ge=1)


class ClusterLabel(BaseModel):
    id: str
    cluster: int
    probability: float


class ClusterResult(BaseModel):
    labels: list[ClusterLabel]
    version: str


def _run_hdbscan(items: list[ClusterItem], min_cluster_size: int, min_samples: int) -> list[ClusterLabel]:
    if not items:
        raise ProblemError(400, "invalid_request", "at least one item is required")
    if len(items) > MAX_CLUSTER_ITEMS:
        raise ProblemError(413, "too_many_items", f"at most {MAX_CLUSTER_ITEMS} items per request")

    dim = len(items[0].vector)
    mat = np.empty((len(items), dim), dtype=np.float32)
    for i, it in enumerate(items):
        if len(it.vector) != dim:
            raise ProblemError(400, "invalid_request", "all vectors must have the same dimension")
        mat[i] = it.vector

    if len(items) < min_cluster_size:
        return [ClusterLabel(id=it.id, cluster=-1, probability=0.0) for it in items]

    norms = np.linalg.norm(mat, axis=1, keepdims=True)
    zero_norms = (norms == 0).flatten()
    norms[zero_norms] = 1.0
    norm_mat = mat / norms

    clusterer = HDBSCAN(
        min_cluster_size=min_cluster_size,
        min_samples=min_samples,
        metric="euclidean",
        copy=True,
    )
    with measure_inference("cluster"):
        clusterer.fit(norm_mat)

    return [
        ClusterLabel(
            id=it.id,
            cluster=int(clusterer.labels_[i]),
            probability=float(clusterer.probabilities_[i]),
        )
        for i, it in enumerate(items)
    ]


@router.post("/cluster")
async def cluster_vectors(request: Request, body: ClusterRequest) -> ClusterResult:
    if not body.items:
        raise ProblemError(400, "invalid_request", "at least one item is required")
    if len(body.items) > MAX_CLUSTER_ITEMS:
        raise ProblemError(413, "too_many_items", f"at most {MAX_CLUSTER_ITEMS} items per request")

    labels = await run_in_threadpool(_run_hdbscan, body.items, body.min_cluster_size, body.min_samples)
    return ClusterResult(
        labels=labels,
        version=f"sklearn-hdbscan-{sklearn.__version__}",
    )
