# Deno Provider

This was originally a pretty quick MVP of the Provider code running in a terminal with [Deno](https://docs.deno.com/). Since Deno supports the Web platform API very well and runs TypeScript code directly, you can entirely reuse the Provider implementation from `../webprovider/lib/` by appropriately redirecting a few imports.

```
./main.ts --url http://localhost:4080
# or:
deno run --allow-all --unstable-sloppy-imports main.ts --url http://localhost:4080
```

One of Deno's features is the permissions model to deny certain accesses by default but we end up needing most of them anyway:
* `--allow-net`: to use networking, for obvious reasons
* `--allow-env`: mostly because `@bufbuild/protobuf` tries to access some config flags
* `--allow-read`: new Web Workers need to read the `wasiworker.ts` script file to launch
* `--allow-write`: the Pyodide runner needs to download and cache the imported modules on disk

The optional configuration flags are:

| flag | description | default |
| ---- | ----------- | ------- |
| `--workers <n>` | Specify the number of concurrent Workers to spawn | `navigator.hardwareConcurrency`, i.e. logical processors |
| `--url http(s)://...` | Base URL to the Broker; the correct path is appended automatically | `http://localhost:4080` |

#### Containerized Launch

See in parent directory for a Dockerfile, which builds a container using this script. To quickly launch a prebuilt container image use:

```
docker run --rm -t ghcr.io/wasimoff/provider --url http://broker.example.com
```

*Hint: you might want to use `--net host` if the Broker is running on `localhost` itself, as the container won't be able to access it otherwise.*

### Minimal Example

If you remove everything that isn't strictly necessary to run the Provider with Deno, you're left with:

```typescript
// @wasimoff is an import alias to the webprovider/lib code
import { WasimoffProvider } from "@wasimoff/worker/provider.ts";

// start the provider
const provider = await WasimoffProvider.init(
  navigator.hardwareConcurrency,   // how many workers
  "http://localhost:4080",         // broker url
  ":memory:",                      // keep files in memory
);

// spawn workers up to capacity
const workers = await provider.pool.scale();

// tell the broker about us
await provider.sendInfo(workers, "deno", `${navigator.userAgent} (${Deno.build.target})`);

// handle remote procedure calls
await provider.handlerequests();
```
