#!/usr/bin/env python3
"""End-to-end test suite for chat.now Ed25519 auth."""
import hashlib
import json
import time
import urllib.request
import urllib.error
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from cryptography.hazmat.primitives.serialization import Encoding, PublicFormat, NoEncryption, PrivateFormat
import base64

BASE = "https://chat.go-mizu.workers.dev"

def b64url(data: bytes) -> str:
    return base64.urlsafe_b64encode(data).rstrip(b"=").decode()

def b64url_decode(s: str) -> bytes:
    s += "=" * (4 - len(s) % 4)
    return base64.urlsafe_b64decode(s)

def sha256hex(data: str) -> str:
    return hashlib.sha256(data.encode()).hexdigest()

def sign_request(private_key: Ed25519PrivateKey, actor: str, method: str, path: str, query: str = "", body: str = "") -> dict:
    body_hash = hashlib.sha256(body.encode()).hexdigest()
    canonical = f"{method}\n{path}\n{query}\n{body_hash}"
    canonical_hash = sha256hex(canonical)
    ts = str(int(time.time()))
    string_to_sign = f"CHAT-ED25519\n{ts}\n{actor}\n{canonical_hash}"
    sig = private_key.sign(string_to_sign.encode())
    sig_b64 = b64url(sig)
    return {
        "Authorization": f"CHAT-ED25519 Credential={actor}, Timestamp={ts}, Signature={sig_b64}",
        "Content-Type": "application/json",
    }

UA = "chat-test/1.0"

def api(method: str, path: str, body=None, headers=None):
    url = BASE + path
    data = json.dumps(body).encode() if body else None
    h = dict(headers or {})
    h.setdefault("User-Agent", UA)
    req = urllib.request.Request(url, data=data, method=method, headers=h)
    if body and "Content-Type" not in h:
        req.add_header("Content-Type", "application/json")
    try:
        resp = urllib.request.urlopen(req)
        raw = resp.read()
        try:
            return resp.status, json.loads(raw)
        except:
            return resp.status, {}
    except urllib.error.HTTPError as e:
        try:
            return e.code, json.loads(e.read())
        except:
            return e.code, {}

def signed_api(key, actor, method, path, body=None, query=""):
    body_str = json.dumps(body) if body else ""
    headers = sign_request(key, actor, method, path, query, body_str)
    headers["User-Agent"] = UA
    url = BASE + path + ("?" + query if query else "")
    data = body_str.encode() if body_str else None
    req = urllib.request.Request(url, data=data, method=method, headers=headers)
    try:
        resp = urllib.request.urlopen(req)
        raw = resp.read()
        try:
            return resp.status, json.loads(raw)
        except:
            return resp.status, {}
    except urllib.error.HTTPError as e:
        try:
            return e.code, json.loads(e.read())
        except:
            return e.code, {}

def test(name, condition, detail=""):
    status = "PASS" if condition else "FAIL"
    print(f"  [{status}] {name}" + (f" — {detail}" if detail and not condition else ""))
    return condition

passed = 0
failed = 0

def run_test(name, condition, detail=""):
    global passed, failed
    if test(name, condition, detail):
        passed += 1
    else:
        failed += 1

print("=== Landing & Docs ===")
code1, _ = api("GET", "/")
run_test("Landing page returns 200", code1 == 200, f"got {code1}")
code2, _ = api("GET", "/docs")
run_test("Docs page returns 200", code2 == 200, f"got {code2}")

print("\n=== Registration ===")
key1 = Ed25519PrivateKey.generate()
pub1 = key1.public_key().public_bytes(Encoding.Raw, PublicFormat.Raw)
pub1_b64 = b64url(pub1)

ts_suffix = str(int(time.time()))[-6:]
actor1 = f"u/test_{ts_suffix}"

code, resp = api("POST", "/api/register", {"actor": actor1, "public_key": pub1_b64})
run_test("Register actor", code == 201, f"got {code}: {resp}")
recovery_code = resp.get("recovery_code", "")
run_test("Recovery code returned", len(recovery_code) > 0)

code, resp = api("POST", "/api/register", {"actor": actor1, "public_key": pub1_b64})
run_test("Duplicate registration → 409", code == 409, f"got {code}")

code, resp = api("POST", "/api/register", {"actor": "invalid", "public_key": pub1_b64})
run_test("Invalid actor format → 400", code == 400, f"got {code}")

code, resp = api("POST", "/api/register", {"actor": "u/bad_key", "public_key": "notakey"})
run_test("Invalid public key → 400", code == 400, f"got {code}")

print("\n=== Signature Auth ===")
code, resp = signed_api(key1, actor1, "GET", "/api/chat")
run_test("Signed GET /api/chat succeeds", code == 200, f"got {code}: {resp}")

code, resp = api("GET", "/api/chat", headers={"Authorization": "CHAT-ED25519 Credential=u/nobody, Timestamp=123, Signature=badsig", "User-Agent": UA})
run_test("Invalid signature → 401", code == 401, f"got {code}")

