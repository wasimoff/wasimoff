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
import { fragmentMessage, PacketDefragmenter } from "./packet";

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
  tasks: number;
  timestamp: number;
  activeConnections: number;
  maxConnections: number;
}

export class WebRTCTransport implements Transport {
  public messages = new PushableAsyncIterable<Transmit>();
  private peerConnections: Map<string, RTCPeerConnection> = new Map();
  private dataChannels: Map<string, RTCDataChannel> = new Map();
  private defragmenters: Map<string, PacketDefragmenter> = new Map();
  private nc: nats_core.NatsConnection;
  private id: string;
  private iceServers: RTCIceServer[];
  private announceMsg?: ProviderAnnounce;
  private readonly registry = createRegistry(file_proto_v1_messages);
  private maxConnections: number;
  private announceInterval: number;
  private announceTimer?: number;

  private constructor(
    natsConnection: nats_core.NatsConnection,
    id: string,
    announceInterval: number,
    maxConnections: number,
  ) {
    this.iceServers = [{ urls: "stun:stun.l.google.com:19302" }];
    this.id = id;
    this.nc = natsConnection;
    this.maxConnections = maxConnections;
    this.announceInterval = announceInterval;
    // subscribe to sdp channel
    const sub = this.nc.subscribe("sdp");
    (async () => {
      for await (const m of sub) {
        await this.handleSdpMessages(m);
      }
    })();

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

    const maxConnectionsParam = url.searchParams.get("maxConnections");
    let maxConnections = 0; // Default to 0 (unlimited) if not provided

    if (maxConnectionsParam) {
      maxConnections = parseInt(maxConnectionsParam, 10);
      if (isNaN(maxConnections) || maxConnections < 0) {
        throw "Invalid maxConnections parameter: must be a non-negative number";
      }
    }

    const con_opts = { servers: url.origin };
    console.info("connecting to nats server", con_opts);
    let nc = await nats_core.wsconnect(con_opts);
    console.info(`connected to ${nc.getServer()}`);
    return new WebRTCTransport(nc, id, announceInterval, maxConnections);
  }

  async send(transmit: Transmit): Promise<void> {
    if (transmit.identifier) {
      const dataChannel = this.dataChannels.get(transmit.identifier);
      if (dataChannel && dataChannel.readyState === "open") {
        const data = toBinary(EnvelopeSchema, transmit.envelope);
        await this.sendFragmentedData(dataChannel, data);
      } else {
        console.error("No open data channel for address", transmit.identifier);
      }
    } else {
      if (transmit.envelope.type === MessageType.Event) {
        if (transmit.envelope.payload === undefined) throw "cannot unpack empty payload";
        let payloadMessage = anyUnpack(transmit.envelope.payload, this.registry);
        if (payloadMessage === undefined) throw "unknown payload type";
        if (isMessage(payloadMessage, Event_ProviderResourcesSchema)) {
          let { concurrency, tasks } = payloadMessage;
          // first announce starts the timer
          this.updateAnnounceMsg({ concurrency, tasks });
          return;
        }
      }
      console.warn("Cannot multiplex transmit with empty identifier", transmit);
    }
  }

  private async sendFragmentedData(
    dataChannel: RTCDataChannel,
    data: Uint8Array<ArrayBuffer>,
  ): Promise<void> {
    const fragments = fragmentMessage(data);

    for (const fragment of fragments) {
      dataChannel.send(fragment.data);
    }
  }

  private processReceivedData(source: string, data: Uint8Array): void {
    let defragmenter = this.defragmenters.get(source);
    if (!defragmenter) {
      defragmenter = new PacketDefragmenter();
      this.defragmenters.set(source, defragmenter);
    }

    defragmenter.processBytes(data);

    // Process all complete messages
    let message;
    while ((message = defragmenter.nextMessage()) !== null) {
      if (message.data.length === 0) {
        console.info(`Received zero-length message from ${source}, disconnecting.`);
        this.removeConnection(source);
        return;
      }
      try {
        const envelope = fromBinary(EnvelopeSchema, message.data);
        const transmit: Transmit = {
          envelope,
          identifier: source,
        };
        this.messages.push(transmit);
      } catch (error) {
        console.error("Failed to parse complete message:", error);
      }
    }
  }

  private announce(): void {
    if (this.announceMsg) {
      this.nc.publish("providers", JSON.stringify(this.announceMsg));
    }
    this.resetAnnounceTimer();
  }

