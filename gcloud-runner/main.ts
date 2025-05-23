import functions from "@google-cloud/functions-framework";
import { WasiWorker } from "@wasimoff/worker/wasiworker";

import os from "os";
import fs from "fs/promises";
import path from "path";

// defined git revision during rollup build
const version = process.env.VERSION || "unknown";

functions.http("wasimoff", async (req, res) => {
  try {

    // check if the binary already exists or download it to /tmp
    const binaryUrl = req.body.binary;
    const binaryPath = path.join("/tmp", path.basename(binaryUrl));
    let wasm: Buffer;
    let cached = false;
    try {
      // try to use local file
      wasm = await fs.readFile(binaryPath);
      cached = true;
    } catch (error) {
        // otherwise download and store
        const resp = await fetch(binaryUrl);
        if (!resp.ok) {
          return res.status(500).json({
            status: resp.status, text: resp.statusText,
            err: "failed to fetch binary",
          });
          // throw new Error(`failed to fetch binary: ${resp.statusText}`);
        };
        wasm = Buffer.from(await resp.arrayBuffer());
        await fs.writeFile(binaryPath, wasm);
    };

    // other required parameters for wasi
    const argv = req.body.argv || [ ];
    const envs = req.body.envs || [ ];

    // construct a new wasimoff worker and run the task
    const runner = new WasiWorker(0, false);
    const result = await runner.runWasip1("faas", { argv, envs, wasm });

    // decode the stdio streams as text
    const stdout = new TextDecoder("utf-8").decode(result.stdout);
    const stderr = new TextDecoder("utf-8").decode(result.stderr);

    // get some memory usage stats after execution
    const total = os.totalmem();
    const free = os.freemem();
    const megabyte = (1024*1024);
    const memory = {
      used: Number(((total - free) / megabyte).toFixed(2)),
      total: Number(((total) / megabyte).toFixed(2)),
      percent: Number(((1 - (free / total)) * 100).toFixed(1)),
    };

    // send the results
    res.json({ version, cached, memory, result: { code: result.returncode, stdout, stderr } });

  } catch (err) {
    res.statusMessage = "error running task";
    res.status(500).json({ error: err });
  };
});

