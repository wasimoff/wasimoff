<script setup lang="ts">
import { computed, watch, ref } from "vue";
import { storeToRefs } from "pinia";
import { nanoid } from "nanoid";

// terminal for logging
import { useTerminal, LogType } from "@app/stores/terminal.ts";
const terminal = useTerminal();

// configuration via url fragment
import { useConfiguration } from "@app/stores/configuration.ts";
const conf = useConfiguration();

// the broker socket to connect
const transport = ref(conf.transport);

// info components
import InfoProviders from "./ClusterInfo.vue";

// link to the provider worker
import { useProvider } from "@app/stores/provider.ts";
let wasimoff = useProvider();
// TODO: typings for ref<remote<...> | undefined>?
const { connected, workers, $pool } = storeToRefs(wasimoff);

// connect immediately on load, when the provider proxy is connected
watch(
  () => wasimoff.$provider,
  async (provider) => {
    if (provider !== undefined) {
      await wasimoff.open(transport.value);

      // maybe autoconnect to the broker
      if (conf.autoconnect) await connect();
      else terminal.warn("Autoconnect disabled. Please connect manually.");

      // fill remaining workers to capacity
      if ($pool.value) await fillWorkers();
    }
  },
);

async function connect() {
  try {
    const url = transport.value;
    await wasimoff.connect(url, getlocalid());
    const message = url.startsWith('ad') ? 'Connected to ArtDeco' : 'Connected to Wasimoff Broker';
    terminal.log(`${message} at ${url} as id=${getlocalid()}`, LogType.Success);
    wasimoff.handlerequests();
  } catch (err) {
    terminal.error(String(err));
  }
}

// get (and/or set) a random client ID in localStorage
function getlocalid(): string {
  let id = localStorage.getItem("wasimoff_id");
  if (id === null) {
    localStorage.setItem("wasimoff_id", nanoid());
    return getlocalid();
  }
  return id;
}

// async function kill() {
//   if (!$pool.value) return terminal.error("$pool not connected yet");
//   await $pool.value.killall();
//   // grace period for some error responses
//   await new Promise(r => setTimeout(r, 100));
//   await wasimoff.disconnect();
// };

async function shutdown() {
  if (!$pool.value) return terminal.error("$pool not connected yet");
  terminal.info("Draining tasks .. please wait.");
  await $pool.value.scale(0);
  await wasimoff.disconnect();
}

// class bindings for the transport url field
const connectionStatus = computed(() =>
  connected.value
    ? { "is-success": true, "has-text-grey": true }
    : { "is-danger": false, "has-text-danger": false },
);

// watch connection status disconnections
watch(
  () => connected.value,
  (conn) => {
    if (conn === false) terminal.log("Connection closed.", LogType.Warning);
  },
);

// ---------- WORKER POOL ---------- //

// add / remove / fill workers in the pool
async function spawnWorker() {
  if (!$pool.value) return terminal.error("$pool not connected yet");
  try {
    await $pool.value.spawn();
  } catch (err) {
    terminal.error(err as string);
  }
}
async function dropWorker() {
  if (!$pool.value) return terminal.error("$pool not connected yet");
  try {
    await $pool.value.drop();
  } catch (err) {
    terminal.error(err as string);
  }
}
async function scaleWorker(n?: number) {
  if (!$pool.value) return terminal.error("$pool not connected yet");
  try {
    await $pool.value.scale(n);
  } catch (err) {
    terminal.error(err as string);
  }
}
async function fillWorkers() {
  if (!$pool.value) return terminal.error("$pool not connected yet");
  try {
    // await $pool.value.fill();
    let max = await $pool.value.capacity;
    while ((await $pool.value.length) < max) await $pool.value.spawn();
    terminal.success(`Filled pool to capacity with ${workers.value.length} runners.`);
  } catch (err) {
    terminal.error(err as string);
  }
}

// get the maximum capacity from pool
const nmax = ref(0);
const unwatch = watch(
  () => wasimoff.$pool,
  async (value) => {
    if (value) {
      // capacity is readonly and should only ever change once
      nmax.value = await value.capacity;
      unwatch();
    }
  },
);
</script>

<template>
  <!-- worker pool controls -->
  <div class="columns">
    <!-- form input for the number of workers -->
    <div class="column">
      <label class="label has-text-grey-dark">Worker Pool</label>
      <div class="field has-addons">
        <div class="control">
          <input
            class="input is-info"
            type="text"
            placeholder="Number of Workers"
            disabled
            :value="workers.length"
            @input="(ev) => scaleWorker((ev.target as HTMLInputElement).value as unknown as number)"
            style="width: 110px"
          /><!-- hotfix for type="number" input ... no problem with type="text" -->
        </div>
        <div class="control">
          <button
            class="button is-family-monospace is-info"
            @click="spawnWorker"
            :disabled="workers.length == nmax"
            title="Add a WASM Runner to the Pool"
          >
            +
          </button>
        </div>
        <div class="control">
          <button
            class="button is-family-monospace is-info"
            @click="dropWorker"
            :disabled="workers.length == 0"
            title="Remove a WASM Runner from the Pool"
          >
            -
          </button>
        </div>
        <!-- hidden for demo -->
        <!-- <div class="control">
          <button class="button is-info" @click="fillWorkers" :disabled="workers.length == nmax" title="Add WASM Runners to maximum capacity">Fill</button>
        </div> -->
      </div>

      <label class="label has-text-grey-dark">Busy Workers</label>
      <!-- TODO: visualization with capacity as maximum slots -->
      <div class="workerfarm" v-if="workers">
        <span v-for="(busy, i) of workers">
          <span class="workersquare" :class="busy ? 'busy' : ''">{{ i + 1 }}</span>
        </span>
      </div>
      <div v-if="workers.length === 0">You have no active Workers.</div>
    </div>

    <!-- connection status -->
    <div class="column">
      <label class="label has-text-grey-dark">Broker Transport</label>
      <div class="field has-addons">
        <div class="control">
          <input
            :readonly="connected"
            class="input"
            :class="connectionStatus"
            type="text"
            title="Broker URL"
            v-model="transport"
          />
        </div>
        <div class="control" v-if="!connected">
          <button class="button is-success" @click="connect" title="Connect Transport">
            Connect
          </button>
        </div>
        <div class="control" v-if="connected">
          <button
            class="button is-warning"
            @click="shutdown"
            title="Drain Workers and close the Transport gracefully"
          >
            Disconnect
          </button>
        </div>
        <!-- hidden for demo -->
        <!-- <div class="control" v-if="connected">
          <button class="button is-danger" @click="kill" title="Kill Workers and close Transport immediately">Kill</button>
        </div> -->
      </div>

      <label class="label has-text-grey-dark">Cluster Information</label>
      <InfoProviders></InfoProviders>
    </div>
  </div>
</template>

<style lang="css" scoped>
.workerfarm {
  max-width: calc((30px + 6px) * 8);
}

.workersquare {
  text-align: center;
  line-height: 30px;
  border-radius: 6px;
  margin: 0 6px 6px 0;
  width: 30px;
  height: 30px;
  display: inline-block;
  background-color: #ddd;
  transition: background-color 0.5s ease-in;
}

.workersquare.busy {
  background-color: rgb(27, 168, 27);
  transition: none;
}
</style>
