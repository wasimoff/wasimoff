package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wasi.team/client"
	wasimoff "wasi.team/proto/v1"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var ( // config flags
	brokerUrl = "http://localhost:4080" // default broker base URL
	verbose   = false                   // be more verbose
	readstdin = false                   // read stdin for exec
	websock   = false                   // use websocket to send tasks
	rootfs    = ""                      // include a rootfs in exec
	trace     = false                   // enable tracing on task
)

var ( // command flags, pick one
	cmdUpload  = ""    // upload this file
	cmdExec    = false // execute cmdline in wasip1
	cmdRunTask = ""    // run prepared task json
	cmdPyodide = ""    // run python file
)

func init() {
	// get the Broker URL from env
	if url, ok := os.LookupEnv("BROKER"); ok {
		brokerUrl = strings.TrimRight(url, "/")
	}
}

var c client.WasimoffClient

func main() {

	// commandline parser
	flag.StringVar(&brokerUrl, "broker", brokerUrl, "URL to the Broker to use")
	flag.StringVar(&cmdUpload, "upload", "", "Upload a file (wasm or zip) to the Broker and receive its ref")
	flag.BoolVar(&cmdExec, "exec", false, "Execute an uploaded binary by passing all non-flag args")
	flag.StringVar(&cmdPyodide, "pyodide", "", "Run a Python script file with Pyodide")
	flag.StringVar(&cmdRunTask, "task", "", "Run a prepared JSON task file (either Wasip1 or Pyodide)")
	flag.BoolVar(&verbose, "verbose", verbose, "Be more verbose and print raw messages for -exec")
	flag.BoolVar(&readstdin, "stdin", readstdin, "Read and send stdin when using -exec (not streamed)")
	flag.BoolVar(&websock, "ws", websock, "Use a WebSocket to connect to Broker")
	flag.StringVar(&rootfs, "rootfs", rootfs, "Use a rootfs ZIP in -exec task")
	flag.BoolVar(&trace, "trace", trace, "Collect timestamps during task lifetime")
	flag.Parse()

	// establish a connection to the broker
	if websock {
		wc, err := client.NewWasimoffWebsocketClient(context.Background(), brokerUrl)
		if err != nil {
			log.Fatalf("ERR: can't connect to Broker: %s", err)
		} else {
			c = wc
		}
	} else {
		c = client.NewWasimoffConnectRpcClient(http.DefaultClient, brokerUrl)
	}

	switch true {

	// upload a file, optionally take another argument as name alias
	case cmdUpload != "":
		alias := flag.Arg(0)
		UploadFile(cmdUpload, alias)

	// execute an ad-hoc command, as if you were to run it locally
	case cmdExec:
		envs := []string{}
		args := flag.Args()
		Execute(args, envs)

	// execute a prepared JSON task
	case cmdRunTask != "":
		RunTaskFile(cmdRunTask)

	// execute a python script task
	case cmdPyodide != "":
		RunPythonScript(cmdPyodide)

	// no command specified
	default:
		fmt.Fprintln(os.Stderr, "ERR: at least one of -upload, -exec, -run, -runpy must be used")
		flag.Usage()
		os.Exit(2)
	}

}

// upload a local file to the Broker
func UploadFile(filename, name string) {

	// read the file
	buf, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal("reading file: ", err)
	}

	// reuse basename as name if it's empty
	if name == "" {
		name = filepath.Base(filename)
	}

	ref, err := c.Upload(buf, name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, ref)
	os.Exit(0)

}

// execute an ad-hoc command line
func Execute(args, envs []string) {
	if len(args) == 0 {
		log.Fatal("need at least one argument")
	}

	// prepare the request
	request := &wasimoff.Task_Wasip1_Request{
		Info: &wasimoff.Task_Metadata{},
		Params: &wasimoff.Task_Wasip1_Params{
			Binary: &wasimoff.File{Ref: proto.String(args[0])},
			Args:   args,
			Envs:   envs,
		},
	}

	// optionally read stdin
	if readstdin {
		stdin, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERR: failed reading stdin:", err)
			os.Exit(1)
		}
		request.Params.Stdin = stdin
	}

	// optionally add rootfs ref
	if rootfs != "" {
		request.Params.Rootfs = &wasimoff.File{Ref: &rootfs}
	}

	// optionally enable tracing
	if trace {
		request.Info.Trace = &wasimoff.Task_Trace{
			Created: proto.Int64(time.Now().UnixNano()),
		}
	}

	runWasip1(request)
}

func runWasip1(request *wasimoff.Task_Wasip1_Request) {

	// make the request
	maybeDumpJson("[RunWasip1] run:", request)
	response, err := c.RunWasip1(context.Background(), request)

	// check for errors
	if err != nil {
		fmt.Fprintf(os.Stderr, "[RunWasip1] ERR: %s\n", err.Error())
		os.Exit(1)
	}
	if response.GetError() != "" {
		fmt.Fprintf(os.Stderr, "[RunWasip1] FAIL: %s\n", response.GetError())
		os.Exit(1)
	}

	// print the result
	ok := response.GetOk()
	maybeDumpJson("[RunWasip1] result:", response)
	if len(ok.GetStderr()) != 0 {
		fmt.Fprintf(os.Stderr, "\033[31m%s\033[0m\n", string(ok.GetStderr()))
	}
	fmt.Fprintln(os.Stdout, string(ok.GetStdout()))
	maybePrintTrace(response.GetInfo())
	os.Exit(int(ok.GetStatus()))

}

