import { Transmit, Transport } from "./index";
import {
  Envelope_MessageType as MessageType,
  EnvelopeSchema,
  Event_ProviderResourcesSchema,
  file_proto_v1_messages,
} from "@wasimoff/proto/v1/messages_pb";
import * as nats_core from "@nats-io/nats-core";
import { PushableAsyncIterable } from "@wasimoff/func/pushableiterable";
import { createRegistry, fromBinary, isMessage, toBinary } from "@bufbuild/protobuf";
import { Signal } from "@wasimoff/func/promises";
import { anyUnpack } from "@bufbuild/protobuf/wkt";

interface SdpMessage {
  source: string;
  destination: string;
  msg: Sdp;
}

type Sdp =
  | { Offer: RTCSessionDescriptionInit }
  | { Answer: RTCSessionDescriptionInit }
  | {
      Candidate: {
        candidate: string;
        sdpMLineIndex: number | null;
        sdpMid: string | null;
        usernameFragment: string | null;
      };
    };

interface ProviderAnnounce {
  id: string;
  concurrency: number;
}

export class WebRTCTransport implements Transport {
  public messages = new PushableAsyncIterable<Transmit>();
  private peerConnections: Map<string, RTCPeerConnection> = new Map();
  private dataChannels: Map<string, RTCDataChannel> = new Map();
  private connectionStates: Map<string, string> = new Map();
  private nc: nats_core.NatsConnection;
  private id: string;
  private iceServers: RTCIceServer[];
  private announceMsg?: ProviderAnnounce;
  private readonly registry = createRegistry(file_proto_v1_messages);

  private constructor(
    natsConnection: nats_core.NatsConnection,
    id: string,
    announceInterval: number,
  ) {
    this.iceServers = [{ urls: "stun:stun.l.google.com:19302" }];
    this.id = id;
    this.nc = natsConnection;
    // subscribe to sdp channel
    const sub = this.nc.subscribe("sdp");
    (async () => {
      for await (const m of sub) {
        await this.handleSdpMessages(m);
      }
    })();

    setInterval(() => {
      this.announce();
    }, announceInterval * 1000);
    this.signal.resolve();
  }

  public static async connect(url: URL): Promise<WebRTCTransport> {
    // Parse and validate URL parameters
    const id = url.searchParams.get("id");
    if (!id) {
      throw "Missing required parameter: id";
    }

    const announceParam = url.searchParams.get("announce");
    if (!announceParam) {
      throw "Missing required parameter: announce";
    }

    const announceInterval = parseInt(announceParam, 10);
    if (isNaN(announceInterval) || announceInterval <= 0) {
      throw "Invalid announce parameter: must be a positive number";
    }

    const con_opts = { servers: url.origin };
    console.info("connecting to nats server", con_opts);
    let nc = await nats_core.wsconnect(con_opts);
    console.info(`connected to ${nc.getServer()}`);
    return new WebRTCTransport(nc, id, announceInterval);
  }

  async send(transmit: Transmit): Promise<void> {
    if (transmit.identifier) {
      const dataChannel = this.dataChannels.get(transmit.identifier);
      if (dataChannel && dataChannel.readyState === "open") {
        dataChannel.send(toBinary(EnvelopeSchema, transmit.envelope));
      } else {
        console.error("No open data channel for address", transmit.identifier);
      }
    } else {
      if (transmit.envelope.type === MessageType.Event) {
        if (transmit.envelope.payload === undefined) throw "cannot unpack empty payload";
        let payloadMessage = anyUnpack(transmit.envelope.payload, this.registry);
        if (payloadMessage === undefined) throw "unknown payload type";
        if (isMessage(payloadMessage, Event_ProviderResourcesSchema)) {
          let { concurrency } = payloadMessage;
          this.announceMsg = { id: this.id, concurrency };
          this.announce();
          return;
        }
      }
      console.warn("Cannot multiplex transmit with empty identifier", transmit);
    }
  }

  private announce(): void {
    if (this.announceMsg) {
      this.nc.publish("providers", JSON.stringify(this.announceMsg));
    }
  }

