/// <reference lib="webworker" />
declare var self: DedicatedWorkerGlobalScope | SharedWorkerGlobalScope;
export {};

import { ProviderStorage } from "@wasimoff/storage/index";
import { Messenger, WebSocketTransport } from "@wasimoff/transport/index";
import { WasiWorkerPool } from "./workerpool";
import { create, Message } from "@bufbuild/protobuf";
import { Event_FileSystemUpdateSchema, Event_ProviderHelloSchema, Event_ProviderResourcesSchema }
  from "@wasimoff/proto/v1/messages_pb";
import { rpchandler } from "@wasimoff/worker/rpchandler";
import { expose, workerReady, transfer, proxy } from "./comlink";
import { Wasip1TaskParams } from "./wasiworker";

/**
 *     Wasimoff Provider
 * ----------------------------------
 * 
 * This is the "entrypoint" to connect to a Broker and get started serving requests.
 * Usage would depend a bit on what environment you're running in. Some variants may
 * be removed / become unsupported in the future, if they're not worth the added
 * complexity.
 * 
 * In Deno (or another terminal-based environment outside the web) you should just
 * instantiate the class directly in the main thread. The Messenger and Storage can
 * be initialized beforehand, amended with your own reconnection handlers and what
 * have you .. and then passed inside the constructor. The comlink won't be exposed
 * if the file is not running in a Worker scope.
 * 
 * In the Web, you should generally start this file in a Worker and let it handle
 * the connection and storage concerns. In that case, the Provider is controlled
 * through comlink and events (for UI updates etc.) should be streamed with an
 * AsyncIterable. To avoid overcommitting resources when opening multiple tabs
 * and, perhaps more importantly, to get the same view in all tabs, you would
 * generally prefer a SharedWorker for the Provider. **However,** in Chrome it
 * is *not* supported to spawn nested DedicatedWorkers within a SharedWorker
 * (https://issues.chromium.org/issues/40902676 and /40695450). On Android, Chrome
 * does not support SharedWorkers (https://issues.chromium.org/issues/40290702)
 * at all. Thus, the safest approach is to always spawn in a Worker and prevent
 * further Providers from starting with the Web Lock API (which is unavailable
 * on Deno, thus this isn't a completely universal approach either).
 * 
 */

export class WasimoffProvider {

  static readonly logprefix = [ `%c[Wasimoff Provider]`, "color: blue;" ];

  constructor(
    /** maximum number of workers in the pool */
    public readonly nmax = navigator.hardwareConcurrency,
    /** connection to the broker */
    public messenger?: Messenger,
    /** storage backend for modules and artifacts */
    public storage?: ProviderStorage,
  ) {
    console.info(...WasimoffProvider.logprefix, "started in", self.constructor.name);

    // setup the proxied pool to send concurrency updates automatically
    this.pool = new Proxy(new WasiWorkerPool(this.nmax), {
      // trap property accesses that return methods which can change the pool length
      get: (target, prop, receiver) => {
        const traps = ["spawn", "scale", "drop", "killall"];
        const method = Reflect.get(target, prop, receiver);
        // wrap the function calls with an update to the broker
        if (typeof method === "function" && traps.includes(prop as string)) {
          return async (...args: any[]) => {
            let result = await (method as any).apply(target, args) as Promise<number>;
            if (this.messenger !== undefined)
              this.sendInfo(await result).catch(() => { /* ignore errors */ });
            return result;
          };
        } else {
          // anything else is passed through
          return method;
        };
      },
    });

  };

  static async init(nmax: number, origin: string, dir: string, id?: string) {
    const p = new WasimoffProvider(nmax);

    // recheck the origin
    let url = new URL(origin);
    if (!/^https?:$/.test(url.protocol)) throw "origin should be https? url";

    // connect the storage
    await p.open(dir, url.origin);

    // replace protocol with websocket for transport
    url.protocol = url.protocol.replace("http", "ws");
    await p.connect(url.origin, id);

    return p;
  };


  // --------->  worker pool

  /** Return a comlink proxy of the worker pool. */
  async poolProxy() {
    return proxy(this.pool);
  }

  // hold the wasiworkers in a pool
  public pool: WasiWorkerPool;

  async run(id: string, task: Wasip1TaskParams) {
    return this.pool.runWasip1(id, task);
  };


  // --------->  file storage

  /** Return a comlink proxy of the storage. */
  public storageProxy() {
    if (!this.storage) throw "storage does not exist yet";
    return proxy(this.storage);
  };

