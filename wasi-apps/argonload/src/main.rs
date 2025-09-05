use argon2::{Algorithm, Argon2, Params, Version};
use clap::Parser;
use rand_core::{OsRng, TryRngCore};
use std::time::Instant;

/// A tool to generate predictable CPU/memory load using Argon2 hashing
#[derive(Parser, Debug)]
#[command(about, long_about = None)]
struct Args {
    /// memory size in MiB
    #[arg(short, long, default_value_t = 32)]
    memory: u32,

    /// number of iterations
    #[arg(short, long, default_value_t = 20)]
    iterations: u32,

    /// degree of parallelism
    #[arg(short, long, default_value_t = 1)]
    parallelism: u32,

    /// do NOT randomize salt
    #[arg(long, default_value_t = false)]
    no_randomize: bool,
}

fn main() {
    // use a panic hook to avoid unclean exit
    std::panic::set_hook(Box::new(|panic_info| {
        eprintln!("ERR: {}", panic_info);
        std::process::exit(1);
    }));

    // parse commandline arguments
    let args = Args::parse();

    // password is hardcoded, we don't care about it
    let password = b"hunter42";
    let mut output = [0u8; 32];
    let mut salt = [0u8; 32];
    if !args.no_randomize {
        OsRng
            .try_fill_bytes(&mut salt)
            .expect("failed reading system randomness");
    }

    // instantiate argon2 with parameters from arguments
    let params = Params::new(args.memory * 1024, args.iterations, args.parallelism, None)
        .expect("invalid argon2 parameters");
    let argon = Argon2::new(Algorithm::Argon2id, Version::V0x13, params);

    println!(
        "argon2id {{ mem: {}, iter: {}, p: {} }}",
        args.memory, args.iterations, args.parallelism,
    );

    let start_time = Instant::now();
    argon
        .hash_password_into(password, &salt, &mut output)
        .unwrap();
    println!(
        "finished in {:.2} seconds: {}",
        start_time.elapsed().as_secs_f64(),
        hex::encode(output),
    );
}
