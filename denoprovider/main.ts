#!/usr/bin/env -S deno run --allow-env --allow-read --allow-write --allow-net --no-prompt --unstable-sloppy-imports

import { type FlagOptions, parseFlags, ValidationError } from "@cliffy/flags";
import { nanoid } from "nanoid";
import { WasimoffProvider } from "@wasimoff/worker/provider.ts";
import { MemoryFileSystem } from "@wasimoff/storage/fs_memory.ts";
import { DenoFileSystem } from "./fs_denonative.ts";
import { Terminator } from "./util.ts";

const flags: (FlagOptions & { help?: string })[] = [{
  name: "help",
  type: "boolean",
  aliases: ["h"],
  optionalValue: true,
}, {
  name: "workers",
  type: "integer",
  aliases: ["w"],
  default: navigator.hardwareConcurrency,
  help: "Number of Worker threads to spawn",
  value(v: number) {
    if (Number.isNaN(v) || v < 1) {
      throw new ValidationError("Number of Workers must be positive.");
    }
    return v;
  },
}, {
  name: "url",
  type: "string",
  aliases: ["u"],
  default: "http://localhost:4080",
  help: "URL to the Broker",
  value(v: string) {
    if (!/^https?:\/\//.test(v)) {
      throw new ValidationError("Broker URL must be a http(s):// scheme");
    }
    return v;
  },
}, {
  name: "storage",
  type: "string",
  aliases: ["d"],
  default: undefined,
  help: "Path to storage directory",
}];

function manual() {
  console.log(`$ ${import.meta.filename?.replace(/.*\//, "")} ...`);
  for (const flag of flags.filter((f) => "help" in f)) {
    const f = `--${flag.name}/-${flag.aliases![0]} <${flag.type}>`;
    console.log(` ${f.padEnd(25, " ")}  ${flag.help} [${flag.default}]`);
  }
}

let args: ReturnType<typeof parseFlags>["flags"];
try {
  args = parseFlags(Deno.args, { flags }).flags;
} catch (err: unknown) {
  console.error(String(err));
  manual();
  Deno.exit(1);
}
if (args.help) {
  manual();
  Deno.exit(0);
}
console.log(args);

// get random client ID from localStorage
let id: string | null;
try {
  id = localStorage.getItem("wasimoff_id");
  if (id === null) {
    id = nanoid();
    localStorage.setItem("wasimoff_id", id);
  }
} catch {
  id = nanoid();
}

// initialize the provider
console.log("%c[Wasimoff]", "color: red;", "starting Deno Provider");
const fs = args.storage !== undefined
  ? await DenoFileSystem.open(args.storage)
  : new MemoryFileSystem();
const provider = await WasimoffProvider.init(args.workers, args.url, fs, id);
const workers = await provider.pool.scale();
await provider.sendConcurrency(workers);

// log received messages
(async () => {
  for await (const event of provider.messenger!.events) {
    const typename: string = event.$typeName;
    // @ts-ignore event messages are only logged, it's fine
    delete event.$typeName;
    if (!typename.endsWith("Throughput")) {
      console.log(`%c[${typename}]`, "color: green;", JSON.stringify(event));
    }
  }
})();

// register signal handler for clean exits
new Terminator(provider.pool, 30_000, (_) => provider.disconnect());

// start handling requests
await provider.handlerequests();

console.error("ERROR: rpc loop exited, connection lost?");
Deno.exit(1);
