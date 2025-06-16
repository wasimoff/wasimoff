# Wasimoff Client

This shows an example implementation of the Wasimoff Client as a commandline interface. To build the CLI run `go build`. (You can also use `go install wasi.team/client@latest` but this currently clobbers the very generic executable name `client`. Sorry, I need to rename this package.)

```
go build -o wasimoff client.go
```

The main operations of this CLI are `-upload <file>`, `-exec <filename> [<args>]` and `-run <jobfile>`.

Other options include `-broker <url>` to use a different Broker, `-ws` to use a WebSocket for asynchronous task submission with `-run`, and `-verbose` to print a few more intermediate steps.

#### Uploading files

First, upload your WebAssembly WASI preview 1 executable (and optional rootfs ZIP) with:

```
./wasimoff -upload <file>
```

For example `./wasimoff -upload tsp.wasm` for the travelling salesman example. The command will also print a `sha256:..` reference, which you can use instead of the filename later. The Broker currently only accepts `application/wasm` and `application/zip` content-types.

#### Execute

You can then start a single invocation as if you were starting the WebAssembly binary locally with:

```
./wasimoff -exec <filename> [<args>]
```

Again, using the TSP example, you could calculate the route for ten random cities using `./wasimoff -exec tsp.wasm rand 10` and get the output back on stdout. This is approximately equivalent to running `wasmtime tsp.wasm ...`. If you need data from stdin, use `-stdin`, which will read until EOF and send it as a binary blob in the request.

#### Jobs

Finally, you can write a JSON job file to start multiple tasks at the same time. The job file looks like this:

```json
{
  // tasks is a list of wasimoff.Task_Wasip1_Params
  "tasks": [{

    // the executable to start; use either "ref" with a filename / sha256 reference
    // or "blob" with a base64-encoded binary blob as a string
    "binary": { "ref": "hello.wasm" },

    // optional: rootfs can contain a ZIP file that is extracted in the virtual
    // WASI filesystem before execution; use "ref" or "blob" as above
    "rootfs": { "ref": "hello.zip" },

    // environment variables and commandline arguments, pretty straightforward;
    // the first string in "args" is actually "arg0", i.e. the filename as seen
    // by the executable itself
    "envs": [ "DOS=demonstration" ],
    "args": [ "hello.wasm", "print_envs", "print_rootfs", "file:hello.txt" ],

    // optional: data to be passed to the application on stdin; must be base64
    // encoded, as it is not necessarily a valid string
    "stdin": "SGVsbG8sIFdvcmxkIQo=",

    // optional: artifacts can be a list of files to return to the client in a
    // ZIP file after execution; useful if the app writes results "to disk"
    "artifacts": [ "hello.txt" ]

  }]
}
```


The file is parsed as a `wasimoff.Task_Wasip1_JobRequest`, where the `parent` and each element in `tasks` is a `wasimoff.Task_Wasip1_Params`. The tasks use a very simple inheritance: each empty / `nil` field is copied straight from the parent (that means `envs` do not get concatenated either, for example). The following is an example job, which starts three identical TSP tasks, each computing a path for ten random cities:


```json
{
  "parent": {
    "binary": { "ref": "tsp.wasm" },
    "args": [ "tsp.wasm", "rand", "10" ]
  },
  "tasks": [ {}, {}, {} ]
}
```

#### Embedded client

In your own application, you'd rather construct these Protobuf messages directly instead of serializing JSON first. To that end, you can import the message definitions via the `wasi.team` vanity import, which points to the root of this repository:

```
go get -u wasi.team/proto
```

```go
package main

import (
  ...
  wasimoff "wasi.team/proto/v1"
)
```

* The generated ConnectRPC client can be imported from `wasi.team/proto/v1/wasimoffv1connect`.
* To use the WebSocket interface, import `wasi.team/broker/net/transport` and use `transport.DialWasimoff()`.

At the moment, I can only direct you towards the `client.go` as an implementation example, sorry. The imports should work form anywhere though, so you can just copy the entire file to get started.

#### TODO

* Create an actual client library at this import path, which wraps all the necessary bits and pieces.
* Move example CLI to `wasi.team/client/cmd/wasimoff`.
