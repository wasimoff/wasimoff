package config

// Prefix for envionment variable names, so HTTP_LISTEN becomes WASIMOFF_HTTP_LISTEN.
const envprefix = "WASIMOFF"

// Configuration via environment variables with github.com/kelseyhightower/envconfig.
type Configuration struct {

	// HTTP_LISTEN is the listening address for the HTTP server.
	HttpListen string `split_words:"true" default:"localhost:4080" desc:"Listening Addr for HTTP server"`

	// HTTP_CERT and HTTP_KEY are paths to a TLS keypair to optionally use for the HTTP server.
	// If none are given, a plaintext server is started. Reload keys with SIGHUP.
	HttpCert string `split_words:"true" desc:"Path to TLS certificate to use"`
	HttpKey  string `split_words:"true" desc:"Path to TLS key to use"`

	// ALLOWED_ORIGINS is a list of allowed Origin headers for transport connections.
	AllowedOrigins []string `split_words:"true" desc:"List of allowed Origins for WebSocket"`

	// STATIC_FILES is a path with static files to serve; usually the webprovider frontend dist.
	StaticFiles string `split_words:"true" default:"../webprovider/dist/" desc:"Serve static files on \"/\" from here"`

	// FILESTORAGE is a path to use for a persistent BoltDB database.
	// An empty string will use an ephemeral in-memory map[string]*File.
	FileStorage string `desc:"Use persistent BoltDB storage for files" default:":memory:"`

	// BENCHMODE activates a mode where the Broker produces infinite workload by itself
	Benchmode int `desc:"Benchmarking mode with n concurrent tasks" default:"0"`

	// METRICS will expose metrics for Prometheus via /metrics
	Metrics bool `desc:"Enable Prometheus exporter on /metrics" default:"false"`

	// DEBUG will enable the pprof handlers under /debug/pprof
	Debug bool `desc:"Enable profiling handlers on /debug/pprof" default:"false"`
}
