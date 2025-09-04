# probably shouldn't mix build environments (cargo + make) but it's a quicky way to an end ..

.PHONY: all native wasm static
all: native wasm
native: gaussian_sleeper
wasm: gaussian_sleeper.wasm
static: gaussian_sleeper.static

# copy binaries from build directories
gaussian_sleeper: target/release/gaussian_sleeper
	cp -f $< $@
gaussian_sleeper.wasm: target/wasm32-wasip1/release/gaussian_sleeper.opt.wasm
	cp -f $< $@
gaussian_sleeper.static: target/x86_64-unknown-linux-musl/release/gaussian_sleeper
	cp -f $< $@

# native binary
target/release/gaussian_sleeper: src/main.rs
	cargo build --release

# webassembly binary
target/wasm32-wasip1/release/gaussian_sleeper.wasm: src/main.rs
	cargo build --target wasm32-wasip1 --release

# optimize wasm binary
target/wasm32-wasip1/release/gaussian_sleeper.opt.wasm: target/wasm32-wasip1/release/gaussian_sleeper.wasm
	wasm-opt -Oz -c $< -o $@

# native for another target, e.g. static musl
target/%/release/gaussian_sleeper: src/main.rs
	cargo build --target $* --release
