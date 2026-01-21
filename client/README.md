# Wasimoff Client

This is an example implementation of the Wasimoff Client using either ConnectRPC or a WebSocket
connection. Install the demo CLI `wasimoff` with:

```
go install wasi.team/client/cmd/wasimoff@latest
```

Alternatively, you can build it yourself from within this directory with:

```
go build ./cmd/wasimoff
```

The main operations of this CLI are:

- **Upload:** `-upload <file>`
- **Execute:** `-exec <ref> [<args>]`
- **Pyodide:** `-pyodide <script.py>`
- **Task:** `-task <task.json>`

Other options include:

- `-broker <url>` to use a different Broker
- `-ws` to use a WebSocket for asynchronous task submission
- `-verbose` to print a few more intermediate steps
- `-stdin` to read and send contents from `/dev/stdin` with `-exec` tasks
- `-rootfs` to use a rootfs ZIP with `-exec` tasks

#### Uploading files

First, upload your WebAssembly WASI preview 1 executable (and optional rootfs ZIP) with:

```
wasimoff -upload <file>
```

For example `wasimoff -upload examples/tsp/tsp.wasm` for the travelling salesman binary. The command
will also print a `sha256:..` reference hash, which you can use instead of the filename later. The
Broker currently only accepts `application/wasm` (for the binary) and `application/zip` (for the
rootfs archive) content-types.

#### Execute

You can then start a single invocation as if you were starting the WebAssembly binary locally with:

```
wasimoff -exec <filename> [<args>]
```

Again, using the TSP example, you could calculate the route for ten random cities using
`wasimoff -exec tsp.wasm rand 10` and get the output back on stdout. This is approximately
equivalent to running `wasmtime tsp.wasm ...`. If you need data from stdin, use `-stdin`, which will
read until EOF and send it as a binary blob in the request.

#### Tasks

Finally, you can write a JSON file to start a predefined task on demand:

```json
{
  // the file is parsed as a google.protobuf.Any message, so you need to specify the type
  "@type": "wasimoff.v1.Task.Wasip1.Request",
  // now the rest of the object is parsed as a wasimoff.Task_Wasip1_Request
  "params": {
    // the executable to start; use either "ref" with a filename / sha256 reference
    // or "blob" with a base64-encoded binary blob as a string
    "binary": { "ref": "hello.wasm" },

    // optional: rootfs can contain a ZIP file that is extracted in the virtual
    // WASI filesystem before execution; use "ref" or "blob" as above
    "rootfs": { "ref": "hello.zip" },

    // environment variables and commandline arguments, pretty straightforward;
    // the first string in "args" is actually "arg0", i.e. the filename as seen
    // by the executable itself
    "envs": ["DOS=demonstration"],
    "args": ["hello.wasm", "print_envs", "print_rootfs", "file:hello.txt"],

    // optional: data to be passed to the application on stdin; must be base64
    // encoded, as it is not necessarily a valid utf-8 string
    "stdin": "SGVsbG8sIFdvcmxkIQo=",

    // optional: artifacts can be a list of files to return to the client in a
    // ZIP file after execution; useful if the app writes results "to disk"
    "artifacts": ["hello.txt"]
  }
}
```

The file is parsed as an "any" message, which can be either a `wasimoff.Task_Wasip1_Request` or
`wasimoff.Task_Pyodide_Request`.

#### Embedded client

In your own application, you'd rather construct these Protobuf messages directly instead of
serializing JSON first. To that end, you can import these packages via the `wasi.team` vanity
import:

```
go get -u wasi.team/proto
go get -u wasi.team/client
```

```go
package main

import (
  ...
  wasimoff "wasi.team/proto/v1"
  "wasi.team/client"
)

func main() {

  // connect a client
  var wc client.WasimoffClient
  wc = client.NewWasimoffConnectRpcClient(http.DefaultClient, "https://wasi.team")

  // construct the request
  request := &wasimoff.Task_Wasip1_Request{
    Params: &wasimoff.Task_Wasip1_Params{
      // Binary, Args, Envs, ...
    },
  }

  // send it
  response, err := wc.RunWasip1(context.Background(), request)
  // err is a general RPC failure
  // response.GetError() is a "technically successful" RPC, where the execution itself failed

}
```

Of course, you're free to instantiate your own ConnectRPC client from
`wasi.team/proto/v1/wasimoffv1connect.NewTasksClient()` or use the WebSocket interface from
`wasi.team/broker/net/transport.DialWasimoff()` directly. The latter is especially recommended if
you plan to submit many tasks asynchronously, which don't necessarily constitute a single "job".
