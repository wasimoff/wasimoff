import { type Envelope } from "@/proto/messages_pb";

/** Transport abstracts away an underlying network connection and marshalling
 * protocol to the Broker or another resource consumer. On the surface, it is a
 * simple interface to receive and send messages but it does not handle any
 * Request-Response semantics. It is possible to reconnect or migrate connections
 * transparently without breaking the message iterator. */
export interface Transport {

  // receive ordered messages from an iterable
  messages: AsyncIterable<Envelope>;

  // send messages with a simple function
  send: (envelope: Envelope) => Promise<void>;

  // signal a closed connection with an AbortController internally
  closed: AbortSignal;
  close: () => void;

}

export { WebSocketTransport } from "./websocket";
export { Messenger } from "./messenger";
