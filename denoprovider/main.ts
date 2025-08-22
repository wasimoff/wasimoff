#!/usr/bin/env -S deno run --allow-env --allow-read --allow-write --allow-net --no-prompt --unstable-sloppy-imports

import { parseArgs } from "@std/cli/parse-args";
import { nanoid } from "nanoid";
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

// get random client ID from localStorage
let id = localStorage.getItem("wasimoff_id");
if (id === null) {
  id = nanoid();
  localStorage.setItem("wasimoff_id", id);
};

// initialize the provider
console.log("%c[Wasimoff]", "color: red;", "starting Provider in Deno ...");
const provider = await WasimoffProvider.init(nproc, brokerurl, ":memory:", id);
const workers = await provider.pool.scale();
await provider.sendConcurrency(workers);

// log received messages
(async () => {
  for await (const event of provider.messenger!.events) {
    const typename: string = event.$typeName;
    delete event.$typeName;
    if (!typename.endsWith("Throughput"))
      console.log(`%c[${typename}]`, "color: green;", JSON.stringify(event));
  };
})();


// log the currently running tasks
function currentTasks() {
  const now = new Date().getTime(); // unix epoch milliseconds
  return provider.pool.currentTasks
    .filter(w => w.busy)
    .map(w => ({
      worker: w.index, // worker index
      task: w.task, // task ID
      started: w.started, // absolute start date
      age: w.started ? (now - w.started.getTime())/1000 : undefined, // age in seconds
    }));
};

// register signal handlers for clean exits
let forcequit = false; // force immediate on second signal
const graceperiod = 30_000; // grace period in ms, 0 = no grace timeout
async function terminator() {

  // called a second time, quit immediately
  if (forcequit) {
    console.error(" kill")
    console.debug("aborted tasks:", currentTasks());
    await provider.disconnect();
    Deno.exit(1);
  } else {
    console.log(" shutdown (send signal again to force immediate exit)");
    forcequit = true;
  };

  // schedule the grace timeout
  if (graceperiod > 0) setTimeout(terminator, graceperiod);

  // wait to finish all current tasks, then quit
  await provider.pool.scale(0);
  await new Promise(r => setTimeout(r, 20)); // ~ flush
  await provider.disconnect();
  Deno.exit(0);
}
// handle SIGTERM (15) and SIGINT (2) the same
Deno.addSignalListener("SIGTERM", terminator);
Deno.addSignalListener("SIGINT",  terminator);

// start handling requests
await provider.handlerequests();

console.error("ERROR: rpc loop exited, connection lost?");
Deno.exit(1);
