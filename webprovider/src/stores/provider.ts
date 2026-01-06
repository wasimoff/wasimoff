import { defineStore } from "pinia";
import { ref } from "vue";

import { WasimoffProvider } from "@wasimoff/worker/provider";
import { WasiWorkerPool } from "@wasimoff/worker/workerpool";
import { ProviderStorage } from "@wasimoff/storage";
import { Messenger } from "@wasimoff/transport";
import { useTerminal } from "./terminal";
import { useConfiguration } from "./configuration";

export const useProvider = defineStore("WasimoffProvider", () => {
  // whether we are currently connected to the broker
  const connected = ref(false);

  // current state of workers in the pool
  const workers = ref<boolean[]>([]);

  // update busy map on interval
  setInterval(async () => {
    // this interval slows my devtools inspector to a crawl but works fine when closed
    if (pool.value) workers.value = pool.value.busy;
  }, 50);

  // keep direct references instead of proxies (no $ prefix needed)
  const provider = ref<WasimoffProvider>();
  const pool = ref<WasiWorkerPool>();
  const messenger = ref<Messenger>();
  const storage = ref<ProviderStorage>();

  // have a terminal for logging
  const terminal = useTerminal();

  // load configuration values
  const config = useConfiguration();

  // check if we're running exclusively (not open in another tab)
  const exclusive = new Promise<void>((resolve) => {
    if ("locks" in navigator) {
      navigator.locks.request("wasimoff", { ifAvailable: true }, async (lock) => {
        if (lock === null) {
          return terminal.error("ERROR: another Provider is already running; refusing to start!");
        }
        // got the lock, continue startup
        resolve();
        // return an "infinite" Promise; lock is only released when tab is closed
        return new Promise((r) => window.addEventListener("beforeunload", r));
      });
    } else {
      // can't check the lock, warn about it and continue anyway
      terminal.warn("WARNING: Web Locks API not available; can't check for exclusive Provider!");
      resolve();
    }
  });

  // start the provider when the lock has been acquired
  exclusive.then(async () => {
    // instantiate the provider directly in the main thread
    connected.value = false;
    provider.value = new WasimoffProvider(config.workers, undefined, undefined, config.verbose);

    // wrap the pool in a proxy to keep worker count updated
    pool.value = new Proxy(provider.value.pool, {
      // trap property accesses that return methods which can change the pool length
      get: (target, prop, receiver) => {
        const traps = ["spawn", "scale", "drop", "killall"];
        const method = Reflect.get(target, prop, receiver);
        // wrap the function calls with an update to the broker
        if (typeof method === "function" && traps.includes(prop as string)) {
          return async (...args: any[]) => {
            let result = (await (method as any).apply(target, args)) as Promise<number>;
            try {
              workers.value = target.busy;
            } catch {}
            return result;
          };
        } else {
          // anything else is passed through
          return method;
        }
      },
    });

    // try to grab a wakelock to keep screen on
    if ("wakeLock" in navigator) {
      try {
        const lock = await navigator.wakeLock.request("screen");
        terminal.info(
          "Acquired wakelock. Screen timeout is disabled as long as this tab remains in foreground.",
        );
        lock.addEventListener("release", () => terminal.warn("Wakelock was revoked!"));
        window.addEventListener("beforeunload", () => lock.release());
      } catch (err) {
        terminal.warn(`Could not acquire wakelock: ${err}`);
      }
    } else {
      terminal.info("Wakelock API unavailable.");
    }
  });

  async function open(...args: Parameters<WasimoffProvider["open"]>) {
    if (!provider.value) throw "no provider connected yet";
    // open the filesystem, get direct reference to storage
    await provider.value.open(...args);
    storage.value = provider.value.storage;
  }

  async function connect(...args: Parameters<WasimoffProvider["connect"]>) {
    if (!provider.value) throw "no provider connected yet";
    // connect the transport (waits for readiness), get direct reference to messenger
    await provider.value.connect(...args);
    messenger.value = provider.value.messenger;
    connected.value = true;
    // fill the pool it it's empty
    if (workers.value.length === 0 && pool.value) {
      // doing it manually here is more responsive, because
      // each spawn updates the workers ref
      let capacity = pool.value.capacity;
      while (pool.value.length < capacity) await pool.value.spawn();
    }
  }

  async function disconnect() {
    if (!provider.value) throw "no provider connected yet";
    await provider.value.disconnect();
    connected.value = false;
  }

  async function handlerequests() {
    if (!provider.value) throw "no provider connected yet";
    await provider.value.handlerequests();
    // the above promise only returns when the loop dies
    connected.value = false;
  }

  // exported as store
  return {
    // plain refs
    connected,
    workers,
    // direct references (no comlink proxies)
    $provider: provider, // Keep $ naming for compatibility
    $pool: pool,
    $messenger: messenger,
    $storage: storage,
    // special-cased methods
    open,
    connect,
    disconnect,
    handlerequests,
  };
});
