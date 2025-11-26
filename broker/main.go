package main

import (
	"log"
	"net/http"
	"os"

	"wasi.team/broker/config"
	"wasi.team/broker/net/server"
	"wasi.team/broker/provider"
	"wasi.team/broker/scheduler"
	"wasi.team/broker/scheduler/client"
	"wasi.team/proto/v1/wasimoffv1connect"
)

func main() {
	printBanner()
	printVersion()

	// use configuration from environment variables
	conf := config.GetConfiguration()
	log.Printf("%#v", &conf)

	// create a new http server for the broker
	mux := http.NewServeMux()
	broker, err := server.NewServer(mux, conf.HttpListen, conf.HttpCert, conf.HttpKey)
	if err != nil {
		log.Fatalf("failed to start server: %s", err)
	}

	// create a provider store and scheduler
	store, err := provider.NewProviderStore(conf.FileStorage, &conf)
	if err != nil {
		log.Fatalf("failed to create provider store: %s", err)
	}
	selector := scheduler.NewSimpleMatchSelector(store)
	// selector := scheduler.NewRoundRobinSelector(store)
	// selector := scheduler.NewAnyFreeSelector(store)

	// provider endpoint
	mux.HandleFunc("GET /api/provider/ws", provider.WebSocketHandler(store, conf.AllowedOrigins))
	log.Printf("Provider socket: %s/api/provider/ws", broker.Addr())

	// create a queue for the tasks and start the dispatcher
	go scheduler.Dispatcher(store, selector, 32)

	// maybe start the "benchmode" load generation
	go client.BenchmodeTspFlood(store, conf.Benchmode)

	// client endpoints
	rpc := &client.ConnectRpcServer{Store: store}
	// -- websocket
	mux.HandleFunc("GET /api/client/ws", client.ClientSocketHandler(rpc))
	log.Printf("Client socket: %s/api/client/ws", broker.Addr())
	// -- connectrpc
	path, handler := wasimoffv1connect.NewTasksHandler(rpc)
	mux.Handle("/api/client"+path, http.StripPrefix("/api/client", handler))
	log.Printf("Client RPC: %s%s", broker.Addr(), "/api/client"+path)
	// -- plain http
	mux.Handle("/api/client/run/{wasm}", client.HttpExecWasip1Handler(rpc))
	log.Printf("Client HTTP: %s%s", broker.Addr(), "/api/client/run/{wasm}")

	// storage: serve files from and upload into store storage
	mux.Handle("GET /api/storage/{filename}", store.Storage)
	mux.HandleFunc("POST /api/storage/upload", store.Storage.Upload())
	log.Printf("Upload at %s/api/storage/upload", broker.Addr())

	// health and version message
	mux.HandleFunc("GET /healthz", server.Healthz())
	mux.HandleFunc("GET /api/version", server.Version())

	// pprof endpoint for debugging
	if conf.Debug {
		mux.Handle("GET /debug/pprof/", server.Profiling())
		log.Printf("DEBUG: broker PID is %d", os.Getpid())
		log.Printf("DEBUG: pprof profiles at %s/debug/pprof", broker.Addr())
	}

	// prometheus metrics
	if conf.Metrics {
		mux.Handle("/metrics", server.Prometheus())
		log.Printf("Prometheus metrics: %s/metrics", broker.Addr())
	}

	// serve static files for frontend
	mux.Handle("/", http.FileServer(http.Dir(conf.StaticFiles)))

	// start listening http server
	log.Printf("Broker listening on %s", broker.Addr())
	if err := broker.ListenAndServe(); err != nil {
		log.Fatalf("oops: %s", err)
	}

}

//
// ---
