# Example Applications

This directory contains a few example applications that can be compiled to WebAssembly and execute
on Wasimoff.

- `helloworld/`: contains a simple application reimplemented in Go, Rust and pure WAT. The former
  two showcase a few examples using the WASI syscalls, like reading files and output to stdout.
- `travelling_salesman/`: a Rust application that uses an exhaustive route search between some
  arbitrary coordinates of random German cities. You'll see this a lot as a somewhat predictable
  compute workload around Wasimoff. See `../client/examples/tsp/` on how to use it.
- `ffmpeg/`: Link to an external project, which compiles the multimedia trancoding toolkit `ffmpeg`
  to WASI. I won't pretend that it makes much sense to software-encode h264 videos in a browser tab,
  _but it works!_
