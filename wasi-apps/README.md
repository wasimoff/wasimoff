# Example Applications

This directory contains a few example applications that can be compiled to WebAssembly WASIp1 and execute on Wasimoff. This target is well supported in the [Go](https://go.dev/blog/wasi) and [Rust](https://doc.rust-lang.org/beta/rustc/platform-support/wasm32-wasip1.html) toolchains.

The general idea is that you can develop your workload application locally just like a normal commandline application, i.e. use a commandline parser for string arguments, read input from standard input or a file from the working directory. When you're ready, compile your application to the `wasip1` target and upload it to Wasimoff, where it will be distributed to run on any volunteering devices that are currently connected.

So, instead of your "Hello, World!" running locally like this:

```bash
$ go build -o hello helloworld.go
$ ./hello "Alice"
Hello, Alice!
```

It might be executed on a volunteer's connected browser like this:

```bash
$ GOARCH=wasm GOOS=wasip1 go build -o hello.wasm helloworld.go
$ curl -X POST https://wasi.team/api/storage/upload -d @hello.wasm
$ curl -X POST https://wasi.team/api/client/run/hello.wasm -H "X-Args: Alice"
Hello, Alice!
```

Usage is even simpler when using the [`wasimoff`](https://pkg.go.dev/wasi.team/client) client binary.

### apps

- `argonload`: Rust binary using the [argon2](https://docs.rs/argon2/latest/argon2/) crate to generate predictable computational load; in a previous life, this was a password hashing algorithm.

- `helloworld`: contains a simple application reimplemented in Go, Rust and pure WAT; the former two showcase a few examples using the WASI syscalls, like reading files and output to stdout.

- `travelling_salesman`: a Rust application that uses an exhaustive route search between some arbitrary coordinates of random German cities. You'll see this a lot as a somewhat predictable compute workload around Wasimoff. See `../client/examples/tsp/` on how to use it.

- `ffmpeg`: Link to an external project, which compiles the multimedia trancoding toolkit `ffmpeg` to WASI. I won't pretend that it makes much sense to software-encode h264 videos in a browser tab, _but it works!_

- ... and probably more.
