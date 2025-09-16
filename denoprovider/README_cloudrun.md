# Google Cloud Run Function Runner

_What a name!_

This is an experiment to deploy the equivalent of a single Wasimoff runner in a Google Cloud
function, i.e. a "true" FaaS-platform. It can be configured as an additional offloading target to be
used when no other Providers are currently connected to a Broker.

- You'll need a GCP account and install and authenticate locally using the `gcloud` CLI.
- The https://wasi.team URL is currently hardcoded in a few places.
- The deployment script `yarn deploy` probably won't work without modification for you.

It uses the same idea as the Deno provider, in that you can re-use most of the TypeScript code in
`../webprovider/lib/` by using a suitable bundler like [Rollup.js](https://rollupjs.org/), which
transpiles and packs everything in a single `main.js` JavaScript file, to be containerized by the
default NodeJS buildpack.

Each function invocation currently expects a `wasimoff.Task_Wasip1_Request` message in the body,
either JSON (`application/json`) or Protobuf (`application/proto`) encoded. Support for Pyodide
tasks is not tested and added yet.

You can run the function server locally for testing using `yarn dev`. Do **note, however** that
since the script only starts a single runner and NodeJS does not easily support spawning new Web
Workers (and I didn't want to write a shim for `worker_threads` yet), the implementation is
effectively single-threaded. Not a problem on GCP, where you can just let each call spawn a new
instance for compute-heavy workloads. You could probably just as well use a Deno image, to support
spawning multiple Workers.
