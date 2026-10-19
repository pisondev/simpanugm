#!/usr/bin/env python3
"""
Load test script for SimpanUGM API endpoints.
Tests public listing, login, and claims list according to docs/engineering/testing-strategy.md
and verifies performance against NFR-PERF-01.
"""

import argparse
import concurrent.futures
import http.cookiejar
import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request
from typing import Dict, List, Optional, Tuple


def auto_detect_member_email() -> Optional[str]:
    """Attempt to find an active member email from the local database."""
    cmd = [
        "docker", "compose",
        "-f", "deploy/compose.yaml",
        "-f", "deploy/compose.dev.yaml",
        "--env-file", "deploy/env/compose.env",
        "exec", "-T", "postgres",
        "psql", "-U", "simpan_app", "-d", "simpan", "-t", "-A",
        "-c", "SELECT email FROM users WHERE role = 'member' AND status = 'active' ORDER BY created_at DESC LIMIT 1;",
    ]
    try:
        res = subprocess.run(cmd, capture_output=True, text=True, timeout=5)
        if res.returncode == 0:
            email = res.stdout.strip()
            if email:
                return email
    except Exception:
        pass
    return None


class SessionClient:
    """Manages an HTTP session with cookie handling."""

    def __init__(self, base_url: str, session_cookie: Optional[str] = None):
        self.base_url = base_url.rstrip("/")
        self.session_cookie = session_cookie
        self.jar = http.cookiejar.CookieJar()
        self.opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(self.jar))

    def request(
        self, method: str, path: str, body: Optional[dict] = None, headers: Optional[dict] = None
    ) -> Tuple[int, float, Optional[str]]:
        url = f"{self.base_url}{path}"
        data = None
        req_headers = {"User-Agent": "SimpanUGM-LoadTest/1.0"}
        if self.session_cookie:
            req_headers["Cookie"] = f"simpan_session={self.session_cookie}"
        if headers:
            req_headers.update(headers)

        if body is not None:
            data = json.dumps(body).encode("utf-8")
            req_headers["Content-Type"] = "application/json"

        req = urllib.request.Request(url, data=data, headers=req_headers, method=method)
        start = time.perf_counter()
        try:
            with self.opener.open(req, timeout=15) as resp:
                elapsed = (time.perf_counter() - start) * 1000.0
                return resp.status, elapsed, None
        except urllib.error.HTTPError as e:
            elapsed = (time.perf_counter() - start) * 1000.0
            return e.code, elapsed, str(e)
        except Exception as e:
            elapsed = (time.perf_counter() - start) * 1000.0
            return 0, elapsed, str(e)

    def login(self, email: str, password: str) -> Optional[str]:
        status, _, _ = self.request("POST", "/api/v1/auth/login", {"email": email, "password": password})
        if status == 200:
            for c in self.jar:
                if c.name == "simpan_session":
                    self.session_cookie = c.value
                    return c.value
        return None


def percentile(data: List[float], p: float) -> float:
    if not data:
        return 0.0
    sorted_data = sorted(data)
    idx = int(len(sorted_data) * (p / 100.0))
    if idx >= len(sorted_data):
        idx = len(sorted_data) - 1
    return sorted_data[idx]


class ScenarioResult:
    def __init__(self, name: str, budget_p95_ms: float):
        self.name = name
        self.budget_p95_ms = budget_p95_ms
        self.latencies: List[float] = []
        self.status_codes: Dict[int, int] = {}
        self.errors: int = 0
        self.total_duration_sec: float = 0.0

    def record(self, status: int, latency_ms: float, err: Optional[str]):
        self.latencies.append(latency_ms)
        self.status_codes[status] = self.status_codes.get(status, 0) + 1
        if status < 200 or status >= 400:
            self.errors += 1

    def print_summary(self) -> bool:
        total = len(self.latencies)
        if total == 0:
            print(f"[{self.name}] No requests were executed.")
            return False

        p50 = percentile(self.latencies, 50)
        p90 = percentile(self.latencies, 90)
        p95 = percentile(self.latencies, 95)
        p99 = percentile(self.latencies, 99)
        min_lat = min(self.latencies)
        max_lat = max(self.latencies)
        avg_lat = sum(self.latencies) / total
        rps = total / self.total_duration_sec if self.total_duration_sec > 0 else 0.0
        passed = p95 <= self.budget_p95_ms and self.errors == 0

        print(f"\n--- Scenario: {self.name} ---")
        print(f"Total Requests: {total}")
        print(f"Duration:       {self.total_duration_sec:.2f} s")
        print(f"Throughput:     {rps:.2f} req/s")
        print(f"Errors:         {self.errors} ({(self.errors / total * 100.0):.1f}%)")
        print(f"Status codes:   {dict(sorted(self.status_codes.items()))}")
        print(f"Latencies (ms): min={min_lat:.1f}, p50={p50:.1f}, p90={p90:.1f}, p95={p95:.1f}, p99={p99:.1f}, max={max_lat:.1f}, avg={avg_lat:.1f}")
        print(f"NFR Budget:     p95 target <= {self.budget_p95_ms:.0f} ms -> {'PASS' if passed else 'FAIL'}")
        return passed


