import { construct, releaseProxy, type WrappedWorker } from "./comlink";
import { type WasiWorker, type Wasip1TaskParams, Wasip1TaskResult, PyodideTaskParams, PyodideTaskResult } from "./wasiworker";
import { Queue } from "@wasimoff/func/queue";

// colorful console logging prefix
const logprefix = [ "%c[WasiWorkerPool]", "color: purple;" ] as const;

/** Worker threadpool, which dispatches tasks to WasmWorkers. */
export class WasiWorkerPool {

  constructor(
    /** The absolute maximum number of workers in this pool. */
    public readonly capacity: number = navigator.hardwareConcurrency,
  ) { };

  // hold the Workers in an array
  private pool: WrappedWorker<WasiWorker, {
    index: number,
    busy: boolean,
    taskid?: string,
    started?: Date,
    cancelled?: boolean,
    reject?: () => void,
  }>[] = [];

  /** Incrementing index for new workers. */
  private nextindex = 0;

  /** Get the number of Workers currently in the pool. */
  get length() { return this.pool.length; };

  /** Get a "bitmap" of busy workers. */
  get busy() { return this.pool.map(w => w.busy); };

  /** Get a list with information about current tasks. */
  get currentTasks() {
    return this.pool.map(w => ({
      index: w.index,
      busy: w.busy,
      task: w.taskid,
      started: w.started,
    }));
  };

  // an asynchronous queue to fetch an available worker
  private idlequeue = new Queue<typeof this.pool[0]>;


  // --------->  spawn new workers

  /** Add a new Worker to the pool. */
  async spawn() {
    // TODO: serialization for multiple async calls, e.g. call spawn twice with len=cap-1

    // check for maximum size
    // if (this.length >= this.capacity)
    //   throw "Maximum pool capacity reached!";

    // construct a new worker with comlink
    let index = this.nextindex++;
    console.info(...logprefix, "spawn Worker", index);
    const worker = new Worker(new URL("./wasiworker.ts", import.meta.url), { type: "module" });
    const link = await construct<typeof WasiWorker>(worker, index); // TODO: use Pyodide dist on Broker

    // append to pool and enqueue available for work
    const wrapped = { index, worker, link, busy: false };
    this.pool.push(wrapped);
    this.idlequeue.put(wrapped);
    return this.length;

  };

  /** Scale to a certain number of Workers is in the pool, clamped by `nmax`. */
  async scale(n: number = this.capacity) {
    n = this.clamped(n);
    if (this.length < n)
      while (this.length < n) await this.spawn();
    else
      while (this.length > n) await this.drop();
    return this.length;
  };

  // clamp a desired value to maximum number of workers
  private clamped(n?: number): number {
    if (n === undefined || n > this.capacity) return this.capacity;
    if (n <= 0) return 0;
    return n;
  };


  // --------->  terminate workers

  /** Stop a Worker gracefully and remove it from the pool. */
  async drop() {
    // exit early if pool is already empty
    if (this.length === 0) return this.length;
    // take an idle worker from the queue
    const worker = await this.idlequeue.get();
    // remove it from the pool and release resources
    this.pool.splice(this.pool.findIndex(el => el === worker), 1);
    console.info(...logprefix, "shutdown worker", worker.index);
    worker.link[releaseProxy]();
    worker.worker.terminate();
    return this.length;
  };

  /** Forcefully terminate all Workers and reset the queue. */
  async killall() {
    if (this.length === 0) return;
    console.warn(...logprefix, `killing all ${this.length} workers`);
    this.pool.forEach(w => {
      w.link[releaseProxy]();
      w.worker.terminate();
    });
    this.pool = [];
    this.idlequeue = new Queue();
    return this.length;
  };

  /** Cancel a running task. There's not really any good way of stopping an
   * execution once the WebAssembly module is started, so just terminate and
   * respawn the worker. */
  async cancel(taskid: string) {
    // find a worker executing this task id
    let w = this.pool.find(w => w.taskid === taskid);
    if (w !== undefined) {
      w.cancelled = true;
      console.warn(...logprefix, `cancel and respawn worker ${w.index}`);
      // terminate and remove from pool
      this.pool.splice(this.pool.findIndex(el => el === w), 1);
      w.link[releaseProxy]();
      w.worker.terminate();
      w.reject?.();
      // and respawn
      await this.spawn();
    };
  }


  // --------->  send tasks to workers

  /** The `run` method tries to get an idle worker from
   * the pool and executes a Wasi task on it. The `next` function is called
   * when a worker has been taken from the queue and before execution begins.
   * Afterwards, the method makes sure to put the worker back into the queue,
   * so *don't* keep any references to it around! The result of the computation
   * is finally returned to the caller in a Promise. */
  async runWasip1(id: string, task: Wasip1TaskParams): Promise<Wasip1TaskResult> {
    if (this.length === 0) throw new Error("no workers in pool");

    // take an idle worker from the queue
    const worker = await this.idlequeue.get();
    worker.busy = true;
    worker.taskid = id;
    worker.started = new Date();

    // try to execute the task and put worker back into queue
    try {
      // promise can be rejected if the task is cancelled
      return await new Promise<Wasip1TaskResult>((resolve, reject) => {
        worker.reject = reject;
        worker.link.runWasip1(id, task).then(resolve, reject);
      });
    } finally {
      // don't requeue if it's terminated
      if (worker.cancelled !== true) {
        worker.busy = false;
        worker.taskid = undefined;
        worker.started = undefined;
        await this.idlequeue.put(worker);
      };
    };

  };


  async runPyodide(id: string, task: PyodideTaskParams): Promise<PyodideTaskResult> {
    if (this.length === 0) throw new Error("no workers in pool");

    // take an idle worker from the queue
    const worker = await this.idlequeue.get();
    worker.busy = true;
    worker.taskid = id;
    worker.started = new Date();

    // try to execute the task and forcibly respawn afterwards due to memory leak
    // https://github.com/pyodide/pyodide/discussions/4338
    try {
      // promise can be rejected if the task is cancelled
      return await new Promise<PyodideTaskResult>((resolve, reject) => {
        worker.reject = reject;
        worker.link.runPyodide(id, task).then(resolve, reject);
      });
    } finally {
      // always respawn this worker
      console.log(...logprefix, `force worker ${worker.index} respawn`);
      worker.busy = false;
      worker.taskid = undefined;
      worker.started = undefined;
      worker.link[releaseProxy]();
      worker.worker.terminate();
      await this.spawn();
      // move splice last to avoid "no workers in pool" errors
      this.pool.splice(this.pool.findIndex(el => el === worker), 1);
    };

  };

}
