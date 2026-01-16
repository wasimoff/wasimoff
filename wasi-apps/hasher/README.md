# Hasher

This is a toy example that could be used for a crypto coin, where you need a
"proof-of-work". It takes a `-message <string>` and then appends an incrementing
counter, until the resulting SHA256 hash of the concatenation has `-zeros n`
leading zeros in its hexadecimal representation.

When using a cryptographically-secure hash function, it's impossible to predict
its output without computing it so you effectively need to brute-force it and
just try random (or in this case incrementing) values, until you've found a
suitable match.

I realize that this is not an ideal "workload" example, as the computation can
take a very long time depending on your input string and the desired number of
leading zeros. It is, after all, random chance. Anyway:

```bash
$ go build
$ ./hasher
2024/09/05 15:04:42 use a positive integer for -zeros

$ ./hasher -zeros 6
Searching hash for "Hello, World!" with 6 leading zeros ...
0 .. 1000000 .. 2000000 .. 3000000 .. 4000000 .. 5000000 .. 6000000 .. 7000000 .. 8000000 .. 9000000 .. 9057462!
Hello, World!|9057462 => 000000d652ae8447b4a580044447f915389facb3c3975ecb74125892eb6e6bc1

$ ./hasher -zeros 6 -message "Something else."
Searching hash for "Something else." with 6 leading zeros ...
0 .. 1000000 .. 2000000 .. 3000000 .. 4000000 .. 5000000 .. 6000000 .. 7000000 .. 8000000 .. 9000000 .. 10000000 .. 11000000 .. 12000000 .. 13000000 .. 14000000 .. 15000000 .. 16000000 .. 17000000 .. 18000000 .. 19000000 .. 19183301!
Something else.|19183301 => 000000a585dcc19a8dac9a57092c60cdff80e37f4bd6592b02a67c62ed64c667
```
