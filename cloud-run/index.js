const functions = require('@google-cloud/functions-framework');
const shim = require("@bjorn3/browser_wasi_shim");

const os = require('os');
const fs = require('fs').promises;
const path = require('path');

const version = "storefile/1";

functions.http('runwasi', async (req, res) => {
    //res.send(`Hello ${req.query.name || req.body.name || 'World'}!`);
    try {

        // get some stats before execution
        const free = () => ({
            total: os.totalmem(),
            used: os.totalmem() - os.freemem(),
        });
        const membefore = free();
        const tmpdir = await fs.readdir("/tmp");

        // check if the binary already exists or download it to /tmp
        const binaryUrl = req.body.binary;
        const binaryPath = path.join("/tmp", path.basename(binaryUrl));
        let binary; // buffer
        let cached = false;

        try {
            // try to use local file
            binary = await fs.readFile(binaryPath);
            cached = true;
        } catch (error) {
            // otherwise download and store
            const resp = await fetch(binaryUrl);
            if (!resp.ok) throw new Error(`failed to fetch binary: ${resp.statusText}`);
            binary = await resp.arrayBuffer();
            await fs.writeFile(binaryPath, Buffer.from(binary));
        };

        // other required parameters for wasi
        const argv = req.body.argv || [ ];
        const env = req.body.env || [ ];

        // load the webassembly from request body
        //const wasm = await WebAssembly.compile(req.body);
        const wasm = await WebAssembly.compile(binary);

        // prepare the wasi shim
        const fds = [
            new shim.OpenFile(new shim.File([])), // stdin
            new shim.OpenFile(new shim.File([])), // stdout
            new shim.OpenFile(new shim.File([])), // stderr
        ];
        const wasi = new shim.WASI(argv, env, fds);

        // instantiate the webassembly module
        const instance = await WebAssembly.instantiate(wasm, {
            "wasi_snapshot_preview1": wasi.wasiImport,
        });
        wasi.start(instance);
        const memafter = free();

        // decode the outputs for response
        const stdout = new TextDecoder("utf-8").decode(fds[1].file.data);
        const stderr = new TextDecoder("utf-8").decode(fds[2].file.data);
        //res.send(stdout);
        res.json({
            version,
            cached, tmpdir,
            stdout, stderr,
            membefore, memafter,
        });

    } catch (err) {
        res.statusMessage = "Error instantiating WebAssembly module"
        res.status(500).send(String(err));
    };

});
