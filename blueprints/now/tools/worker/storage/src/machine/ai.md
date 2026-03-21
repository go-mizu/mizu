# Storage for AI

> Connect Claude and ChatGPT to your files. Share between AIs and with anyone.

Storage is an MCP-native file storage service. Connect once, then save, find, and share files by asking your AI in natural language.

## Available MCP Tools

| Tool | Description | Example Prompt |
|------|-------------|----------------|
| `storage_read` | Read a file's contents | "Show me config.json" |
| `storage_write` | Create or overwrite a file | "Save this as report.md" |
| `storage_list` | List files in a folder | "What files do I have in docs/?" |
| `storage_search` | Search files by name | "Find all CSV files" |
| `storage_share` | Create a temporary public link | "Share the report with my team" |
| `storage_move` | Move or rename a file | "Move it to /work" |
| `storage_delete` | Delete a file | "Delete the old draft" |
| `storage_stats` | Show storage usage | "How much space am I using?" |

## Connect in 30 Seconds

### Claude.ai

1. Open **Settings > Integrations**
2. Click **Add custom connector**
3. Enter URL: `https://storage.liteio.dev/mcp`
4. Click Add, verify your email — done

### ChatGPT

1. Open **Settings > Connected apps**
2. Click **Add app > Add by URL**
3. Enter URL: `https://storage.liteio.dev/mcp`
4. Sign in with email — done

### Claude Desktop

Add to your `claude_desktop_config.json` (Settings > Developer > Edit Config):

```json
{
  "mcpServers": {
    "storage": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "https://storage.liteio.dev/mcp"]
    }
  }
}
```

Restart Claude Desktop after saving.

## Example: Save and Share

```
You:    "Save the meeting notes as notes/2025-03-20.md"
Claude: Done! Saved notes/2025-03-20.md (4.2 KB)

You:    "What files do I have in notes/?"
Claude: Your notes/ folder has 3 files:
        • 2025-03-20.md  4.2 KB
        • 2025-03-18.md  2.1 KB
        • ideas.txt       890 B

You:    "Share the latest one"
Claude: Here's your link: storage.liteio.dev/s/m9x2k
        Expires in 24 hours.
```

The recipient opens the link in their browser — no account needed.

## Cross-Platform: ChatGPT + Claude

Both AIs connect to the same storage. Save a file from ChatGPT, find it in Claude.

```
ChatGPT: "Save this report as Q1-results.pdf"
         → File saved to your storage.

Claude:  "What files do I have?"
         → Shows Q1-results.pdf (saved from ChatGPT)

Claude:  "Share Q1-results.pdf with my team"
         → Creates a public link anyone can open.
```

Your files follow you across AI platforms.

## How It Works

1. You connect your AI to Storage via MCP (one-time setup)
2. When you ask your AI to manage files, it calls Storage tools
3. Files are stored in cloud object storage (R2) — globally distributed, zero egress
4. Share links are time-limited and accessible without authentication

## Links

- [Developer Guide](https://storage.liteio.dev/developers) — API docs, code examples
- [CLI](https://storage.liteio.dev/cli) — Terminal interface
- [API Reference](https://storage.liteio.dev/api) — Full endpoint documentation
- [Pricing](https://storage.liteio.dev/pricing) — Free tier and plans
