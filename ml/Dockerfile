FROM python:3.12-slim
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1 UV_COMPILE_BYTECODE=1 UV_LINK_MODE=copy
COPY --from=ghcr.io/astral-sh/uv:0.13 /uv /usr/local/bin/uv
WORKDIR /app
COPY pyproject.toml uv.lock ./
RUN uv sync --frozen --no-dev --no-install-project
COPY app ./app
COPY eval ./eval
COPY models.lock.json ./
# the palette and the brand list come from the reviewed seed, compose passes data/seed as the build context seed
COPY --from=seed colours.csv brands.csv /app/seed/
RUN useradd --system --uid 10001 --home /app ml && mkdir -p /models && chown ml /models
USER ml
ENV PATH="/app/.venv/bin:$PATH" HF_HOME=/models ML_SEED_DIR=/app/seed
EXPOSE 8100
# the first start downloads about 1.5 GB of weights into the volume, so readiness has a long grace period
HEALTHCHECK --interval=15s --timeout=3s --start-period=900s CMD ["python", "-c", "import urllib.request, sys; sys.exit(0 if urllib.request.urlopen('http://127.0.0.1:8100/readyz', timeout=2).status == 200 else 1)"]
CMD ["uvicorn", "app.main:create_app", "--factory", "--host", "0.0.0.0", "--port", "8100", "--workers", "1"]
