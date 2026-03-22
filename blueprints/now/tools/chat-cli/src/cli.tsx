import { Command } from "commander";
import {
  resolveConfig,
  saveConfig,
  defaultConfigPath,
  goCliConfigPath,
  importGoConfig,
  type Config,
} from "./auth/config.js";
import {
  generateKeypair,
  base64url,
  base64urlDecode,
  signRequest,
  fingerprintAsync,
} from "./auth/signer.js";
import { ChatClient } from "./api/client.js";

const program = new Command()
  .name("chat-now")
  .description("TUI + CLI for chat.go-mizu.workers.dev")
  .version("0.1.0")
  .option("--server <url>", "Server URL", "https://chat.go-mizu.workers.dev")
  .option("--config <path>", "Config file path")
  .option("--pretty", "Pretty-print JSON output");

function output(data: unknown, pretty: boolean) {
  console.log(pretty ? JSON.stringify(data, null, 2) : JSON.stringify(data));
}

async function getConfigOrDie(opts: { config?: string }): Promise<Config> {
  const cfg = await resolveConfig(opts.config);
  if (!cfg) {
    console.error('No identity found. Run "chat-now init" first.');
    process.exit(1);
    throw new Error("unreachable");
  }
  return cfg;
}

function makeClient(cfg: Config, serverOverride?: string): ChatClient {
  const config = serverOverride ? { ...cfg, server: serverOverride } : cfg;
  const signer = (method: string, path: string, query: string, body: string) =>
    signRequest({
      actor: config.actor,
      privateKey: base64urlDecode(config.private_key),
      method,
      path,
      query,
      body,
    });
  return new ChatClient(config, signer);
}

// init
program
  .command("init")
  .description("Generate keypair or import from Go CLI")
  .option("--actor <name>", "Actor name (u/alice or a/bot1)")
  .option("--import", "Import from Go CLI config")
  .action(async (opts) => {
    const parentOpts = program.opts();
    const configPath = parentOpts.config || defaultConfigPath();

    if (opts.import) {
      const goConfig = await importGoConfig(goCliConfigPath());
      if (!goConfig) {
        console.error("Go CLI config not found at", goCliConfigPath());
        process.exit(1);
        return;
      }
      await saveConfig(configPath, goConfig);
      console.log(`Imported identity: ${goConfig.actor}`);
      console.log(`Fingerprint: ${goConfig.fingerprint}`);
      console.log(`Config: ${configPath}`);
      return;
    }

    const actor = opts.actor;
    if (!actor) {
      console.error("--actor is required (e.g. u/alice)");
      process.exit(1);
    }
    if (!/^[ua]\/[\w.@-]{1,64}$/.test(actor)) {
      console.error("Invalid actor format. Use u/<name> or a/<name>");
      process.exit(1);
    }

    const { publicKey, privateKey } = await generateKeypair();
    const fp = await fingerprintAsync(publicKey);
    const server = parentOpts.server;

    const config: Config = {
      actor,
      public_key: base64url(publicKey),
      private_key: base64url(privateKey),
      fingerprint: fp,
      server,
    };

    // Register with server
    const client = makeClient(config, server);
    try {
      const res = await client.register(actor, publicKey);
      console.log(`Registered: ${res.actor}`);
      console.log(`Recovery code: ${res.recovery_code}`);
      console.log("Save your recovery code — it cannot be retrieved later.");
    } catch (e: unknown) {
      console.error(`Registration failed: ${e instanceof Error ? e.message : e}`);
      process.exit(1);
    }

    await saveConfig(configPath, config);
    console.log(`Fingerprint: ${fp}`);
    console.log(`Config: ${configPath}`);
  });

// whoami
program
  .command("whoami")
  .description("Show current identity")
  .action(async () => {
    const opts = program.opts();
    const cfg = await getConfigOrDie(opts);
    output(
      { actor: cfg.actor, fingerprint: cfg.fingerprint, server: cfg.server },
      !!opts.pretty,
    );
  });

