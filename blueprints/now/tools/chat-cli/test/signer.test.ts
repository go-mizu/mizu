import { describe, it } from "node:test";
import assert from "node:assert/strict";
import {
  buildCanonicalRequest,
  buildStringToSign,
  sha256hex,
  base64url,
  base64urlDecode,
  signRequest,
  buildAuthHeader,
  generateKeypair,
} from "../src/auth/signer.ts";

describe("base64url", () => {
  it("encodes without padding", () => {
    const buf = new TextEncoder().encode("test");
    const encoded = base64url(buf);
    assert.ok(!encoded.includes("="));
    assert.ok(!encoded.includes("+"));
    assert.ok(!encoded.includes("/"));
  });

  it("round-trips", () => {
    const original = new Uint8Array([0, 1, 2, 255, 254]);
    const encoded = base64url(original);
    const decoded = base64urlDecode(encoded);
    assert.deepEqual(decoded, original);
  });
});

describe("sha256hex", () => {
  it("hashes empty string correctly", async () => {
    const hash = await sha256hex("");
    assert.equal(hash, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855");
  });

  it("hashes content", async () => {
    const hash = await sha256hex("hello");
    assert.equal(hash, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824");
  });
});

describe("buildCanonicalRequest", () => {
  it("builds GET with no query or body", async () => {
    const cr = await buildCanonicalRequest("GET", "/api/chat", "", "");
    const emptyHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855";
    assert.equal(cr, `GET\n/api/chat\n\n${emptyHash}`);
  });

  it("builds POST with body", async () => {
    const body = '{"text":"hello"}';
    const cr = await buildCanonicalRequest("POST", "/api/chat/c_123/messages", "", body);
    const bodyHash = await sha256hex(body);
    assert.equal(cr, `POST\n/api/chat/c_123/messages\n\n${bodyHash}`);
  });

  it("sorts query params", async () => {
    const cr = await buildCanonicalRequest("GET", "/api/chat", "limit=10&before=m_abc", "");
    const emptyHash = await sha256hex("");
    assert.equal(cr, `GET\n/api/chat\nbefore=m_abc&limit=10\n${emptyHash}`);
  });
});

describe("buildStringToSign", () => {
  it("builds correctly", async () => {
    const sts = await buildStringToSign(1710000000, "u/alice", "canonical-hash-hex");
    assert.equal(sts, "CHAT-ED25519\n1710000000\nu/alice\ncanonical-hash-hex");
  });
});

describe("signRequest", () => {
  it("generates valid keypair and signs", async () => {
    const { publicKey, privateKey } = await generateKeypair();
    assert.equal(publicKey.length, 32);
    assert.equal(privateKey.length, 64);

    const header = await signRequest({
      actor: "u/test",
      privateKey,
      method: "GET",
      path: "/api/chat",
      query: "",
      body: "",
    });
    assert.ok(header.startsWith("CHAT-ED25519 Credential=u/test, Timestamp="));
    assert.ok(header.includes("Signature="));
  });
});

describe("buildAuthHeader", () => {
  it("formats correctly", () => {
    const header = buildAuthHeader("u/alice", 1710000000, "c2lnbmF0dXJl");
    assert.equal(
      header,
      "CHAT-ED25519 Credential=u/alice, Timestamp=1710000000, Signature=c2lnbmF0dXJl",
    );
  });
});
