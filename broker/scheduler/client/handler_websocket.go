package client

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"wasimoff/broker/net/transport"
	"wasimoff/broker/provider"
	wasimoff "wasimoff/proto/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/prototext"
)

// ClientSocketHandler returns a http.HandlerFunc to be used on a route that shall serve
// as an endpoint for Clients to connect to. This particular handler uses WebSocket
// transport with either Protobuf or JSON encoding, negotiated using subprotocol strings.
// func ClientSocketHandler(rpc *WasimoffRPCServer) http.HandlerFunc {
func ClientSocketHandler(rpc *ConnectRpcServer) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		addr := transport.ProxiedAddr(r)

		// upgrade the transport
		// using wildcard in Allowed-Origins because Client can be anywhere
		wst, err := transport.UpgradeToWebSocketTransport(w, r, []string{"*"})
		if err != nil {
			log.Printf("[%s] New Client socket: upgrade failed: %s", addr, err)
			return
		}
		messenger := transport.NewMessengerInterface(wst)
		log.Printf("[%s] New Client socket", addr)

		// channel for finished requests
		// TODO: limit task creation with an equally-sized ticket channel
		done := make(chan *provider.AsyncTask, 32)

		defer log.Printf("[%s] Client socket closed", addr)
		for {
			select {

			// connection closing
			case <-r.Context().Done():
				return
			case <-messenger.Closing():
				return

			// print any received events
			case event, ok := <-messenger.Events():
				if !ok { // messenger closing
					return
				}
				log.Printf("{client %s} %s", addr, prototext.Format(event))

			// dispatch received requests
			case request, ok := <-messenger.Requests():
				if !ok { // messenger closing
					return
				}
				switch taskrequest := request.Request.(type) {

				case *wasimoff.Task_Wasip1_Request:
					go func(ctx context.Context, req transport.IncomingRequest, task *wasimoff.Task_Wasip1_Request) {
						r := connect.NewRequest(task)
						resp, err := rpc.RunWasip1(ctx, r)
						if err != nil {
							request.Respond(ctx, nil, err)
						} else {
							request.Respond(ctx, resp.Msg, nil)
						}
					}(r.Context(), request, taskrequest)
					continue

				case *wasimoff.Task_Pyodide_Request:
					go func(ctx context.Context, req transport.IncomingRequest, task *wasimoff.Task_Pyodide_Request) {
						r := connect.NewRequest(task)
						resp, err := rpc.RunPyodide(ctx, r)
						if err != nil {
							request.Respond(ctx, nil, err)
						} else {
							request.Respond(ctx, resp.Msg, nil)
						}
					}(r.Context(), request, taskrequest)
					continue

				default: // unexpected message type
					request.Respond(r.Context(), nil, fmt.Errorf("expecting only Task_*_Request messages on this socket"))
					continue

				}

			// respond with finished results
			case task := <-done:
				request, ok := task.Context.Value(ctxkeyRequest{}).(transport.IncomingRequest)
				if !ok {
					log.Fatalf("ClientSocketHandler: couldn't get incoming request from context")
				}

				// pass through both internal and response errors directly
				request.Respond(r.Context(), task.Response, task.Error)
				log.Printf("Task respond: %s :: %#v\n", task.Request.GetInfo().GetId(), prototext.Format(task.Response))

			}
		}

	}
}

// typed key to store original request in a context
type ctxkeyRequest struct{}
