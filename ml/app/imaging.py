import io

from PIL import Image, ImageOps, UnidentifiedImageError

from app.errors import ProblemError

ACCEPTED = {"JPEG", "WEBP", "PNG"}
# a decompression bomb guard on top of the byte limit
MAX_PIXELS = 40_000_000


def decode(data: bytes, max_bytes: int) -> Image.Image:
    if len(data) > max_bytes:
        raise ProblemError(413, "too_large", f"image is larger than {max_bytes} bytes")
    try:
        probe = Image.open(io.BytesIO(data))
        if probe.format not in ACCEPTED:
            raise ProblemError(400, "invalid_image", "only JPEG, WebP and PNG images are accepted")
        if probe.width * probe.height > MAX_PIXELS:
            raise ProblemError(413, "too_large", "image has too many pixels")
        image = ImageOps.exif_transpose(Image.open(io.BytesIO(data)))
        return image.convert("RGB")
    except (UnidentifiedImageError, OSError, ValueError, SyntaxError) as exc:
        raise ProblemError(400, "invalid_image", "the file is not a readable image") from exc


def downscale(image: Image.Image, longest: int) -> Image.Image:
    if max(image.size) <= longest:
        return image
    scale = longest / max(image.size)
    return image.resize((round(image.width * scale), round(image.height * scale)), Image.Resampling.LANCZOS)