  private async handleSdpMessages(m: nats_core.Msg) {
    console.debug(new TextDecoder().decode(m.data));

    const sdpMessage: SdpMessage = JSON.parse(new TextDecoder().decode(m.data));

    if (sdpMessage.destination === this.id) {
      if ("Offer" in sdpMessage.msg) {
        const sdpOffer = sdpMessage.msg.Offer;

        const peerConnection = new RTCPeerConnection({
          iceServers: this.iceServers,
        });
        this.peerConnections.set(sdpMessage.source, peerConnection);

        await peerConnection.setRemoteDescription(sdpOffer);
        const answer = await peerConnection.createAnswer();
        await peerConnection.setLocalDescription(answer);

        const answerMessage: SdpMessage = {
          source: this.id,
          destination: sdpMessage.source,
          msg: { Answer: answer },
        };

        peerConnection.onicecandidate = (event) => {
          console.debug("ice candidate event", event.candidate);
          if (event.candidate !== null && event.candidate.candidate !== "") {
            const candidateData = {
              candidate: event.candidate.candidate,
              sdpMLineIndex: event.candidate.sdpMLineIndex,
              sdpMid: event.candidate.sdpMid,
              usernameFragment: event.candidate.usernameFragment,
            };
            // this is necessary because webrtc-polyfill is not spec compliant
            if (candidateData.candidate.startsWith("a=")) {
              candidateData.candidate = candidateData.candidate.substring(2);
            }
            // Skip candidates with masked .local addresses
            if (candidateData.candidate.includes(".local")) {
              console.debug("Skipping masked .local candidate:", candidateData.candidate);
              return;
            }
            const iceMessage: SdpMessage = {
              source: this.id,
              destination: sdpMessage.source,
              msg: { Candidate: candidateData },
            };
            this.nc.publish("sdp", JSON.stringify(iceMessage));
          }
        };

        // Set up data channel event handlers
        peerConnection.ondatachannel = (event) => {
          const dataChannel = event.channel;
          dataChannel.binaryType = "arraybuffer";
          if (dataChannel.label === "wasimoff") {
            this.dataChannels.set(sdpMessage.source, dataChannel);

            dataChannel.onopen = () => {
              this.connectionStates.set(sdpMessage.source, "connected");
              console.info("Data channel opened for:", sdpMessage.source);
            };

            dataChannel.onclose = () => {
              this.connectionStates.set(sdpMessage.source, "disconnected");
              console.info("Data channel closed for:", sdpMessage.source);
            };

            dataChannel.onerror = (error) => {
              console.error("Data channel error for:", sdpMessage.source, error);
            };
            dataChannel.onmessage = async (event) => {
              // Receive Envelope and push together with the source identifier to the message queue
              console.debug!("Received data channel data", event.data);
              const data = event.data;
              let array: Uint8Array;
              if (data instanceof Blob) array = await data.bytes();
              else array = new Uint8Array(data);
              const envelope = fromBinary(EnvelopeSchema, array);
              const transmit: Transmit = {
                envelope,
                identifier: sdpMessage.source,
              };
              this.messages.push(transmit);
            };
          }
        };

        // Publish the answer back via NATS
        this.nc.publish("sdp", JSON.stringify(answerMessage));
      } else if ("Candidate" in sdpMessage.msg) {
        const sdpCandidate = sdpMessage.msg.Candidate;
        const peerConnection = this.peerConnections.get(sdpMessage.source);
        if (peerConnection) {
          const candidate = new RTCIceCandidate({
            candidate: sdpCandidate.candidate,
            sdpMLineIndex: sdpCandidate.sdpMLineIndex,
            sdpMid: sdpCandidate.sdpMid,
            usernameFragment: sdpCandidate.usernameFragment,
          });
          await peerConnection.addIceCandidate(candidate);
        } else {
          console.warn("No peer connection found for source:", sdpMessage.source);
        }
      }
    }
  }

  // signal to wait for readiness when sending
  private signal = Signal();
  public ready = this.signal.promise;

  // handle closure and cancellation
  private controller = new AbortController();
  public closed = this.controller.signal;

  public close(reason: string = "closed normally", _: boolean = true) {
    let err = new Error(`WebRTC transport closed: ${reason}`);
    // Close all peer connections
    for (const [_, peerConnection] of this.peerConnections) {
      peerConnection.close();
    }
    this.peerConnections.clear();
    this.dataChannels.clear();
    this.connectionStates.clear();

    // Close NATS connection
    this.nc.close().catch((err) => console.error("Error closing NATS connection:", err));

    this.messages.close();
    this.signal.reject(err);
    this.controller.abort(err);
  }
}

// timeout for connections?