def run_worker(
    scenario: str,
    base_url: str,
    email: str,
    password: str,
    session_cookie: Optional[str],
    duration_sec: float,
    delay_sec: float,
    login_rounds: int,
    result: ScenarioResult,
    stop_event: List[bool],
):
    client = SessionClient(base_url, session_cookie=session_cookie)

    if scenario == "login":
        for _ in range(login_rounds):
            if stop_event[0]:
                break
            login_client = SessionClient(base_url)
            status, lat, err = login_client.request("POST", "/api/v1/auth/login", {"email": email, "password": password})
            result.record(status, lat, err)
            if delay_sec > 0:
                time.sleep(delay_sec)
        return

    start_time = time.time()
    while not stop_event[0]:
        if duration_sec > 0 and (time.time() - start_time) >= duration_sec:
            break

        if scenario == "public":
            status, lat, err = client.request("GET", "/api/v1/found-reports/public?limit=30")
        elif scenario == "claims":
            status, lat, err = client.request("GET", "/api/v1/claims")
        else:
            raise ValueError(f"Unknown scenario: {scenario}")

        result.record(status, lat, err)
        if delay_sec > 0:
            time.sleep(delay_sec)


def run_scenario(
    name: str,
    scenario_type: str,
    base_url: str,
    email: str,
    password: str,
    concurrency: int,
    duration_sec: float,
    delay_sec: float,
    login_rounds: int,
    budget_p95_ms: float,
    session_cookie: Optional[str] = None,
) -> bool:
    print(f"Starting {name} (concurrency={concurrency}, duration={duration_sec}s, delay={delay_sec}s)...")
    result = ScenarioResult(name, budget_p95_ms)
    stop_event = [False]

    start_time = time.perf_counter()
    with concurrent.futures.ThreadPoolExecutor(max_workers=concurrency) as executor:
        futures = [
            executor.submit(
                run_worker,
                scenario_type,
                base_url,
                email,
                password,
                session_cookie,
                duration_sec,
                delay_sec,
                login_rounds,
                result,
                stop_event,
            )
            for _ in range(concurrency)
        ]
        try:
            if scenario_type == "login":
                concurrent.futures.wait(futures)
            else:
                time.sleep(duration_sec)
        finally:
            stop_event[0] = True
            concurrent.futures.wait(futures)

    result.total_duration_sec = time.perf_counter() - start_time
    return result.print_summary()


def main():
    parser = argparse.ArgumentParser(description="SimpanUGM Load Test Runner")
    parser.add_argument("--base-url", default=os.getenv("SIMPAN_BASE_URL", "http://localhost:8000"), help="Base URL of application")
    parser.add_argument("--email", default=os.getenv("LOAD_TEST_EMAIL", ""), help="User email for login / authenticated requests")
    parser.add_argument("--password", default=os.getenv("LOAD_TEST_PASSWORD", "kata sandi panjang 2026"), help="User password")
    parser.add_argument("--concurrency", type=int, default=3, help="Concurrent virtual users (default: 3)")
    parser.add_argument("--duration", type=float, default=4.0, help="Duration in seconds per scenario (default: 4.0)")
    parser.add_argument("--delay", type=float, default=0.12, help="Delay in seconds between requests per worker (default: 0.12)")
    parser.add_argument("--scenario", choices=["all", "public", "login", "claims"], default="all", help="Scenario to run")
    args = parser.parse_args()

    email = args.email
    if not email:
        detected = auto_detect_member_email()
        if detected:
            email = detected
            print(f"Auto-detected member account: {email}")
        else:
            print("Error: No test account found. Provide --email and --password.", file=sys.stderr)
            sys.exit(1)

    print(f"Testing against {args.base_url} with user {email}")

    session_cookie = None
    if args.scenario in ("all", "public", "claims"):
        auth_client = SessionClient(args.base_url)
        session_cookie = auth_client.login(email, args.password)
        if not session_cookie:
            print(f"Error: Failed to obtain session cookie for {email}", file=sys.stderr)
            sys.exit(1)

    all_passed = True

    if args.scenario in ("all", "public"):
        # NFR-PERF-01: API read endpoints p95 at most 300 ms
        ok = run_scenario("Public Found Reports Listing", "public", args.base_url, email, args.password, args.concurrency, args.duration, args.delay, 1, 300.0, session_cookie=session_cookie)
        all_passed = all_passed and ok

    if args.scenario in ("all", "login"):
        # NFR-PERF-01: write endpoints p95 at most 500 ms
        # 1 login per virtual user to stay within BR-SEC-01 budget (5 attempts/15m)
        ok = run_scenario("Authentication Login", "login", args.base_url, email, args.password, args.concurrency, args.duration, 0.2, 1, 500.0)
        all_passed = all_passed and ok

    if args.scenario in ("all", "claims"):
        # NFR-PERF-01: API read endpoints p95 at most 300 ms
        ok = run_scenario("Claims Listing", "claims", args.base_url, email, args.password, args.concurrency, args.duration, args.delay, 1, 300.0, session_cookie=session_cookie)
        all_passed = all_passed and ok

    if all_passed:
        print("\nAll load test scenarios PASSED performance budget targets.")
        sys.exit(0)
    else:
        print("\nSome load test scenarios FAILED budget targets or encountered errors.")
        sys.exit(1)


if __name__ == "__main__":
    main()
