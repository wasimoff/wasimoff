/// Simple Rust binary implementing the [Travelling Salesman Problem (TSP)](https://en.wikipedia.org/wiki/Travelling_salesman_problem)
/// using the deterministic brute-force algorithm from the [`travelling_salesman` crate](https://docs.rs/travelling_salesman/latest/travelling_salesman/).
mod datasets;
use datasets::{xy, City, CityConst, CityRecord, SGB128, WG59};
use time::Duration;

fn main() {
    // collect commandline arguments
    let args: Vec<String> = std::env::args().collect();

    // tsp write [n] – print a random CSV for later
    if args.len() == 3 && (args[1] == "write" || args[1] == "write_wg59") {
        let n = args[2].parse::<usize>().unwrap();
        if n > WG59.len() {
            eprintln!("n is larger than the number of cities in WG59 dataset");
            std::process::exit(1);
        }
        return write(&random_slice(&WG59, n));
    }

    // tsp write [n] – print a random CSV for later
    if args.len() == 3 && (args[1] == "write_sgb128") {
        let n = args[2].parse::<usize>().unwrap();
        if n > SGB128.len() {
            eprintln!("n is larger than the number of cities in SGB128 dataset");
            std::process::exit(1);
        }
        return write(&random_slice(&SGB128, n));
    }

    // tsp rand [n] – brute-solve a random selection
    if args.len() == 3 && args[1] == "rand" {
        let n = args[2].parse::<usize>().unwrap();
        return solve(&random_slice(&WG59, n));
    }

    // tsp read – read a previously generated CSV and brute-solve it
    if args.len() == 2 && (args[1] == "read" || args[1] == "brute") {
        return solve(&read());
    }

    // tsp anneal [t] - read a CSV and run simulated annealing for t seconds
    if args.len() == 3 && args[1] == "anneal" {
        let t = args[2].parse::<i64>().unwrap();
        let cities = read();
        return anneal(&cities, Duration::seconds(t));
    }

    // tsp hill [t] - read a CSV and run hill climbing for t seconds
    if args.len() == 3 && args[1] == "hill" {
        let t = args[2].parse::<i64>().unwrap();
        let cities = read();
        return hillclimb(&cities, Duration::seconds(t));
    }

    // unknown or missing arguments
    eprintln!("unknown arguments! tsp {{ write_{{wg59,sgb128}} [n] | rand [n] | brute | anneal [t] | hill [t] }}");
    std::process::exit(1);
}

/// Read in a CSV file with `x,y,name` and run the `travelling_salesman` solver.
fn read() -> Vec<City> {
    // read from stdin
    let mut reader = csv::ReaderBuilder::new()
        .has_headers(false)
        .from_reader(std::io::stdin());
    // collect cities from csv reader
    let mut cities: Vec<City> = vec![];
    for result in reader.deserialize() {
        let r: CityRecord = result.unwrap();
        cities.push(((r.x, r.y), r.name));
    }
    if cities.is_empty() {
        eprintln!("failed to read any cities from stdin!");
        std::process::exit(1);
    }
    cities
}

/// Generate a CSV file with `x,y,name` for later consumption from the given slice.
fn write(cities: &[City]) {
    // open writer on stdout
    let mut writer = csv::WriterBuilder::new()
        .has_headers(false)
        .from_writer(std::io::stdout());
    // iterate over cities in slice
    for city in cities {
        writer
            .serialize(CityRecord {
                x: city.0 .0,
                y: city.0 .1,
                name: city.1.clone(),
            })
            .unwrap();
    }
    writer.flush().unwrap();
}

/// Run the `travelling_salesman::brute_force` algorithm on the chosen slice of cities.
fn solve(cities: &[City]) {
    // find the optimal path
    let path = travelling_salesman::brute_force::solve(&xy(cities));
    print_distance(cities, &path);
}

/// Run the `travelling_salesman::simulated_annealing` algorithm on the chosen slice of cities.
fn anneal(cities: &[City], time: Duration) {
    // optimize the path with simulated annealing
    let path = travelling_salesman::simulated_annealing::solve(&xy(cities), time);
    print_distance(cities, &path);
}

/// Run the `travelling_salesman::hill_climbing` algorithm on the chosen slice of cities.
fn hillclimb(cities: &[City], time: Duration) {
    // optimize the path with simple hill-climbing without random restarts
    let path = travelling_salesman::hill_climbing::solve(&xy(cities), time);
    print_distance(cities, &path);
}

/// Print the found path
fn print_distance(cities: &[City], path: &travelling_salesman::Tour) {
    // map the path to city names
    let names: Vec<String> = path.route.iter().map(|c| cities[*c].1.clone()).collect();
    // print result
    println!("distance = {}", path.distance);
    eprintln!("path = {:?}", names);
}

/// Pick a random selection of coordinates from a `(x,y): [(f64, f64)]` dataset.
fn random_slice(slice: &[CityConst], amount: usize) -> Vec<City> {
    use rand::seq::SliceRandom;
    use rand::thread_rng;
    let mut copy = slice.to_vec();
    let slice = copy.partial_shuffle(&mut thread_rng(), amount).0;
    slice
        .to_vec()
        .iter()
        .map(|r| (r.0, r.1.to_string()))
        .collect()
}
