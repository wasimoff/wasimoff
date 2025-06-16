# wasimoff

This is the codebase for the computation offloading and task distribution research prototype **wasimoff**. It is a framework which uses WebAssembly as its executable format and is able to send computational tasks to volunteers' browsers through a central broker. Targeting a modern browser API as an execution environment makes participation *exceedingly* simple, while the versatility of WebAssembly supports many different computational problems.

![](architecture.png)

Publications relating to this project:

* [Semjonov, A. (2023). *Opportunistic Distributed Computation Offloading using WebAssembly* (Master’s thesis. University of Hamburg).](https://edoc.sub.uni-hamburg.de/informatik/volltexte/2024/273/pdf/Anton_Semjonov_Opportunistic_Distributed_Computation_Offloading_using_WebAssembly.pdf)

* [Semjonov, A., Bornholdt, H., Edinger, J., & Russo, G. R. (2024, March). *Wasimoff: Distributed computation offloading using WebAssembly in the browser.* In *2024 IEEE International Conference on Pervasive Computing and Communications Workshops and other Affiliated Events (PerCom Workshops)* (pp. 203-208). IEEE.](https://ieeexplore.ieee.org/abstract/document/10503392/) (+ [Artifact](https://ieeexplore.ieee.org/abstract/document/10502812/))

* [Semjonov, A., & Edinger, J. (2024, December). *Demo: Zero-Setup Computation Offloading to Heterogeneous Volunteer Devices Using Web Browsers.* In *Proceedings of the 25th International Middleware Conference: Demos, Posters and Doctoral Symposium* (pp. 3-4).](https://dl.acm.org/doi/abs/10.1145/3704440.3704776)


### Components

The three essential roles in Wasimoff are **Broker** (central entity), **Provider** (resource providers, the machines that execute the workload) and **Client** (users wishing to use the system in FaaS-style). Broker and Client can be found in their respective subdirectories and there's multiple Provider implementations covering modern web browsers and terminals / servers.

* The **Broker** (`broker/`) is a central entity to which all Providers connect and which then distributes tasks among them. Clients upload WebAssembly executables (think of this like registering a function in FaaS) and then queue tasks using these executables. The Broker is written in Go and uses Protobuf messages over WebSocket connections.

* **Providers** are the participants that share their resources with the network. An important goal of this prototype was to implement the Provider entirely on the Web platform API, so it can run in the browser simply by opening a web page.
  * A browser implementation (`webprovider/`) is written in Vue.js and uses Web Workers to execute the WebAssembly modules concurrently.
  * The exact same TypeScript code can also be run with Deno (`denoprovider/`), which makes it easy to start Providers on a server or deploy them with Docker.

* The **Client** (`client/`) interface is either a simple ConnectRPC HTTP API or also a WebSocket connection for asynchronous task submission. Examples exist using `curl` in Bash, as well as a CLI written in Go. It can be used to send individual tasks or schedule a large number of similar tasks with job configuration files.

More detailed documentation can be found in each subdirectory.

#### Protobuf

The communication interfaces of the Broker all use Protobuf messages, which are defined in `proto/v1/messages.proto`. This makes it easy to deploy automatically generated client APIs using ConnectRPC on the one hand and have a well-defined RPC mechanism to the Providers as well.

### Containerized Deployment

This repository includes a multi-stage `Dockerfile`, which compiles the Broker binary in a Go image, compiles the Provider frontend in a NodeJS image, copies both to a barebones container image (`--target wasimoff`) and also prepares another headless provider image using Deno (`--target provider`). These containers are [built automatically in a GitHub action](https://github.com/wasimoff/wasimoff/actions) and published as:

* [`ghcr.io/wasimoff/broker`](https://github.com/wasimoff/wasimoff/pkgs/container/broker)
* [`ghcr.io/wasimoff/provider`](https://github.com/wasimoff/wasimoff/pkgs/container/provider)

For a quick test environment with a Broker and a Provider, use the Docker Compose configuration with:

```
docker compose pull && docker compose up
```

Then go to the `client/` directory and launch some of the example applications.


### WASI applications

The WebAssembly System Interface (WASI) was chosen initially as an abstraction layer for the offloaded tasks and the subdirectory `wasi-apps/` contains a number of example applications, which use this compilation target to show off its versatility and serve as example workloads during the evaluation.

* `ffmpeg` is a compilation of the popular FFmpeg toolkit to WebAssembly. It can be used to transcode videos in a browser tab.
* `helloworld` is a collection of different languages (Rust, TinyGo, WAT) targeting basic features of the WASI target `wasm32-wasi`.
* `travelling_salesman` is an implementation of the .. *you guessed it* .. Travelling Salesman Problem in Rust. It can be compiled to various native binary formats but also to WASI and it serves as the computational workload in all of the evaluation runs.
* `web-demo` is a minimal example of using the `browser_wasi_shim` to execute WebAssembly binaries with environment variables and commandline arguments in the browser.

### Experiments

The sibling [`experiments` repository](https://github.com/wasimoff/experiments) contains various experiments and evaluations of pieces of the networking stack etc. that were considered during development.