// run a prepared task configuration from file
func RunTaskFile(config string) {

	// read the file
	buf, err := os.ReadFile(config)
	if err != nil {
		log.Fatal("reading file: ", err)
	}

	// decode with protojson and decide what to do based on embedded type
	var anymsg anypb.Any
	var anytask proto.Message
	if err = protojson.Unmarshal(buf, &anymsg); err != nil {
		log.Fatal("unmarshal anypb from JSON: ", err)
	}
	if anytask, err = anymsg.UnmarshalNew(); err != nil {
		log.Fatal("unmarshal request from anypb: ", err)
	}
	if verbose {
		log.Println("parsed file as:", anymsg.GetTypeUrl())
	}

	switch task := anytask.(type) {

	case *wasimoff.Task_Wasip1_Request:
		runWasip1(task)

	case *wasimoff.Task_Pyodide_Request:
		runPyodide(task)

	default:
		log.Fatal("this task type is not supported:")

	}

}

// run a python script from file
func RunPythonScript(script string) {

	// read the file
	buf, err := os.ReadFile(script)
	if err != nil {
		log.Fatal("reading file: ", err)
	}
	script = string(buf)

	// prepare a request using this file
	request := &wasimoff.Task_Pyodide_Request{
		Params: &wasimoff.Task_Pyodide_Params{
			Run: &wasimoff.Task_Pyodide_Params_Script{
				Script: script,
			},
		},
	}

	runPyodide(request)
}

func runPyodide(request *wasimoff.Task_Pyodide_Request) {

	// make the request
	maybeDumpJson("[RunPyodide] run:", request)
	response, err := c.RunPyodide(context.Background(), request)

	// check for errors
	if err != nil {
		fmt.Fprintf(os.Stderr, "[RunPyodide] ERR: %s\n", err.Error())
		os.Exit(1)
	}
	if response.GetError() != "" {
		fmt.Fprintf(os.Stderr, "[RunPyodide] FAIL: %s\n", response.GetError())
		os.Exit(1)
	}

	// print task result
	ok := response.GetOk()
	fmt.Fprintf(os.Stderr, "# Pyodide v%s: https://pyodide.org/en/%s/usage/packages-in-pyodide.html\n", ok.GetVersion(), ok.GetVersion())
	if len(ok.GetStderr()) != 0 {
		fmt.Fprintf(os.Stderr, "\033[31m%s\033[0m\n", string(ok.GetStderr()))
	}
	fmt.Fprintln(os.Stdout, string(ok.GetStdout()))
	if ok.Pickle != nil {
		fmt.Fprintf(os.Stderr, "\nresult pickle: %s\n", base64.StdEncoding.EncodeToString(ok.GetPickle()))
	}

}

func maybeDumpJson(pre string, m proto.Message) {
	if verbose {
		js, _ := protojson.Marshal(m)
		log.Println(pre, string(js))
	}
}

func maybePrintTrace(info *wasimoff.Task_Metadata) {
	if trace {

		// pop the trace to print separately
		trace := info.GetTrace()
		info.Trace = nil

		// perform span correction
		trace.ClockSkewCorrection()

		// print metadata and the list of steps
		fmt.Fprintf(os.Stderr, "\033[1mTask Metadata:\n\033[0;36m%s\033[0m\n", prototext.Format(info))
		fmt.Fprintf(os.Stderr, "\033[36mstart: %s\033[0m\n", time.Unix(0, *trace.Created))
		fmt.Fprintf(os.Stderr, "\033[36m%14s | %14s    | component\033[0m\n", "absolute", "relative")
		for i, ev := range trace.Events {

			// compute time diffs from start and to last
			fromStart := *ev.Unixnano - *trace.Created
			fromLast := fromStart
			if i > 0 {
				fromLast = *ev.Unixnano - *trace.Events[i-1].Unixnano
			}

			// make a colorful label depending on component
			label := ev.Event.String()
			switch ev.Event.Component() {
			case wasimoff.Task_TraceEvent_Component_Client:
				label = "\033[31m" + label
			case wasimoff.Task_TraceEvent_Component_Broker:
				label = "\033[32m- " + label
			case wasimoff.Task_TraceEvent_Component_Provider:
				label = "\033[33m--- " + label
			}

			fmt.Fprintf(os.Stderr, "\033[36m%14.3f | %14.3f ms | %v\033[0m\n",
				float64(fromStart)/1_000_000,
				float64(fromLast)/1_000_000,
				label)

		}

		info.Trace = trace
		maybeDumpJsonTo3(info)

	}
}

func maybeDumpJsonTo3(msg proto.Message) {

	// try to open fd and check if we can write to it
	if _, err := os.Stat("/dev/fd/3"); err != nil {
		return
	}
	fd := os.NewFile(3, "/dev/fd/3")
	if fd == nil {
		return
	}
	defer fd.Close()
	if _, err := fd.Write([]byte{}); err != nil {
		return
	}

	// dump the message into this filedescriptor
	fd.WriteString(protojson.Format(msg))

}
