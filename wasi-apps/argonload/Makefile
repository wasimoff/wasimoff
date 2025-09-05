# probably shouldn't mix build environments (cargo + make) but it's a quicky way to an end ..

.PHONY: all native wasm static
all: native wasm
native: argonload
wasm: argonload.wasm
static: argonload.static

# copy binaries from build directories
argonload: target/release/argonload
	cp -f $< $@
argonload.wasm: target/wasm32-wasip1/release/argonload.opt.wasm
	cp -f $< $@
argonload.static: target/x86_64-unknown-linux-musl/release/argonload
	cp -f $< $@

# native binary
target/release/argonload: src/main.rs
	cargo build --release

# webassembly binary
target/wasm32-wasip1/release/argonload.wasm: src/main.rs
	cargo build --target wasm32-wasip1 --release

# optimize wasm binary
target/wasm32-wasip1/release/argonload.opt.wasm: target/wasm32-wasip1/release/argonload.wasm
	wasm-opt -Oz -c $< -o $@

# native for another target, e.g. static musl
target/%/release/argonload: src/main.rs
	cargo build --target $* --release
