import { create, isMessage, Message as ProtoMessage } from "@bufbuild/protobuf";
import * as wasimoff from "@wasimoff/proto/v1/messages_pb";
import { getRef, isRef, ProviderStorage } from "@wasimoff/storage/index";
import { type WasimoffProvider } from "./provider";
import { PyodideTaskParams, Wasip1TaskParams } from "./wasiworker";

// Handle incoming RemoteProcedureCalls on the Messenger iterable. Moved into a
// separate file for better readability and separation of concerns in a way.

export async function rpchandler(
  this: WasimoffProvider,
  request: ProtoMessage,
): Promise<ProtoMessage> {
  switch (true) {
    // execute a wasip1 task
    case isMessage(request, wasimoff.Task_Wasip1_RequestSchema):
      return <Promise<wasimoff.Task_Wasip1_Response>> (async () => {
        // deconstruct the request and check type
        let { info, params } = request;
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
          if (this.storage === undefined) throw "cannot access storage yet";
          let m = await this.storage.getWasmModule(task.binary.ref);
          if (m === undefined) throw "binary not found in storage";
          else wasm = m;
        } else {
          throw new Error("binary: neither blob nor ref were given");
        }

        console.debug(...rpcHandlerPrefix, info.id, task);
        try {
          // execute the module in a worker
          let run = await this.pool.runWasip1(info.id, {
            wasm: wasm,
            argv: task.args,
            envs: task.envs,
            stdin: task.stdin,
            rootfs: await getRootfsZip(this.storage, task.rootfs),
            artifacts: task.artifacts,
          } as Wasip1TaskParams);
          // send back the result
          return create(wasimoff.Task_Wasip1_ResponseSchema, {
            result: {
              case: "ok",
              value: {
                status: run.returncode,
                stdout: run.stdout,
                stderr: run.stderr,
                artifacts: run.artifacts ? { blob: run.artifacts } : undefined,
              },
            },
          });
        } catch (err) {
          // format exceptions as WasiResponse.Error
          return create(wasimoff.Task_Wasip1_ResponseSchema, {
            result: { case: "error", value: String(err) },
          });
        }
      })();

    // execute a pyodide task
    case isMessage(request, wasimoff.Task_Pyodide_RequestSchema):
      return <Promise<wasimoff.Task_Pyodide_Response>> (async () => {
        // deconstruct the request and check type
        let { info, params } = request;
        if (info === undefined || params === undefined) {
          throw "info and params cannot be undefined";
        }
        if (params.run.case === undefined) {
          throw "pyodide.run cannot be undefined";
        }
        let task: PyodideTaskParams = {
          packages: params.packages,
          run: params.run.value,
          envs: params.envs,
          stdin: params.stdin,
          rootfs: await getRootfsZip(this.storage, params.rootfs),
          artifacts: params.artifacts,
        };

        console.debug(...rpcHandlerPrefix, info.id, task);
        try {
          let run = await this.pool.runPyodide(info.id, task);
          return create(wasimoff.Task_Pyodide_ResponseSchema, {
            result: {
              case: "ok",
              value: {
                pickle: run.pickle,
                stdout: run.stdout,
                stderr: run.stderr,
                version: run.version,
                artifacts: run.artifacts ? { blob: run.artifacts } : undefined,
              },
            },
          });
        } catch (err) {
          // format exceptions as WasiResponse.Error
          return create(wasimoff.Task_Pyodide_ResponseSchema, {
            result: { case: "error", value: String(err) },
          });
        }
      })();

    // cancel a running task
    case isMessage(request, wasimoff.Task_CancelSchema):
      return <Promise<wasimoff.Task_Cancel>> (async () => {
        const { id, reason } = request;
        if (id !== undefined) {
          console.warn(`cancelling task '${id}': ${reason}`);
          await this.pool.cancel(id);
        } else {
          throw "missing the task id to cancel!";
        }
        return request; // echo back
      })();

    // list files in storage
    case isMessage(request, wasimoff.Filesystem_Listing_RequestSchema):
      return <Promise<wasimoff.Filesystem_Listing_Response>> (async () => {
        if (this.storage === undefined) throw "cannot access storage yet";
        const files = await this.storage.filesystem.list();
        return create(wasimoff.Filesystem_Listing_ResponseSchema, { files });
      })();

    // probe for a specific file in storage
    case isMessage(request, wasimoff.Filesystem_Probe_RequestSchema):
      return <Promise<wasimoff.Filesystem_Probe_Response>> (async () => {
        if (this.storage === undefined) throw "cannot access storage yet";
        let ok = await this.storage.filesystem.get(request.file) !== undefined;
        return create(wasimoff.Filesystem_Probe_ResponseSchema, { ok });
      })();

    // binaries uploaded from the broker inside an rpc
    case isMessage(request, wasimoff.Filesystem_Upload_RequestSchema):
      return <Promise<wasimoff.Filesystem_Upload_Response>> (async () => {
        if (request.upload === undefined) throw "empty upload";
        if (this.storage === undefined) throw "cannot access storage yet";
        let { blob, media, ref } = request.upload;
        // overwrite name with computed digest
        if (!isRef(ref)) ref = await getRef(blob);
        await this.storage.filesystem.put(ref, new File([blob], ref, { type: media }));
        return create(wasimoff.Filesystem_Upload_ResponseSchema, {});
      })();

    default:
      throw "not implemented yet";
  }
}

const rpcHandlerPrefix = ["%c[RPCHandler]", "color: orange;"];

// get rootfs archive from the optional argument in a task
export async function getRootfsZip(
  storage?: ProviderStorage,
  file?: wasimoff.File,
): Promise<Uint8Array | undefined> {
  if (file !== undefined) {
    if (file.blob.length !== 0) {
      // a direct blob was given
      return file.blob;
    } else if (file.ref !== "") {
      // file is a reference, fetch it
      if (storage === undefined) throw "cannot access storage yet";
      const z = await storage.getZipArchive(file.ref);
      if (z === undefined) throw "zip not found in storage";
      return new Uint8Array(z);
    } else {
      throw new Error("rootfs: neither blob nor ref were given");
    }
  }
}
