# tracebench

Run deterministic workloads against a Wasimoff broker. It can either use a function generator given
configured distributions or follow trace from a Huawei dataset.

```
go build
./tracebench -help
```

- Without `-broker ...` the tool runs in a dry-run mode, to check if your workload is as desired,
  without actually sending any tasks.
- Use `-wait` to wait for all in-flight tasks at the end of the configured duration. Otherwise the
  measurement is stopped abruptly and remaining tasks are cancelled.

The tool writes a file with event traces for each task in a varint-length encoded protobuf file
(like `tracebench-1765548497-309160.pb`). You can either compile the Protobuf message definitions
for your language of choice or convert it into a JSONL (each task is one JSON object per line):

```
go run ./proto2jsonl < tracebench.pb
```

### funcgen

Runs a YAML file with configured workloads. See `example_funcgen.yaml` for possible options.

```
./tracebench -funcgen workloads.yaml -wait -broker http://localhost:4080/
```

### tracer

This mode uses the FaaS traces from
[Huawei's 2023 data release from their private cloud](https://github.com/sir-lab/data-release/). Use
the downloader in `dataset/download.py` to retrieve the necessary data files and configure the
workloads as shown in `example_csvtrace.yaml`. You may use `dataset/viewer.py` or the original
authors' Notebooks to find suitable function columns for your use-case.

```
./tracebench -trace huawei.yaml -wait -broker http://localhost:4080/
```