// create
program
  .command("create")
  .description("Create a room")
  .option("--title <title>", "Room title")
  .option("--visibility <v>", "public or private", "public")
  .action(async (opts) => {
    const parentOpts = program.opts();
    const cfg = await getConfigOrDie(parentOpts);
    const client = makeClient(cfg, parentOpts.server);
    try {
      const chat = await client.createChat({
        title: opts.title,
        visibility: opts.visibility,
      });
      output(chat, !!parentOpts.pretty);
    } catch (e: unknown) {
      console.error(e instanceof Error ? e.message : e);
      process.exit(1);
    }
  });

// dm
program
  .command("dm <peer>")
  .description("Start or resume DM with peer")
  .action(async (peer) => {
    const opts = program.opts();
    const cfg = await getConfigOrDie(opts);
    const client = makeClient(cfg, opts.server);
    try {
      const chat = await client.startDm(peer);
      output(chat, !!opts.pretty);
    } catch (e: unknown) {
      console.error(e instanceof Error ? e.message : e);
      process.exit(1);
    }
  });

// join
program
  .command("join <id>")
  .description("Join a chat")
  .action(async (id) => {
    const opts = program.opts();
    const cfg = await getConfigOrDie(opts);
    const client = makeClient(cfg, opts.server);
    try {
      await client.joinChat(id);
    } catch (e: unknown) {
      console.error(e instanceof Error ? e.message : e);
      process.exit(1);
    }
  });

// get
program
  .command("get <id>")
  .description("Get chat details")
  .action(async (id) => {
    const opts = program.opts();
    const cfg = await getConfigOrDie(opts);
    const client = makeClient(cfg, opts.server);
    try {
      const chat = await client.getChat(id);
      output(chat, !!opts.pretty);
    } catch (e: unknown) {
      console.error(e instanceof Error ? e.message : e);
      process.exit(1);
    }
  });

// list
program
  .command("list")
  .description("List chats")
  .option("--limit <n>", "Limit results", "50")
  .action(async (opts) => {
    const parentOpts = program.opts();
    const cfg = await getConfigOrDie(parentOpts);
    const client = makeClient(cfg, parentOpts.server);
    try {
      const chats = await client.listChats({ limit: parseInt(opts.limit) });
      output(chats, !!parentOpts.pretty);
    } catch (e: unknown) {
      console.error(e instanceof Error ? e.message : e);
      process.exit(1);
    }
  });

// dms
program
  .command("dms")
  .description("List DM conversations")
  .option("--limit <n>", "Limit results", "50")
  .action(async (opts) => {
    const parentOpts = program.opts();
    const cfg = await getConfigOrDie(parentOpts);
    const client = makeClient(cfg, parentOpts.server);
    try {
      const dms = await client.listDms({ limit: parseInt(opts.limit) });
      output(dms, !!parentOpts.pretty);
    } catch (e: unknown) {
      console.error(e instanceof Error ? e.message : e);
      process.exit(1);
    }
  });

// send
program
  .command("send <id> <text>")
  .description("Send a message")
  .action(async (id, text) => {
    const opts = program.opts();
    const cfg = await getConfigOrDie(opts);
    const client = makeClient(cfg, opts.server);
    try {
      const msg = await client.sendMessage(id, text);
      output(msg, !!opts.pretty);
    } catch (e: unknown) {
      console.error(e instanceof Error ? e.message : e);
      process.exit(1);
    }
  });

// messages
program
  .command("messages <id>")
  .description("List messages in a chat")
  .option("--limit <n>", "Limit results", "50")
  .option("--before <id>", "Cursor for pagination")
  .action(async (id, opts) => {
    const parentOpts = program.opts();
    const cfg = await getConfigOrDie(parentOpts);
    const client = makeClient(cfg, parentOpts.server);
    try {
      const msgs = await client.listMessages(id, {
        limit: parseInt(opts.limit),
        before: opts.before,
      });
      output(msgs, !!parentOpts.pretty);
    } catch (e: unknown) {
      console.error(e instanceof Error ? e.message : e);
      process.exit(1);
    }
  });

// Default: launch TUI
program.action(async () => {
  const opts = program.opts();
  const cfg = await getConfigOrDie(opts);
  const { launchTui } = await import("./tui/App.js");
  await launchTui(cfg, opts.server);
});

program.parseAsync();
