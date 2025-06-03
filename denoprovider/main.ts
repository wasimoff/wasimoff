#!/usr/bin/env -S deno run --allow-env --allow-read --allow-write --allow-net --no-prompt --unstable-sloppy-imports

import { parseArgs } from "@std/cli/parse-args";
import { WasimoffProvider } from "@wasimoff/worker/provider.ts";

// parse commandline arguments
const help = (fatal: boolean = false) => {
  console.log("$", import.meta.filename?.replace(/.*\//, ""), "[--workers n] [--url <Broker URL>]");
  Deno.exit(fatal ? 1 : 0);
};
const args = parseArgs(Deno.args, {
  alias: { "workers": "w", "url": "u", "help": "h" },
  default: {
    "workers": navigator.hardwareConcurrency,
    "url": "http://localhost:4080",
  },
  boolean: [ "help" ],
  string: [ "url" ],
  unknown: (arg) => { console.warn("Unknown argument:", arg); help(true); }
});

// print help if requested
if (args.help) help();

// validate the values
const brokerurl = args.url;
if (!/^https?:\/\//.test(brokerurl)) throw "--url must be a HTTP(S) origin (http?://)";
const nproc = Math.floor(Number(args.workers));
if (Number.isNaN(nproc) || nproc < 1) throw "--workers must be a positive number";

// initialize the provider
console.log("%c[Wasimoff]", "color: red;", "starting Provider in Deno ...");
const provider = await WasimoffProvider.init(nproc, brokerurl, ":memory:");
const workers = await provider.pool.scale();
await provider.sendInfo(workers, "deno", `${navigator.userAgent} (${Deno.build.target})`);

// log received messages
(async () => {
  for await (const event of provider.messenger!.events) {
    const typename: string = event.$typeName;
    delete event.$typeName;
    if (!typename.endsWith("Throughput"))
      console.log(`%c[${typename}]`, "color: green;", JSON.stringify(event));
  };
})();

// register signal handlers for clean exits
let forcequit = false; // force immediate on second signal
Deno.addSignalListener("SIGINT", async () => {
  if (forcequit) {
    console.error(" kill")
    await provider.disconnect();
    Deno.exit(1);
  };
  console.log(" shutdown (send signal again to force immediate exit)");
  forcequit = true;
  // wait to finish all current tasks, then quit
  await provider.pool.scale(0);
  // TODO: short timeout to allow outgoing messages, should be a flush
  await new Promise(r => setTimeout(r, 20));
  await provider.disconnect();
  Deno.exit(0);
});

// start handling requests
await provider.handlerequests();

console.error("ERROR: rpc loop exited, connection lost?");
Deno.exit(1);
