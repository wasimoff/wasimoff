# argonload

This is a simple application using the `argon2` crate to compute random password hashes with given
parameters for memory size and iteration count. It can be used to create a somewhat deterministric
load, which depends on the hardware performance and scales about linearly with increased parameters.

```
$ wasmtime ./argonload.wasm
argon2id { mem: 32, iter: 20, p: 1 }
finished in 0.53 seconds: 31e4659da35f6459c87c992834bbeacb5cc1cd22f4774c9b9a9fac47703480c3
```

Observe this hyperfine benchmark, for example:

```
$ hyperfine -L i 10,20,50,100,200,1000 "./argonload -i {i}"
Benchmark 1: ./argonload -i 10
  Time (mean ± σ):     169.9 ms ±  24.6 ms    [User: 155.1 ms, System: 13.6 ms]
  Range (min … max):   142.1 ms … 213.6 ms    15 runs

Benchmark 2: ./argonload -i 20
  Time (mean ± σ):     310.0 ms ±  15.6 ms    [User: 298.1 ms, System: 11.6 ms]
  Range (min … max):   280.6 ms … 340.9 ms    10 runs

Benchmark 3: ./argonload -i 50
  Time (mean ± σ):     745.0 ms ±  38.7 ms    [User: 731.0 ms, System: 13.1 ms]
  Range (min … max):   673.9 ms … 803.4 ms    10 runs

Benchmark 4: ./argonload -i 100
  Time (mean ± σ):      1.459 s ±  0.071 s    [User: 1.444 s, System: 0.014 s]
  Range (min … max):    1.370 s …  1.604 s    10 runs

Benchmark 5: ./argonload -i 200
  Time (mean ± σ):      3.032 s ±  0.179 s    [User: 3.021 s, System: 0.010 s]
  Range (min … max):    2.746 s …  3.373 s    10 runs

Benchmark 6: ./argonload -i 1000
  Time (mean ± σ):     16.212 s ±  0.573 s    [User: 16.200 s, System: 0.012 s]
  Range (min … max):   15.062 s … 17.171 s    10 runs

Summary
  ./argonload -i 10 ran
    1.82 ± 0.28 times faster than ./argonload -i 20
    4.38 ± 0.68 times faster than ./argonload -i 50
    8.59 ± 1.31 times faster than ./argonload -i 100
   17.84 ± 2.79 times faster than ./argonload -i 200
   95.42 ± 14.23 times faster than ./argonload -i 1000
```
