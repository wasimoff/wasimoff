import { Transmit, Transport } from "./index";
import {
  Envelope_MessageType as MessageType,
  EnvelopeSchema,
  Event_ProviderResourcesSchema,
  file_proto_v1_messages,
} from "@wasimoff/proto/v1/messages_pb";
import * as nats_core from "@nats-io/nats-core";
import { PushableAsyncIterable } from "@wasimoff/func/pushableiterable";
import {
  create,
  createRegistry,
  fromBinary,
  isMessage,
  Message as ProtoMessage,
  toBinary,
} from "@bufbuild/protobuf";
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
  private natsUrl: string;
  private nc?: nats_core.NatsConnection;
  private uuid: string;
  private iceServers: RTCIceServer[];
  private announceIntervalMillis: number;
  private announceMsg?: ProviderAnnounce;
  private readonly registry = createRegistry(file_proto_v1_messages);

  public constructor(
    natsUrl: string,
    announceIntervalMinutes: number,
    id: string,
  ) {
    this.iceServers = [{ urls: "stun:stun.l.google.com:19302" }];
    this.natsUrl = natsUrl;
    this.uuid = id;
    this.setupNatsConnection();
    this.announceIntervalMillis = announceIntervalMinutes * 60 * 1000;
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
          this.announceMsg = { id: this.uuid, concurrency };
          this.announce();
          return;
        }
      }
      console.warn("Cannot multiplex transmit with empty identifier", transmit);
    }
  }

  private async setupNatsConnection() {
    // verify nats url
    const con_opts = { servers: this.natsUrl };
    try {
      console.info("connecting to nats server", this.natsUrl);
      this.nc = await nats_core.wsconnect(con_opts);
      console.info(`connected to ${this.nc.getServer()}`);

      // subscribe to sdp channel
      const sub = this.nc.subscribe("sdp");
      (async () => {
        for await (const m of sub) {
          await this.handleSdpMessages(m);
        }
      })();

      // send announce every announceInterval
      setInterval(() => {
        this.announce();
      }, this.announceIntervalMillis);

      this.signal.resolve();
    } catch (err) {
      console.error(`error connecting to ${JSON.stringify(con_opts)}`, err);
    }
  }

  private announce(): void {
    if (this.nc && this.announceMsg) {      
      this.nc.publish("providers", JSON.stringify(this.announceMsg));
    }
  }

  private async handleSdpMessages(m: nats_core.Msg) {
    console.debug(new TextDecoder().decode(m.data));

    const sdpMessage: SdpMessage = JSON.parse(new TextDecoder().decode(m.data));

    if ("Offer" in sdpMessage.msg && sdpMessage.destination === this.uuid) {
      const sdpOffer = sdpMessage.msg.Offer;

      const peerConnection = new RTCPeerConnection({
        iceServers: this.iceServers,
      });
      this.peerConnections.set(sdpMessage.source, peerConnection);

      await peerConnection.setRemoteDescription(sdpOffer);
      const answer = await peerConnection.createAnswer();
      await peerConnection.setLocalDescription(answer);

      const answerMessage: SdpMessage = {
        source: this.uuid,
        destination: sdpMessage.source,
        msg: { Answer: answer },
      };

      peerConnection.onicecandidate = (event) => {
        console.debug("ice candidate event", event.candidate);
        if (event.candidate) {
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
          const iceMessage: SdpMessage = {
            source: this.uuid,
            destination: sdpMessage.source,
            msg: { Candidate: candidateData },
          };
          if (this.nc) {
            this.nc.publish("sdp", JSON.stringify(iceMessage));
          }
        }
      };

      // Set up data channel event handlers
      peerConnection.ondatachannel = (event) => {
        const dataChannel = event.channel;
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
            console.error(
              "Data channel error for:",
              sdpMessage.source,
              error,
            );
          };
          dataChannel.onmessage = async (event) => {
            // Receive Envelope and push together with the source identifier to the message queue
            console.debug!("Received data channel data", event.data);
            const data = event.data;
            if (data instanceof Blob) {
              const array = await data.bytes();
              const envelope = fromBinary(EnvelopeSchema, array);
              const transmit: Transmit = {
                envelope,
                identifier: sdpMessage.source,
              };
              this.messages.push(transmit);
            }
          };
        }
      };

      // Publish the answer back via NATS
      if (this.nc) {
        this.nc.publish("sdp", JSON.stringify(answerMessage));
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
    if (this.nc) {
      this.nc
        .close()
        .catch((err) => console.error("Error closing NATS connection:", err));
    }
    this.messages.close();
    this.signal.reject(err);
    this.controller.abort(err);
  }
}

// timeout for connections?
