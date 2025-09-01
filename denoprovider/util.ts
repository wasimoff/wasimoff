import { WasiWorkerPool } from "@wasimoff/worker/workerpool.ts";

// log the currently running tasks
export function currentTasks(pool: WasiWorkerPool) {
  const now = new Date().getTime(); // unix epoch milliseconds
  return pool.currentTasks
    .filter((w) => w.busy)
    .map((w) => ({
      worker: w.index, // worker index
      task: w.task, // task ID
      started: w.started, // absolute start date
      age: w.started ? (now - w.started.getTime()) / 1000 : undefined, // age in seconds
    }));
}

// signal handler for clean exits
export class Terminator {
  // force immediate exit on second signal
  private forcequit = false;

  constructor(
    private pool: WasiWorkerPool,
    private readonly grace = 30_000,
    private beforeexit?: (forced: boolean) => void | Promise<void>,
  ) {
    // handle SIGTERM (15) and SIGINT (2) the same
    const terminator = () => {
      this.terminate();
    };
    Deno.addSignalListener("SIGTERM", terminator);
    Deno.addSignalListener("SIGINT", terminator);
  }

  public async terminate() {
    // called a second time, force quit
    if (this.forcequit) {
      console.error(" kill");
      console.debug("aborted tasks:", currentTasks(this.pool));
      if (this.beforeexit) await this.beforeexit(true);
      Deno.exit(1);
    }

    // print acknowledgement and schedule grace timeout
    console.log(" shutdown (send signal again to force immediate exit)");
    this.forcequit = true;
    if (this.grace > 0) setTimeout(this.terminate, this.grace);

    // wait to finish all current tasks, then quit
    await this.pool.scale(0);
    await new Promise((r) => setTimeout(r, 50)); // ~ flush messages
    if (this.beforeexit) await this.beforeexit(false);
    Deno.exit(0);
  }
}
