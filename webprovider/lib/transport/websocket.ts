import {
  createRegistry,
  fromBinary,
  fromJsonString,
  toBinary,
  toJsonString,
} from "@bufbuild/protobuf";
import {
  type Envelope,
  Envelope_MessageType,
  EnvelopeSchema,
  file_proto_v1_messages,
  Subprotocol,
} from "@wasimoff/proto/v1/messages_pb";
import { Transmit, type Transport } from "./index";
import { PushableAsyncIterable } from "@wasimoff/func/pushableiterable";
import { Signal } from "@wasimoff/func/promises";

export class WebSocketTransport implements Transport {
  /** Connect using any known WebSocket subprotocol. */
  public static connect(url: string | URL): WebSocketTransport {
    let ws = new WebSocket(url, [
      // offer all known subprotocols on connection
      WebSocketTransport.provider_v1_protobuf,
      WebSocketTransport.provider_v1_json,
    ]);
    return new WebSocketTransport(ws);
  }

  /** Setup a Transport from an opened connection and wire up all the event listeners. */
  private constructor(private ws: WebSocket) {
    this.ws.binaryType = "arraybuffer";

    this.ws.addEventListener("open", () => {
      console.log(...prefix.open, "connection established", {
        url: this.ws.url,
        protocol: this.ws.protocol,
      });
      this.signal.resolve();
    });

    this.ws.addEventListener("error", (event) => {
      // per MDN: "fired when a connection [...] has been closed due to an error"
      console.error(...prefix.err, "connection closed due to an error", event);
    });

    this.ws.addEventListener("close", ({ code, reason, wasClean }) => {
      // TODO: implement reconnection handler without tearing everything down
      console.warn(...prefix.warn, `WebSocket connection closed:`, {
        code,
        reason,
        wasClean,
        url: this.ws.url,
      });
      this.close(reason, wasClean);
    });

    this.ws.addEventListener("message", ({ data }) => {
      try {
        let envelope = this.unmarshal(data);
        if (debugging && envelope.payload?.typeUrl !== "wasimoff/Throughput") {
          console.debug(
            ...prefix.rx,
            envelope.sequence,
            Envelope_MessageType[envelope.type],
            envelope.payload?.typeUrl,
            envelope.error,
          );
        }
        this.messages.push({ envelope });
      } catch (err) {
        console.error(...prefix.err, err);
        this.messages.push(Promise.reject(err));
      }
    });
  }

  /** explicit registry is needed for JSON marshal with custom type prefix */
  private readonly registry = createRegistry(file_proto_v1_messages);

  /** messages is an iterable of all incoming, already unmarshalled to Envelopes */
  public messages = new PushableAsyncIterable<Transmit>();

  /** send picks the correct codec depending on negotiated subprotocol and marshalls the envelope */
  public async send(transmit: Transmit): Promise<void> {
    this.closed.throwIfAborted();
    await this.signal.promise;
    if (debugging) {
      console.debug(
        ...prefix.tx,
        transmit.envelope.sequence,
        Envelope_MessageType[transmit.envelope.type],
        transmit.envelope.payload?.typeUrl,
        transmit.envelope.error,
      );
    }
    switch (this.ws.protocol) {
      case WebSocketTransport.provider_v1_protobuf:
        return this.ws.send(toBinary(EnvelopeSchema, transmit.envelope));

      case WebSocketTransport.provider_v1_json:
        return this.ws.send(
          toJsonString(EnvelopeSchema, transmit.envelope, { registry: this.registry }),
        );

      default: // oops?
        let err = WebSocketTransport.Err.ProtocolViolation.Negotiation(this.ws.protocol);
        this.close(err.message);
        throw err;
    }
  }

  /** unmarshal does just that and picks the correct codec based on negotiated subprotocol */
  private unmarshal(data: string | ArrayBuffer): Envelope {
    switch (this.ws.protocol) {
      case WebSocketTransport.provider_v1_protobuf:
        if (data instanceof ArrayBuffer) {
          return fromBinary(EnvelopeSchema, new Uint8Array(data));
        } else throw WebSocketTransport.Err.ProtocolViolation.MessageType("text", this.ws.protocol);

      case WebSocketTransport.provider_v1_json:
        if (typeof data === "string") {
          return fromJsonString(EnvelopeSchema, data, { registry: this.registry });
        } else {throw WebSocketTransport.Err.ProtocolViolation.MessageType(
            "binary",
            this.ws.protocol,
          );}

      default: // oops?
        let err = WebSocketTransport.Err.ProtocolViolation.Negotiation(this.ws.protocol);
        this.close(err.message);
        throw err;
    }
  }

  // signal to wait for readiness when sending
  private signal = Signal();
  public ready = this.signal.promise;

  // handle closure and cancellation
  private controller = new AbortController();
  public closed = this.controller.signal;

  public close(reason: string = "closed normally", wasClean: boolean = true) {
    // see https://www.rfc-editor.org/rfc/rfc6455.html#section-7.4 for defined status codes
    // but ws.close(code) can only be [ 1000, 3000..4999 ], so just use 1000 below
    let err = new WebSocketTransport.Err.TransportClosed(1000, reason, wasClean, this.ws.url);
    this.ws.close(1000, reason);
    this.messages.close();
    this.signal.reject(err);
    this.controller.abort(err);
  }
}

export namespace WebSocketTransport {
  // provide shorthands for the subprotocols as strings
  export const provider_v1_protobuf = Subprotocol[Subprotocol.wasimoff_provider_v1_protobuf];
  export const provider_v1_json = Subprotocol[Subprotocol.wasimoff_provider_v1_json];

  // define possible error classes statically
  // extend Errors for custom error names
  export namespace Err {
    // the underlying connection was closed
    export class TransportClosed extends Error {
      constructor(
        public code: number,
        public reason: string,
        public wasClean: boolean,
        public url: string,
      ) {
        super(`WebSocket closed: ${JSON.stringify({ code, reason })}`);
        this.name = this.constructor.name;
      }
    }

    // unsupported protocol on the wire
    export class ProtocolViolation extends Error {
      constructor(message: string, public protocol: string) {
        super(`${message}: ${protocol}`);
        this.name = this.constructor.name;
      }
      static Negotiation(p: string) {
        return new ProtocolViolation("unsupported protocol", p);
      }
      static MessageType(t: string, p: string) {
        return new ProtocolViolation(`wrong message type ${t} for protocol`, p);
      }
    }
  }
}

// enable console.logs in the "hot" path (tx/rx)?
const debugging = false;

// pretty console logging prefixes
const prefix = {
  open: ["%c[WebSocket]%c open", "color: skyblue;", "color: greenyellow;"],
  rx: ["%c[WebSocket]%c « Rx", "color: skyblue;", "color: blue;"],
  tx: ["%c[WebSocket]%c Tx »", "color: skyblue;", "color: greenyellow;"],
  err: ["%c[WebSocket]%c Error", "color: skyblue;", "color: firebrick;"],
  warn: ["%c[WebSocket]%c Warning", "color: skyblue;", "color: goldenrod;"],
};
