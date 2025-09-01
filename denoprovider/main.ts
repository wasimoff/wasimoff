#!/usr/bin/env -S deno run --allow-env --allow-read --allow-write --allow-net --no-prompt --unstable-sloppy-imports

import { parseArgs } from "@std/cli/parse-args";
import { nanoid } from "nanoid";
import { WasimoffProvider } from "@wasimoff/worker/provider.ts";
import { Terminator } from "./util.ts";

// parse commandline arguments
const help = (fatal: boolean = false) => {
  console.log(
    "$",
    import.meta.filename?.replace(/.*\//, ""),
    "[--workers n] [--url <Broker URL>]",
  );
  Deno.exit(fatal ? 1 : 0);
};
const args = parseArgs(Deno.args, {
  alias: { workers: "w", url: "u", help: "h" },
  default: {
    workers: navigator.hardwareConcurrency,
    url: "http://localhost:4080",
  },
  boolean: ["help"],
  string: ["url"],
  unknown: (arg) => {
    console.warn("Unknown argument:", arg);
    help(true);
  },
});

// print help if requested
if (args.help) help();

// validate the values
const brokerurl = args.url;
if (!/^https?:\/\//.test(brokerurl)) {
  throw "--url must be a HTTP(S) origin (http?://)";
}
const nproc = Math.floor(Number(args.workers));
if (Number.isNaN(nproc) || nproc < 1) {
  throw "--workers must be a positive number";
}

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
const provider = await WasimoffProvider.init(nproc, brokerurl, ":memory:", id);
const workers = await provider.pool.scale();
await provider.sendConcurrency(workers);

// log received messages
(async () => {
  for await (const event of provider.messenger!.events) {
    const typename: string = event.$typeName;
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
