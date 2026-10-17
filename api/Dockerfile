# build context is the repository root, so the seed files can be copied in
FROM golang:1.26-bookworm AS build
WORKDIR /src/api
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ ./
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/simpan ./cmd/simpan \
 && mkdir -p /out/media

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/simpan /simpan
COPY --from=build --chown=nonroot:nonroot /out/media /var/lib/simpan/media
COPY data/seed /app/seed
COPY deploy/dev-seed /app/dev-seed
ENV STORAGE_FS_ROOT=/var/lib/simpan/media SEED_DIR=/app/seed
USER nonroot
EXPOSE 8090 9091
ENTRYPOINT ["/simpan"]
CMD ["serve"]
