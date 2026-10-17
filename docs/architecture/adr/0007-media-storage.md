# ADR-0007 Storage interface, filesystem by default, S3 compatible as an option

Status: Accepted, 2026-10-17

## Context

Photos and evidence, a few hundred kilobytes per variant, never public, always served
through an authorisation check. Deployment target is one VPS. A self hosted object store
is the common default, but MinIO stopped publishing community binaries and container
images in late 2025 and put the community edition into maintenance mode, which makes it a
poor base for a new project.

## Decision

A `storage.Store` interface in Go with `Put`, `Get`, `Delete`, `Stat`. Two drivers:
`fs`, writing under a mounted volume, and `s3`, using `aws-sdk-go-v2` against any S3
compatible endpoint (Cloudflare R2, AWS S3, Garage). The driver is selected by
`STORAGE_DRIVER`. Production on the VPS uses `fs`.

## Consequences

- No extra container for the default deployment. Backups of the volume use restic, the
  same tool as the database dumps.
- Moving to managed storage is a configuration change plus a one time copy.
- Media is always streamed through `api`, so presigned URLs are not used. At our sizes and
  volumes that costs little and keeps authorisation in one place.
- Both drivers run the same contract test suite.

## Alternatives

- **MinIO.** See context.
- **Storing images in PostgreSQL as `bytea`.** Simple, but it bloats the database and its
  backups and makes retention slower.