  private updateAnnounceMsg(resources?: { concurrency: number; tasks: number }): void {
    if (!resources && !this.announceMsg) {
      return;
    }

    const now_ns = (performance.now() + performance.timeOrigin) * 1_000_000;
    this.announceMsg = {
      id: this.id,
      concurrency: resources?.concurrency ?? this.announceMsg?.concurrency ?? 0,
      tasks: resources?.tasks ?? this.announceMsg?.tasks ?? 0,
      timestamp: now_ns,
      activeConnections: this.peerConnections.size,
      maxConnections: this.maxConnections,
    };

    // Announce immediately when msg changes
    this.announce();
  }

  private removeConnection(source: string): void {
    // Remove data channel
    this.dataChannels.delete(source);
    // Remove defragmenter
    this.defragmenters.delete(source);

    // Close and remove peer connection
    const peerConnection = this.peerConnections.get(source);
    if (peerConnection) {
      peerConnection.close();
      this.peerConnections.delete(source);
    }

    this.updateAnnounceMsg();

    console.info(
      `Removed connection for: ${source}. Active connections: ${this.peerConnections.size}`,
    );
  }

  private async handleSdpMessages(m: nats_core.Msg) {
    console.debug(new TextDecoder().decode(m.data));

    const sdpMessage: SdpMessage = JSON.parse(new TextDecoder().decode(m.data));

    if (sdpMessage.destination === this.id) {
      if ("Offer" in sdpMessage.msg) {
        // Check if we've reached the maximum connection limit
        if (this.maxConnections > 0 && this.peerConnections.size >= this.maxConnections) {
          console.warn(
            `Rejecting connection from ${sdpMessage.source}: maximum connections (${this.maxConnections}) reached. Current connections: ${this.peerConnections.size}`,
          );
          return; // Don't process the offer
        }

        const sdpOffer = sdpMessage.msg.Offer;

        const peerConnection = new RTCPeerConnection({
          iceServers: this.iceServers,
        });
        this.peerConnections.set(sdpMessage.source, peerConnection);

        this.updateAnnounceMsg();

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

        // Add connection state change handler
        peerConnection.onconnectionstatechange = () => {
          const state = peerConnection.connectionState;
          console.debug(`Peer connection state changed for ${sdpMessage.source}: ${state}`);

          if (state === "failed" || state === "closed" || state === "disconnected") {
            console.info(`Peer connection ${state} for: ${sdpMessage.source}`);
            this.removeConnection(sdpMessage.source);
          }
        };

        // Set up data channel event handlers
        peerConnection.ondatachannel = (event) => {
          const dataChannel = event.channel;
          dataChannel.binaryType = "arraybuffer";
          if (dataChannel.label === "wasimoff") {
            this.dataChannels.set(sdpMessage.source, dataChannel);

            dataChannel.onopen = () => {
              console.info("Data channel opened for:", sdpMessage.source);
            };

            dataChannel.onclose = () => {
              console.info("Data channel closed for:", sdpMessage.source);
              this.removeConnection(sdpMessage.source);
            };

            dataChannel.onerror = (error) => {
              console.error("Data channel error for:", sdpMessage.source, error);
              this.removeConnection(sdpMessage.source);
            };

            dataChannel.onmessage = async (event) => {
              console.debug("Received data channel fragment", event.data);
              const data = event.data;
              let array: Uint8Array;
              if (data instanceof Blob) array = await data.bytes();
              else array = new Uint8Array(data);

              this.processReceivedData(sdpMessage.source, array);
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

  private startAnnounceTimer(): void {
    this.announceTimer = setTimeout(() => {
      this.announce();
    }, this.announceInterval * 1000);
  }

  private resetAnnounceTimer(): void {
    if (this.announceTimer) {
      clearTimeout(this.announceTimer);
    }
    this.startAnnounceTimer();
  }

  public close(reason: string = "closed normally", _: boolean = true) {
    let err = new Error(`WebRTC transport closed: ${reason}`);

    // Clear announce timer
    if (this.announceTimer) {
      clearTimeout(this.announceTimer);
    }

    // Close all peer connections
    for (const [_, peerConnection] of this.peerConnections) {
      peerConnection.close();
    }
    this.dataChannels.clear();
    this.defragmenters.clear();
    this.peerConnections.clear();

    // Close NATS connection
    this.nc.close().catch((err) => console.error("Error closing NATS connection:", err));

    this.messages.close();
    this.signal.reject(err);
    this.controller.abort(err);
  }
}
