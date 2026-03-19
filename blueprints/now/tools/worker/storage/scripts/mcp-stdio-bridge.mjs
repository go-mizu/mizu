#!/usr/bin/env node
/**
 * MCP stdio-to-HTTP bridge for storage.now
 *
 * Reads JSON-RPC messages from stdin (Content-Length framing),
 * forwards to remote MCP endpoint with Bearer auth, writes responses to stdout.
 *
 * Environment:
 *   STORAGE_API_TOKEN  — Bearer token (required)
 *   STORAGE_MCP_URL    — MCP endpoint (default: https://storage.liteio.dev/mcp)
 */

const MCP_URL = process.env.STORAGE_MCP_URL || "https://storage.liteio.dev/mcp";
const TOKEN = process.env.STORAGE_API_TOKEN;

if (!TOKEN) {
  process.stderr.write("STORAGE_API_TOKEN environment variable is not set\n");
  process.exit(1);
}

let buffer = "";
let contentLength = -1;

process.stdin.setEncoding("utf-8");
process.stdin.on("data", (chunk) => {
  buffer += chunk;
  processBuffer();
});
process.stdin.on("end", () => process.exit(0));

function processBuffer() {
  while (true) {
    if (contentLength === -1) {
      const headerEnd = buffer.indexOf("\r\n\r\n");
      if (headerEnd === -1) return;
      const headerBlock = buffer.slice(0, headerEnd);
      buffer = buffer.slice(headerEnd + 4);
      const match = headerBlock.match(/Content-Length:\s*(\d+)/i);
      if (!match) continue;
      contentLength = parseInt(match[1], 10);
    }
    if (buffer.length < contentLength) return;
    const body = buffer.slice(0, contentLength);
    buffer = buffer.slice(contentLength);
    contentLength = -1;
    handleMessage(body);
  }
}

async function handleMessage(body) {
  let req;
  try {
    req = JSON.parse(body);
  } catch {
    send({ jsonrpc: "2.0", id: null, error: { code: -32700, message: "Parse error" } });
    return;
  }

  process.stderr.write(`→ ${req.method} (id=${req.id ?? "notification"})\n`);

  try {
    const res = await fetch(MCP_URL, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${TOKEN}`,
      },
      body: JSON.stringify(req),
    });

    // 204 No Content = notification ack, no response needed
    if (res.status === 204) {
      process.stderr.write(`← 204 (no content)\n`);
      return;
    }

    const json = await res.json();
    process.stderr.write(`← ${res.status} ok\n`);

    // Only send response for requests (has id), not notifications
    if (req.id != null) {
      send(json);
    }
  } catch (err) {
    process.stderr.write(`← error: ${err.message}\n`);
    if (req.id != null) {
      send({
        jsonrpc: "2.0",
        id: req.id,
        error: { code: -32603, message: `Bridge error: ${err.message}` },
      });
    }
  }
}

function send(obj) {
  const msg = JSON.stringify(obj);
  const len = Buffer.byteLength(msg);
  process.stdout.write(`Content-Length: ${len}\r\n\r\n${msg}`);
}
