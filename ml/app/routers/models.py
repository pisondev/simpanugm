from fastapi import APIRouter, Request
from pydantic import BaseModel

router = APIRouter(prefix="/v1")


class ModelInfo(BaseModel):
    component: str
    model_id: str
    revision: str
    dimension: int | None
    loaded_at: str | None


class ModelList(BaseModel):
    items: list[ModelInfo]


@router.get("/models")
def list_models(request: Request) -> ModelList:
    registry = request.app.state.registry
    items = [
        ModelInfo(
            component=c.name,
            model_id=c.model_id,
            revision=c.revision,
            dimension=c.dimension,
            loaded_at=registry.loaded_at[c.name].isoformat() if c.name in registry.loaded_at else None,
        )
        for c in registry.components.values()
    ]
    return ModelList(items=items)
