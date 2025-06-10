import functions from "@google-cloud/functions-framework";
import { WasiWorker, Wasip1TaskParams } from "@wasimoff/worker/wasiworker";
import { ProviderStorage } from "@wasimoff/storage";
import { getRootfsZip } from "@wasimoff/worker/rpchandler";

import * as wasimoff from "@wasimoff/proto/v1/messages_pb";
import * as pb from "@bufbuild/protobuf";

// defined git revision during rollup build
const version: string = process.env.VERSION || "unknown";
console.log("starting wasimoff-faas-runner, version", version);

// initialize a storage to cache modules and zip in memory
// TODO: get origin from env or implement proper url handling
const storage = new ProviderStorage(":memory:", "https://wasi.team");

/** Each request should be an offloading task, so either  */
functions.http("wasimoff", async (req, res) => {
  try {

    // parse the incoming request
    let request: wasimoff.Task_Wasip1_Request;
    let content_type = req.header("content-type") as "application/json" | "application/proto";
    switch (content_type) {
      case "application/json":
        request = pb.fromJson(wasimoff.Task_Wasip1_RequestSchema, req.body) as wasimoff.Task_Wasip1_Request;
        break;
      case "application/proto":
        request = pb.fromBinary(wasimoff.Task_Wasip1_RequestSchema, req.body) as wasimoff.Task_Wasip1_Request;
        break;
      default:
        throw new Error("only accepting Task_Wasip1_Request in JSON or Protobuf encoding");
        break;
    };

    // mostly copied from rpchandler.ts from here on ...

    // deconstruct the request and check type
    let { info, params } = request;
    if (info === undefined || params === undefined)
      throw "info and params cannot be undefined";

    const task = params;
    if (task.binary === undefined)
      throw "wasip1.binary cannot be undefined";

    // get or compile the webassembly module
    let wasm: WebAssembly.Module;
    if (task.binary.blob.length !== 0) {
      wasm = await WebAssembly.compile(task.binary.blob);
    } else if (task.binary.ref !== "") {
      let m = await storage.getWasmModule(task.binary.ref);
      if (m === undefined) throw "binary not found in storage";
      else wasm = m;
    } else {
      throw new Error("binary: neither blob nor ref were given");
    };

    // construct a new wasimoff worker and run the task
    // console.debug(info.id, task);
    const runner = new WasiWorker(0, false);
    const result = await runner.runWasip1(info.id, {
      wasm: wasm,
      argv: task.args || [],
      envs: task.envs || [],
      stdin: task.stdin,
      rootfs: await getRootfsZip(storage, task.rootfs),
      artifacts: task.artifacts,
    } as Wasip1TaskParams);
    printMemoryUsage();

    // format and send back the result protobuf
    const response = pb.create(wasimoff.Task_Wasip1_ResponseSchema, {
      result: { case: "ok", value: {
        status: result.returncode,
        stdout: result.stdout,
        stderr: result.stderr,
        artifacts: result.artifacts ? { blob: result.artifacts } : undefined,
      }},
    });
    switch(content_type) {
      case "application/json":
        res.json(pb.toJson(wasimoff.Task_Wasip1_ResponseSchema, response));
        break;
      case "application/proto":
        res.setHeader("content-type", content_type);
        res.send(Buffer.from(pb.toBinary(wasimoff.Task_Wasip1_ResponseSchema, response)));
        break;
      default:
        // this should never happen ..
        throw new Error("unknown content-type to return");
        break;
    }

  } catch (error) {
    // format exceptions as WasiResponse.Error
    const err = pb.create(wasimoff.Task_Wasip1_ResponseSchema, {
      result: { case: "error", value: String(error || "unspecified error"), },
    });
    res.status(400).json(err);
    return;
    // res.statusMessage = "Task Error";
    // res.status(400).json({ error: error !== undefined ? String(error) : "unspecified error" });
  };
});

// print { used, total, percent } of memory
import os from "os";
function printMemoryUsage() {
  const total = os.totalmem();
  const free = os.freemem();
  const megabyte = (1024*1024);
  const memory = {
    used: Number(((total - free) / megabyte).toFixed(2)),
    total: Number(((total) / megabyte).toFixed(2)),
    percent: Number(((1 - (free / total)) * 100).toFixed(1)),
  };
  console.log("memory:", memory);
}
