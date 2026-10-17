from pathlib import Path

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_prefix="", case_sensitive=False)

    ml_internal_token: str = Field(min_length=16)
    ml_models_lock: Path = Path(__file__).resolve().parent.parent / "models.lock.json"
    ml_max_image_bytes: int = 10 * 1024 * 1024
    ml_torch_threads: int = 2
    log_level: str = "info"
    service_version: str = "dev"
