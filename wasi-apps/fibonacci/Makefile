# probably shouldn't mix build environments (cargo + make) but it's a quicky way to an end ..

exe.wasm: target/wasm32-wasip1/release/wasi-executable.wasm
	wasm-opt -Oz -c $< -o $@

target/wasm32-wasip1/release/wasi-executable.wasm: src/main.rs
	cargo build --target wasm32-wasip1 --release

target/release/wasi-executable: src/main.rs
	cargo build --release

.PHONY: run-wasm run-native
run-wasm: exe.wasm
	wasmtime run --dir ./ -- $< 42

run-native: target/release/wasi-executable
	./$< 42