  /** Open a storage by URL. */
  async open(directory: string, origin: string) {

    // can't close a filesystem yet
    if (this.storage !== undefined)
      throw "another storage is opened already";

    this.storage = new ProviderStorage(directory, origin);
    this.storage.updates.on(update => {
      if (this.messenger) this.messenger.sendEvent(create(Event_FileSystemUpdateSchema, update));
    });

  };


  // --------->  messenger connections

  /** Return a comlink proxy of the messenger. */
  async messengerProxy() {
    if (!this.messenger) throw "messenger does not exist yet";
    return proxy(this.messenger);
  };

  // (re)connect to a broker by url
  async connect(origin: string, id?: string) {

    // close previous connections
    if (this.messenger !== undefined && !this.messenger.closed.aborted) {
      this.messenger.close("reconnecting");
    };

    // only the websocket transport is implemented so far
    let url = new URL(origin);
    if (url.origin.match(/^wss?:$/) === null) url.protocol = url.protocol.replace("http", "ws");
    url.pathname = "/api/provider/ws";
    if (id !== undefined) url.searchParams.set("id", id);
    const wst = WebSocketTransport.connect(url.href);
    this.messenger = new Messenger(wst);
    await wst.ready;

    // send current concurrency with our useragent
    // TODO: this is sent automatically, even for Deno (which isn't "web")
    this.sendInfo(this.pool.length, "web", navigator.userAgent);

  };

  async disconnect() {
    if (this.messenger !== undefined && !this.messenger.closed.aborted) {
      this.messenger.close("bye");
    };
  };


  // --------->  handle rpc requests on messenger

  // bind the rpchandler function into this class
  private rpchandler = rpchandler.bind(this);

  /** Start handling RPC requests from the messenger. Await this method to be
   * notified when the connection closes because that will break the loop inside. */
  async handlerequests() {

    // storage must be opened already to register rpchandler
    if (this.storage === undefined)
      throw "need to open a storage first";

    // must have an open messenger on which to receive requests
    if (this.messenger === undefined || this.messenger.closed.aborted)
      throw "need to connect to a broker first";

    // this will loop until the messenger is closed
    for await (const request of this.messenger.requests) {
      request(request => this.rpchandler(request));
    };

  };

  /** Get a ReadableStream of the Events from the messenger. */
  async getEventstream() {

    // must have an open messenger on which to receive events
    if (this.messenger === undefined || this.messenger.closed.aborted)
      throw "need to connect to a broker first";

    // create a ReadableStream from the events iterable
    const iterator = this.messenger.events[Symbol.asyncIterator]()
    const stream = new ReadableStream<Message>({
      async pull(controller) {
        let { done, value } = await iterator.next();
        if (done) return controller.close();
        if (value) controller.enqueue(value);
      },
    });

    // transfer the stream
    // this doesn't work on safari :(
    return transfer(stream, [ stream ]);

  };

  async getEventIteratorNext() {

    // must have an open messenger on which to receive events
    if (this.messenger === undefined || this.messenger.closed.aborted)
      throw "need to connect to a broker first";

    // create an iterator and return a proxied next
    const iterator = this.messenger.events[Symbol.asyncIterator]()
    return proxy(() => iterator.next());

  }


  // --------->  shorthands to send events

  async sendInfo(pool?: number, name?: string, useragent?: string) {
    if (this.messenger === undefined) throw "not connected yet";
    if (pool !== undefined) {
      this.messenger.sendEvent(create(Event_ProviderResourcesSchema, { concurrency: pool }));
    };
    if (name !== undefined || useragent !== undefined) {
      this.messenger.sendEvent(create(Event_ProviderHelloSchema, { name, useragent }));
    };
  };




};


// detect if we're running in a worker and expose the comlink interface
if (typeof self !== "undefined"
  && self.constructor.name === "DedicatedWorkerGlobalScope"
  && self instanceof DedicatedWorkerGlobalScope) {

  // in a "normal" Worker
  // locks should be handled externally, before the Worker is even started
  console.log(...WasimoffProvider.logprefix, "new dedicated Worker started");
  expose(WasimoffProvider, self);
  self.postMessage(workerReady);

} else if (self.constructor.name === "SharedWorkerGlobalScope" && self instanceof SharedWorkerGlobalScope) {

  // in a SharedWorker, listen for connections 
  console.log(...WasimoffProvider.logprefix, "new SharedWorker started");
  self.addEventListener("connect", ({ ports }) => {
    console.log(...WasimoffProvider.logprefix, "new connection");
    const port = ports[0];
    expose(WasimoffProvider, port);
    port.postMessage(workerReady);
  });

  // TODO: proper connection manager?
  // search for `Tabulator` in sharedworker.ts, somewhere before 6f0cd00

}
