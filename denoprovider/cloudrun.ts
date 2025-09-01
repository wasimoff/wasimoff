#!/usr/bin/env -S deno run --allow-env --allow-read --allow-write --allow-net --no-prompt --unstable-sloppy-imports

import { Application } from "@oak/oak";
import { Wasip1TaskParams } from "@wasimoff/worker/wasiworker.ts";
import { WasiWorkerPool } from "@wasimoff/worker/workerpool.ts";
import { ProviderStorage } from "@wasimoff/storage/index.ts";
import { getRootfsZip } from "@wasimoff/worker/rpchandler.ts";
import { Terminator } from "./util.ts";

import * as wasimoff from "@wasimoff/proto/v1/messages_pb.ts";
import * as pb from "@bufbuild/protobuf";

console.log("%c[Wasimoff]", "color: red", "starting FaaS Provider", {
  cpu: navigator.hardwareConcurrency,
  agent: navigator.userAgent,
});

// initialize a storage to cache modules and zip files in memory
const origin = Deno.env.get("BROKER_ORIGIN") || "https://wasi.team";
const storage = new ProviderStorage(":memory:", origin);

// create a worker pool with one thread per logical cpu
const pool = new WasiWorkerPool(navigator.hardwareConcurrency);
await pool.scale();

// port that the app will listen on
const port = Number(Deno.env.get("PORT")) || 8000;

// create the oak app for request handler
const app = new Application({ state: { shutdown: false, counter: 0 } });
app.use(async (ctx) => { // https://jsr.io/@oak/oak/doc/context/~/Context
  // request counter for readable logging
  const i = `task[${app.state.counter++}]`;

  // is the app shutting down? (don't accept new tasks)
  if (app.state.shutdown) {
    ctx.response.status = 503;
    ctx.response.body = "going away";
    return;
  }

  let response: wasimoff.Task_Wasip1_Response | null = null;
  const content_type = ctx.request.headers.get("content-type") as
    | "application/json"
    | "application/proto";

  try {
    // parse the incoming request
    let request: wasimoff.Task_Wasip1_Request;
    switch (content_type) {
      case "application/json":
        request = pb.fromJson(
          wasimoff.Task_Wasip1_RequestSchema,
          await ctx.request.body.json(),
        );
        break;

      case "application/proto":
        request = pb.fromBinary(
          wasimoff.Task_Wasip1_RequestSchema,
          new Uint8Array(await ctx.request.body.arrayBuffer()),
        );
        break;

      default:
        throw new Error(
          "only accepting Task_Wasip1_Request in JSON or Protobuf encoding",
        );
    }

    // mostly copied from rpchandler.ts from here on ...

    // deconstruct the request and check type
    const { info, params } = request;
    if (info === undefined || params === undefined) {
      throw "info and params cannot be undefined";
    }

    const task = params;
    if (task.binary === undefined) {
      throw "wasip1.binary cannot be undefined";
    }

    // get or compile the webassembly module
    let wasm: WebAssembly.Module;
    if (task.binary.blob.length !== 0) {
      wasm = await WebAssembly.compile(task.binary.blob);
    } else if (task.binary.ref !== "") {
      const m = await storage.getWasmModule(task.binary.ref);
      if (m === undefined) throw "binary not found in storage";
      else wasm = m;
    } else {
      throw new Error("binary: neither blob nor ref were given");
    }

    console.debug(i, "run:", info.id);
    const result = await pool.runWasip1(info.id, {
      wasm: wasm,
      argv: task.args || [],
      envs: task.envs || [],
      stdin: task.stdin,
      rootfs: await getRootfsZip(storage, task.rootfs),
      artifacts: task.artifacts,
    } as Wasip1TaskParams);

    // format and send back the result protobuf
    response = pb.create(wasimoff.Task_Wasip1_ResponseSchema, {
      result: {
        case: "ok",
        value: {
          status: result.returncode,
          stdout: result.stdout,
          stderr: result.stderr,
          artifacts: result.artifacts ? { blob: result.artifacts } : undefined,
        },
      },
    }) as wasimoff.Task_Wasip1_Response;
  } catch (error) {
    // format exceptions as WasiResponse.Error
    console.error(i, error);
    response = pb.create(wasimoff.Task_Wasip1_ResponseSchema, {
      result: { case: "error", value: String(error || "unspecified error") },
    }) as wasimoff.Task_Wasip1_Response;
    ctx.response.status = 400;
  } finally {
    // serialize the response, if any
    if (response !== null) {
      switch (content_type) {
        case "application/json":
          ctx.response.type = "json";
          ctx.response.body = pb.toJson(
            wasimoff.Task_Wasip1_ResponseSchema,
            response,
          );
          break;

        case "application/proto":
          ctx.response.type = content_type;
          ctx.response.body = pb.toBinary(
            wasimoff.Task_Wasip1_ResponseSchema,
            response,
          );
          break;

        default:
          // assert unreachable
          ((_: never) => {})(content_type);
      }
    }
  }
});

// register signal handler for clean exits
// GCP gives 10s grace period before SIGKILL
new Terminator(pool, 9_000, (_) => {
  app.state.shutdown = true;
});

// start the webserver
console.log(
  "%c[Wasimoff]%c oak listening on port %c%d",
  "color: red",
  "color: none;",
  "color: cyan;",
  port,
);
await app.listen({ port });