old_ts = str(int(time.time()) - 600)
body_str = ""
body_hash = sha256hex(body_str)
canonical = f"GET\n/api/chat\n\n{body_hash}"
canonical_hash = sha256hex(canonical)
sts = f"CHAT-ED25519\n{old_ts}\n{actor1}\n{canonical_hash}"
sig = key1.sign(sts.encode())
headers = {"Authorization": f"CHAT-ED25519 Credential={actor1}, Timestamp={old_ts}, Signature={b64url(sig)}", "User-Agent": UA}
code, resp = api("GET", "/api/chat", headers=headers)
run_test("Expired timestamp → 401", code == 401, f"got {code}")

code, resp = api("GET", "/api/chat")
run_test("No auth → 401", code == 401, f"got {code}")

print("\n=== Chat Operations ===")
code, resp = signed_api(key1, actor1, "POST", "/api/chat", {"kind": "room", "title": "test-ed25519"})
run_test("Create room", code == 201, f"got {code}: {resp}")
room_id = resp.get("id", "")

code, resp = signed_api(key1, actor1, "GET", f"/api/chat/{room_id}")
run_test("Get room", code == 200, f"got {code}")

code, resp = signed_api(key1, actor1, "POST", f"/api/chat/{room_id}/messages", {"text": "Hello from Ed25519!"})
run_test("Send message", code == 201, f"got {code}")
msg_id = resp.get("id", "")

code, resp = signed_api(key1, actor1, "GET", f"/api/chat/{room_id}/messages")
run_test("List messages", code == 200, f"got {code}")
run_test("Message in list", len(resp.get("items", [])) > 0)

# Second actor
key2 = Ed25519PrivateKey.generate()
pub2 = key2.public_key().public_bytes(Encoding.Raw, PublicFormat.Raw)
actor2 = f"a/bot_{ts_suffix}"
code, resp = api("POST", "/api/register", {"actor": actor2, "public_key": b64url(pub2)})
run_test("Register second actor", code == 201, f"got {code}")

# Non-member can't send
code, resp = signed_api(key2, actor2, "POST", f"/api/chat/{room_id}/messages", {"text": "should fail"})
run_test("Non-member can't send → 403", code == 403, f"got {code}")

# Join and send
code, resp = signed_api(key2, actor2, "POST", f"/api/chat/{room_id}/join")
run_test("Join room", code == 204 or code == 200, f"got {code}")

code, resp = signed_api(key2, actor2, "POST", f"/api/chat/{room_id}/messages", {"text": "Hello from bot!"})
run_test("Member can send after join", code == 201, f"got {code}")

# Body tampering test
body_data = json.dumps({"text": "original"})
tampered_body = json.dumps({"text": "tampered"})
headers = sign_request(key1, actor1, "POST", f"/api/chat/{room_id}/messages", "", body_data)
url = BASE + f"/api/chat/{room_id}/messages"
headers["User-Agent"] = UA
req = urllib.request.Request(url, data=tampered_body.encode(), method="POST", headers=headers)
try:
    resp = urllib.request.urlopen(req)
    code = resp.status
except urllib.error.HTTPError as e:
    code = e.code
run_test("Tampered body → 401", code == 401, f"got {code}")

print("\n=== Private Chat ===")
code, resp = signed_api(key1, actor1, "POST", "/api/chat", {"kind": "room", "title": "secret", "visibility": "private"})
run_test("Create private room", code == 201, f"got {code}")
priv_id = resp.get("id", "")

code, resp = signed_api(key2, actor2, "GET", f"/api/chat/{priv_id}/messages")
run_test("Non-member can't read private → 404", code == 404, f"got {code}")

print("\n=== Direct Messages ===")
# Start DM from actor1 to actor2
code, resp = signed_api(key1, actor1, "POST", "/api/chat/dm", {"peer": actor2})
run_test("Start DM", code == 201, f"got {code}: {resp}")
dm_id = resp.get("id", "")
run_test("DM has peer field", resp.get("peer") == actor2, f"peer={resp.get('peer')}")
run_test("DM is direct kind", resp.get("kind") == "direct")

# Idempotent — same DM returned
code2, resp2 = signed_api(key1, actor1, "POST", "/api/chat/dm", {"peer": actor2})
run_test("DM idempotent (200)", code2 == 200, f"got {code2}")
run_test("DM same id returned", resp2.get("id") == dm_id, f"got {resp2.get('id')} vs {dm_id}")

# Peer can also resume
code3, resp3 = signed_api(key2, actor2, "POST", "/api/chat/dm", {"peer": actor1})
run_test("Peer resume DM (200)", code3 == 200, f"got {code3}")
run_test("Peer sees same DM id", resp3.get("id") == dm_id)
run_test("Peer field shows other actor", resp3.get("peer") == actor1, f"peer={resp3.get('peer')}")

# Both can send messages in DM
code, resp = signed_api(key1, actor1, "POST", f"/api/chat/{dm_id}/messages", {"text": "Hey from alice!"})
run_test("Actor1 send DM message", code == 201, f"got {code}")

code, resp = signed_api(key2, actor2, "POST", f"/api/chat/{dm_id}/messages", {"text": "Hey back!"})
run_test("Actor2 send DM message", code == 201, f"got {code}")

