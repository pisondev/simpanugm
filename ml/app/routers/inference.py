import json
from typing import Annotated, Literal

from fastapi import APIRouter, File, Form, Request, UploadFile
from pydantic import BaseModel, Field
from starlette.concurrency import run_in_threadpool

from app import imaging
from app.analysis.category import CategoryPrompts
from app.errors import ProblemError
from app.inference import Inference
from app.registry import IMAGE_ENCODER, TEXT_ENCODER, XMODAL_TEXT_ENCODER

router = APIRouter(prefix="/v1")

MAX_IMAGES = 8
MAX_TEXTS = 32
MAX_CATEGORIES = 200


class Embeddings(BaseModel):
    component: str
    model: str
    revision: str
    embeddings: list[list[float]]


class TextRequest(BaseModel):
    texts: list[Annotated[str, Field(max_length=4000)]]
    kinds: list[Literal["e5", "clip_text"]] = Field(min_length=1)


class ScoredCategory(BaseModel):
    slug: str
    score: float


class DetectedColour(BaseModel):
    name: str
    hex: str
    share: float


class VisibleText(BaseModel):
    value: str
    masked: bool


class BrandHit(BaseModel):
    name: str
    score: float


class AnalysisResult(BaseModel):
    categories: list[ScoredCategory]
    colours: list[DetectedColour]
    text: VisibleText
    brands: list[BrandHit]
    versions: dict[str, str]


def _inference(request: Request) -> Inference:
    registry = request.app.state.registry
    if not registry.ready:
        raise ProblemError(503, "not_ready", "models are still loading")
    inference: Inference = request.app.state.inference
    return inference


def _need(inference: Inference, component: str) -> None:
    if component not in inference.registry.components:
        raise ProblemError(503, "not_ready", f"{component} is not in models.lock.json")


async def _read(files: list[UploadFile], limit: int) -> list[bytes]:
    out = []
    for f in files:
        data = await f.read(limit + 1)
        out.append(data)
    return out


@router.post("/embed/image")
async def embed_image(request: Request, files: Annotated[list[UploadFile], File()]) -> Embeddings:
    inference = _inference(request)
    _need(inference, IMAGE_ENCODER)
    if not files:
        raise ProblemError(400, "invalid_request", "at least one file is required")
    if len(files) > MAX_IMAGES:
        raise ProblemError(413, "too_many_items", f"at most {MAX_IMAGES} images per request")
    limit = request.app.state.settings.ml_max_image_bytes
    images = [imaging.decode(data, limit) for data in await _read(files, limit)]
    vectors = await run_in_threadpool(inference.embed_images, images)
    spec = inference.registry.get(IMAGE_ENCODER)
    return Embeddings(component=IMAGE_ENCODER, model=spec.model_id, revision=spec.revision, embeddings=vectors.tolist())


@router.post("/embed/text")
def embed_text(request: Request, body: TextRequest) -> dict[str, Embeddings]:
    inference = _inference(request)
    if not body.texts:
        raise ProblemError(400, "invalid_request", "at least one text is required")
    if len(body.texts) > MAX_TEXTS:
        raise ProblemError(413, "too_many_items", f"at most {MAX_TEXTS} texts per request")
    out: dict[str, Embeddings] = {}
    for kind in dict.fromkeys(body.kinds):
        component = TEXT_ENCODER if kind == "e5" else XMODAL_TEXT_ENCODER
        _need(inference, component)
        spec = inference.registry.get(component)
        vectors = inference.embed_texts(body.texts, kind)
        out[kind] = Embeddings(
            component=component, model=spec.model_id, revision=spec.revision, embeddings=vectors.tolist()
        )
    return out


def _categories(raw: str) -> list[CategoryPrompts]:
    try:
        items = json.loads(raw)
        cats = [CategoryPrompts(str(i["slug"]), [str(p) for p in i["prompts"]]) for i in items]
    except (ValueError, KeyError, TypeError) as exc:
        raise ProblemError(422, "invalid_request", "categories must be a JSON list of {slug, prompts}") from exc
    if len(cats) > MAX_CATEGORIES:
        raise ProblemError(413, "too_many_items", f"at most {MAX_CATEGORIES} categories")
    return cats


@router.post("/analyze/image")
async def analyze_image(
    request: Request,
    file: Annotated[UploadFile, File()],
    categories: Annotated[str, Form()] = "[]",
    prompt_set_hash: Annotated[str, Form(max_length=128)] = "",
) -> AnalysisResult:
    inference = _inference(request)
    for component in (IMAGE_ENCODER, "ocr"):
        _need(inference, component)
    limit = request.app.state.settings.ml_max_image_bytes
    image = imaging.decode((await _read([file], limit))[0], limit)
    cats = _categories(categories)
    result = await run_in_threadpool(inference.analyze, image, cats, prompt_set_hash or "none")
    return AnalysisResult(
        categories=[ScoredCategory(slug=c.slug, score=c.score) for c in result.categories],
        colours=[DetectedColour(name=c.name, hex=c.hex, share=c.share) for c in result.colours],
        text=VisibleText(value=result.text, masked=result.masked),
        brands=[BrandHit(name=b.name, score=b.score) for b in result.brands],
        versions=result.versions,
    )
