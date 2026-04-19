"""Notification relay: stores FCM tokens per user and forwards wormhole codes via FCM."""

import os
import sqlite3
import threading
import time
import uuid
from contextlib import contextmanager

import firebase_admin
from firebase_admin import credentials, messaging
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

app = FastAPI(title="wormhole-relay")

# ── Firebase Admin ────────────────────────────────────────────────────────────
_cred_path = os.environ.get("GOOGLE_APPLICATION_CREDENTIALS", "serviceAccount.json")
firebase_admin.initialize_app(credentials.Certificate(_cred_path))

# ── SQLite (thread-safe via lock) ─────────────────────────────────────────────
_DB = os.environ.get("DB_PATH", "devices.db")
_lock = threading.Lock()


def _conn():
    c = sqlite3.connect(_DB)
    c.row_factory = sqlite3.Row
    return c


def _init_db():
    with _conn() as c:
        c.executescript("""
            CREATE TABLE IF NOT EXISTS devices (
                uid        TEXT NOT NULL,
                device_id  TEXT NOT NULL,
                fcm_token  TEXT NOT NULL,
                platform   TEXT NOT NULL DEFAULT 'android',
                updated_at INTEGER NOT NULL DEFAULT 0,
                PRIMARY KEY (uid, device_id)
            );
            CREATE TABLE IF NOT EXISTS pending_codes (
                id           TEXT PRIMARY KEY,
                uid          TEXT NOT NULL,
                device_id    TEXT NOT NULL,
                code         TEXT NOT NULL,
                filename     TEXT NOT NULL,
                created_at   INTEGER NOT NULL,
                acked        INTEGER NOT NULL DEFAULT 0
            );
        """)


_init_db()


# ── Models ────────────────────────────────────────────────────────────────────
class RegisterRequest(BaseModel):
    uid: str
    device_id: str
    fcm_token: str
    platform: str = "android"   # "android" | "windows"


class NotifyRequest(BaseModel):
    uid: str
    sender_device_id: str
    code: str
    filename: str


class AckRequest(BaseModel):
    uid: str
    device_id: str
    code_id: str


# ── Helpers ───────────────────────────────────────────────────────────────────
def _ts() -> int:
    return int(time.time())


def _get_other_devices(uid: str, sender_device_id: str):
    with _conn() as c:
        return c.execute(
            "SELECT device_id, fcm_token, platform FROM devices "
            "WHERE uid=? AND device_id!=?",
            (uid, sender_device_id),
        ).fetchall()


# ── Endpoints ─────────────────────────────────────────────────────────────────
@app.post("/register")
def register(req: RegisterRequest):
    with _lock, _conn() as c:
        c.execute(
            "INSERT OR REPLACE INTO devices VALUES (?,?,?,?,?)",
            (req.uid, req.device_id, req.fcm_token, req.platform, _ts()),
        )
    return {"status": "ok"}


@app.post("/notify")
def notify(req: NotifyRequest):
    """Store pending code for each target device; send FCM to Android targets."""
    devices = _get_other_devices(req.uid, req.sender_device_id)
    if not devices:
        return {"queued": 0, "fcm_sent": 0}

    android_tokens = []
    code_ids = {}

    with _lock, _conn() as c:
        for dev in devices:
            cid = str(uuid.uuid4())
            code_ids[dev["device_id"]] = cid
            c.execute(
                "INSERT INTO pending_codes VALUES (?,?,?,?,?,?,0)",
                (cid, req.uid, dev["device_id"], req.code, req.filename, _ts()),
            )
            if dev["platform"] == "android":
                android_tokens.append(dev["fcm_token"])

    fcm_sent = 0
    if android_tokens:
        msg = messaging.MulticastMessage(
            data={
                "code": req.code,
                "filename": req.filename,
                "code_id": code_ids.get(devices[0]["device_id"], ""),
            },
            notification=messaging.Notification(
                title="📥 Входящий файл",
                body=req.filename,
            ),
            android=messaging.AndroidConfig(priority="high"),
            tokens=android_tokens,
        )
        resp = messaging.send_each_for_multicast(msg)
        fcm_sent = resp.success_count

    return {"queued": len(devices), "fcm_sent": fcm_sent}


@app.get("/poll/{uid}/{device_id}")
def poll(uid: str, device_id: str):
    """Windows app calls this every 30 s to pick up pending codes."""
    with _conn() as c:
        rows = c.execute(
            "SELECT id, code, filename FROM pending_codes "
            "WHERE uid=? AND device_id=? AND acked=0 "
            "ORDER BY created_at",
            (uid, device_id),
        ).fetchall()
    return {"codes": [dict(r) for r in rows]}


@app.post("/ack")
def ack(req: AckRequest):
    """Mark a pending code as handled (file received or declined)."""
    with _lock, _conn() as c:
        c.execute(
            "UPDATE pending_codes SET acked=1 "
            "WHERE id=? AND uid=? AND device_id=?",
            (req.code_id, req.uid, req.device_id),
        )
    return {"status": "ok"}


@app.delete("/cleanup")
def cleanup():
    """Remove codes older than 24 h (call from a cron job)."""
    cutoff = _ts() - 86400
    with _lock, _conn() as c:
        c.execute("DELETE FROM pending_codes WHERE created_at<?", (cutoff,))
    return {"status": "ok"}
