from pathlib import Path

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_prefix="", case_sensitive=False)

    ml_internal_token: str = Field(min_length=16)
    ml_models_lock: Path = Path(__file__).resolve().parent.parent / "models.lock.json"
    ml_max_image_bytes: int = 10 * 1024 * 1024
    ml_torch_threads: int = 2
    # one inference at a time per slot keeps the CPU from being oversubscribed (matching-engine section 9)
    ml_concurrency: int = 2
    # colours.csv and brands.csv, the repository copy unless the image carries its own
    ml_seed_dir: Path = Path(__file__).resolve().parent.parent.parent / "data" / "seed"
    hf_home: Path = Path.home() / ".cache" / "huggingface"
    log_level: str = "info"
    service_version: str = "dev"
