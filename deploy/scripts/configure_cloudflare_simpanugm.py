#!/usr/bin/env python3
"""Manage the single SimpanUGM DNS record without changing zone-wide settings."""

from __future__ import annotations

import json
import os
import ssl
import sys
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import Request, urlopen

DOMAIN = "simpanugm.pisondev.com"
ZONE = "pisondev.com"
ORIGIN = os.environ.get("ORIGIN_IP", "")
API = "https://api.cloudflare.com/client/v4"


def call(path: str, method: str = "GET", payload: dict | None = None) -> object:
    token = os.environ.get("CLOUDFLARE_API_TOKEN", "")
    if not token:
        raise RuntimeError("Set CLOUDFLARE_API_TOKEN in the environment.")
    if not ORIGIN:
        raise RuntimeError("Set ORIGIN_IP to the public address of the server.")
    headers = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}
    body = json.dumps(payload).encode() if payload is not None else None
    request = Request(API + path, data=body, headers=headers, method=method)
    try:
        with urlopen(request, timeout=20) as response:
            result = json.load(response)
    except (HTTPError, URLError) as error:
        raise RuntimeError(f"Cloudflare API request failed: {error}") from error
    if not result.get("success"):
        codes = ", ".join(str(item.get("code", "unknown")) for item in result.get("errors", []))
        raise RuntimeError(f"Cloudflare API rejected the request: {codes or 'unknown error'}")
    return result.get("result")


def zone_id() -> str:
    rows = call("/zones?" + urlencode({"name": ZONE}))
    matches = [row for row in rows if row.get("name") == ZONE and row.get("status") == "active"]
    if len(matches) != 1:
        raise RuntimeError(f"Expected one active Cloudflare zone for {ZONE}.")
    return str(matches[0]["id"])


def main(mode: str) -> None:
    if mode not in {"inspect", "dns", "proxy"}:
        raise RuntimeError("Usage: configure_cloudflare_simpanugm.py inspect|dns|proxy")

    zone = zone_id()
    ssl_mode = call(f"/zones/{zone}/settings/ssl")
    if ssl_mode.get("value") != "strict":
        raise RuntimeError("The shared pisondev.com zone must remain on Full (strict). Do not change it here.")

    query = urlencode({"name": DOMAIN})
    records = call(f"/zones/{zone}/dns_records?{query}")
    if len(records) > 1:
        raise RuntimeError(f"Review the {len(records)} existing records for {DOMAIN} manually.")
    record = records[0] if records else None
    if record and (record.get("type") != "A" or record.get("content") != ORIGIN):
        raise RuntimeError(f"The existing {DOMAIN} record points elsewhere. It was not changed.")

    if mode == "inspect":
        state = "absent" if record is None else f"A {ORIGIN}, proxied={record.get('proxied')}"
        print(f"{DOMAIN}: {state}; zone SSL: Full (strict)")
        return

    if mode == "dns":
        payload = {"type": "A", "name": DOMAIN, "content": ORIGIN, "ttl": 1, "proxied": False}
        if record is None:
            call(f"/zones/{zone}/dns_records", "POST", payload)
        elif record.get("proxied"):
            call(f"/zones/{zone}/dns_records/{record['id']}", "PATCH", {"proxied": False, "ttl": 1})
        print(f"{DOMAIN}: DNS-only A record points to {ORIGIN}")
        return

    if record is None:
        raise RuntimeError("Create the DNS-only record first, then provision and verify Hestia TLS.")
    context = ssl.create_default_context()
    try:
        with urlopen(f"https://{DOMAIN}/healthz", timeout=10, context=context) as response:
            if response.status != 200:
                raise RuntimeError(f"Origin HTTPS health check returned HTTP {response.status}.")
    except (HTTPError, URLError, TimeoutError) as error:
        raise RuntimeError(f"Origin TLS and health check must pass before enabling Cloudflare proxy: {error}") from error
    if not record.get("proxied"):
        call(f"/zones/{zone}/dns_records/{record['id']}", "PATCH", {"proxied": True, "ttl": 1})
    print(f"{DOMAIN}: Cloudflare proxy enabled after origin HTTPS health check")


if __name__ == "__main__":
    try:
        main(sys.argv[1] if len(sys.argv) == 2 else "")
    except (RuntimeError, KeyError, TypeError) as error:
        print(f"error: {error}", file=sys.stderr)
        sys.exit(1)
