#!/usr/bin/env -S locust -f
# https://docs.locust.io/en/stable/quickstart.html


from locust import HttpUser, task, tag, events, stats

stats.PERCENTILES_TO_REPORT = [0.25, 0.50, 0.75, 0.80, 0.90, 0.95, 0.98, 0.99, 1.0]


@events.init_command_line_parser.add_listener
def arguments(parser):
    parser.add_argument(
        "--tsp-n",
        type=int,
        env_var="TSP_N",
        default=10,
        help="random set size for travelling_salesman",
    )


class WasimoffWorkload(HttpUser):

    # default url prefix of broker to load
    host = "http://localhost:4080"

    def __wasm(self, binary, argv, name=None):
        "Helper function to run a WebAssembly binary."
        return self.client.post(
            "/api/client/wasimoff.v1.Tasks/RunWasip1",
            json={
                "info": {"reference": "locust"},
                "params": {"binary": {"ref": binary}, "args": argv},
            },
            name=(
                name if name is not None else f"run/{binary}"
            ),  # group requests by binary
            catch_response=True,  # need to look into response
        )

    def wasm(self, binary, argv, name=None):
        "Catch common errors that might happen during an invocation."
        with self.__wasm(binary, argv, name) as response:
            if not response.ok:
                print(f"task not OK: {response.status_code} {response.reason}")
                return response.failure("not successful")
                # raise exception.RescheduleTask()
            if not '"status":0' in response.text:
                print(f"task not successful: {response.text}")
                return response.failure("not successful")

    @task(1)
    @tag("tsp")
    def task_tsp_rand(self):
        "Travelling Salesman Problem with `n` random cities."
        n = str(self.environment.parsed_options.tsp_n)
        self.wasm("tsp.wasm", ["tasp.wasm", "rand", str(n)], f"run/tsp({n})")
