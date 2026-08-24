# pytest file name kept as api_smoke.py (pytest.ini python_files includes *_smoke.py)
import json
import os
import urllib.parse

import pytest
import requests
from websocket import create_connection

API = os.environ.get("API_URL", "http://127.0.0.1:31822")
TOKEN = os.environ.get("AUTH_TOKEN", "mc-dev-31821")


def auth():
    return {"Authorization": f"Bearer {TOKEN}"}


def test_health():
    r = requests.get(f"{API}/health", timeout=5)
    r.raise_for_status()
    body = r.json()
    assert body["ok"] is True
    assert body["data"]["service"] == "macrocanvas"


def test_login_public_credentials():
    r = requests.post(
        f"{API}/api/v1/auth/login",
        json={"username": "geek", "password": "phosphor"},
        timeout=5,
    )
    r.raise_for_status()
    assert r.json()["data"]["token"] == TOKEN


def test_macros_and_p10_validate_run():
    r = requests.get(f"{API}/api/v1/macros", headers=auth(), timeout=5)
    r.raise_for_status()
    ids = [m["id"] for m in r.json()["data"]]
    assert "sample-p10" in ids
    v = requests.post(f"{API}/api/v1/macros/sample-p10/validate", headers=auth(), timeout=5)
    v.raise_for_status()
    assert v.json()["data"]["ok"] is True
    run = requests.post(f"{API}/api/v1/macros/sample-p10/run", headers=auth(), timeout=15)
    run.raise_for_status()
    data = run.json()["data"]
    assert data["status"] in ("succeeded", "stopped")
    assert data["trace"]["p50_ns"] >= 0


def test_status_mode_switch_field_present():
    r = requests.get(f"{API}/api/v1/status", headers=auth(), timeout=5)
    r.raise_for_status()
    mode = r.json()["data"]["device"]["mode"]
    assert mode in ("auto", "real", "mock")
    assert r.json()["data"]["device"]["source_from_tier"] in (
        "physical",
        "kernel_virtual",
        "simulated",
        "injected",
    )


def test_ws_no_origin_ok():
    url = API.replace("http", "ws") + f"/ws/events?token={urllib.parse.quote(TOKEN)}"
    ws = create_connection(url, timeout=5, suppress_origin=True)
    ws.close()


def test_ws_evil_origin_denied():
    url = API.replace("http", "ws") + f"/ws/events?token={urllib.parse.quote(TOKEN)}"
    with pytest.raises(Exception):
        create_connection(
            url,
            timeout=5,
            origin="http://evil.example",
        )
