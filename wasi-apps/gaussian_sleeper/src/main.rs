use rand_distr::{Distribution, Normal};
use std::env;
use std::time::{Duration, Instant};

fn main() {
    // Parse command line arguments
    let args: Vec<String> = env::args().collect();
    if args.len() != 3 {
        eprintln!(
            "Usage: {} <mean_sleep_time_ms> <standard_deviation_ms>",
            args[0]
        );
        std::process::exit(1);
    }

    let mean: f64 = args[1].parse().expect("mean must be a number");
    let std_dev: f64 = args[2].parse().expect("std_dev must be a number");

    // create a normal distribution
    let normal = Normal::new(mean, std_dev).expect("failed to instantiate normal");

    // generate a random sleep duration from the distribution
    let sleep_duration_ms = normal.sample(&mut rand::rng());
    let sleep_duration_ms = sleep_duration_ms.max(0.0); // Ensure non-negative

    println!(
        "spinning for {:.2} ms ~ N({:.2}, {:.2})",
        sleep_duration_ms, mean, std_dev
    );

    // Busy loop for the calculated duration
    let start = Instant::now();
    let target_duration = Duration::from_millis(sleep_duration_ms as u64);

    while start.elapsed() < target_duration {
        // Do some pointless work to load the CPU
        let _ = (0..1000).map(|x| x * x).sum::<i32>();
    }
}