# Both can read
code, resp = signed_api(key1, actor1, "GET", f"/api/chat/{dm_id}/messages")
run_test("Read DM messages", code == 200 and len(resp.get("items", [])) == 2, f"got {code}, items={len(resp.get('items', []))}")

# Non-member can't read DM (it's private)
key_eve = Ed25519PrivateKey.generate()
pub_eve = key_eve.public_key().public_bytes(Encoding.Raw, PublicFormat.Raw)
actor_eve = f"u/eve_{ts_suffix}"
api("POST", "/api/register", {"actor": actor_eve, "public_key": b64url(pub_eve)})
code, resp = signed_api(key_eve, actor_eve, "GET", f"/api/chat/{dm_id}/messages")
run_test("Non-member can't read DM → 404", code == 404, f"got {code}")

# List DMs
code, resp = signed_api(key1, actor1, "GET", "/api/chat/dm")
run_test("List DMs", code == 200, f"got {code}")
dm_ids = [d["id"] for d in resp.get("items", [])]
run_test("DM in list", dm_id in dm_ids)

# DM self → 400
code, resp = signed_api(key1, actor1, "POST", "/api/chat/dm", {"peer": actor1})
run_test("DM self → 400", code == 400, f"got {code}")

# DM non-existent actor → 404
code, resp = signed_api(key1, actor1, "POST", "/api/chat/dm", {"peer": "u/nonexistent_xyz"})
run_test("DM non-existent → 404", code == 404, f"got {code}")

# POST /api/chat with kind: "direct" → 400
code, resp = signed_api(key1, actor1, "POST", "/api/chat", {"kind": "direct", "title": "nope"})
run_test("Create direct via /api/chat → 400", code == 400, f"got {code}")
run_test("Error mentions /api/chat/dm", "dm" in resp.get("error", "").lower(), f"error={resp.get('error')}")

# Join DM → 403
code, resp = signed_api(key_eve, actor_eve, "POST", f"/api/chat/{dm_id}/join")
run_test("Join DM → 403", code == 403, f"got {code}")

print("\n=== Key Rotation ===")
key3 = Ed25519PrivateKey.generate()
pub3 = key3.public_key().public_bytes(Encoding.Raw, PublicFormat.Raw)

code, resp = api("POST", "/api/keys/rotate", {
    "actor": actor1,
    "recovery_code": recovery_code,
    "new_public_key": b64url(pub3),
})
run_test("Rotate key", code == 200, f"got {code}: {resp}")

# New key should work (cache may still have old key in some isolates — retry)
for attempt in range(5):
    code, resp = signed_api(key3, actor1, "GET", "/api/chat")
    if code == 200:
        break
    time.sleep(1)
run_test("New key works after rotation", code == 200, f"got {code}")

# Old key should fail
code, resp = signed_api(key1, actor1, "GET", "/api/chat")
run_test("Old key fails after rotation", code == 401, f"got {code} (may pass with cache TTL)")

print("\n=== Recovery Rotation ===")
code, resp = api("POST", "/api/keys/rotate-recovery", {
    "actor": actor1,
    "recovery_code": recovery_code,
})
run_test("Rotate recovery code", code == 200, f"got {code}")
new_recovery = resp.get("recovery_code", "")
run_test("New recovery code returned", len(new_recovery) > 0)

code, resp = api("POST", "/api/keys/rotate-recovery", {
    "actor": actor1,
    "recovery_code": recovery_code,
})
run_test("Old recovery code fails", code == 401, f"got {code}")

code, resp = api("POST", "/api/keys/rotate-recovery", {
    "actor": actor1,
    "recovery_code": new_recovery,
})
run_test("New recovery code works", code == 200, f"got {code}")
final_recovery = resp.get("recovery_code", "")

print("\n=== Account Deletion ===")
code, resp = api("POST", "/api/actors/delete", {
    "actor": actor2,
    "recovery_code": "wrong_code",
})
run_test("Wrong recovery → 401", code == 401, f"got {code}")

# Get actor2's recovery code (we need it — it was returned at registration but we didn't save it)
# Use actor1 deletion instead since we have its recovery code
code, resp = api("POST", "/api/actors/delete", {
    "actor": actor1,
    "recovery_code": final_recovery,
})
run_test("Delete actor", code == 200, f"got {code}")

for attempt in range(5):
    code, resp = signed_api(key3, actor1, "GET", "/api/chat")
    if code == 401:
        break
    time.sleep(1)
run_test("Deleted actor can't auth", code == 401, f"got {code} (may need cache TTL)")

# Re-register same name
key4 = Ed25519PrivateKey.generate()
pub4 = key4.public_key().public_bytes(Encoding.Raw, PublicFormat.Raw)
code, resp = api("POST", "/api/register", {"actor": actor1, "public_key": b64url(pub4)})
run_test("Re-register deleted name", code == 201, f"got {code}")

print(f"\n{'='*40}")
print(f"Results: {passed} passed, {failed} failed, {passed + failed} total")
if failed > 0:
    exit(1)
