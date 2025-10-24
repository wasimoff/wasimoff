/** Maximum fragment size in bytes (configurable constant) */
export const MAX_FRAGMENT_SIZE: number = 64000;

/** Represents a complete message ready to be processed */
export interface CompleteMessage {
  data: Uint8Array<ArrayBuffer>;
}

/** Represents a fragment of a message */
export interface Fragment {
  data: Uint8Array<ArrayBuffer>;
}

/** Fragment a message into multiple packets with the default fragment size */
export function fragmentMessage(data: Uint8Array): Fragment[] {
  return fragmentMessageWithSize(data, MAX_FRAGMENT_SIZE);
}

/** Fragment a message into multiple packets with a custom fragment size */
export function fragmentMessageWithSize(data: Uint8Array, fragmentSize: number): Fragment[] {
  if (fragmentSize < 5) {
    // Need at least 4 bytes for length + 1 byte for data
    throw new Error("Fragment size must be at least 5 bytes");
  }

  const fragments: Fragment[] = [];

  // First fragment contains the length prefix
  const lengthBytes = new ArrayBuffer(4);
  new DataView(lengthBytes).setUint32(0, data.length, false); // big-endian
  const lengthArray = new Uint8Array<ArrayBuffer>(lengthBytes);
  const firstFragmentDataSize = fragmentSize - 4; // Reserve 4 bytes for length

  if (data.length <= firstFragmentDataSize) {
    // Message fits in a single fragment
    const fragmentData = new Uint8Array(4 + data.length);
    fragmentData.set(lengthArray, 0);
    fragmentData.set(data, 4);
    fragments.push({ data: fragmentData });
  } else {
    // Message needs multiple fragments
    let remainingData = data;

    // First fragment with length prefix
    const firstFragment = new Uint8Array(fragmentSize);
    firstFragment.set(lengthArray, 0);
    firstFragment.set(remainingData.slice(0, firstFragmentDataSize), 4);
    fragments.push({ data: firstFragment });

    remainingData = remainingData.slice(firstFragmentDataSize);

    // Subsequent fragments
    while (remainingData.length > 0) {
      const chunkSize = Math.min(remainingData.length, fragmentSize);
      fragments.push({
        data: remainingData.slice(0, chunkSize),
      });
      remainingData = remainingData.slice(chunkSize);
    }
  }

  return fragments;
}

/** State for packet defragmentation */
export type DefragmentationState =
  | { type: "WaitingForNewMessage" }
  | { type: "WaitingForData"; expectedLength: number; buffer: Uint8Array };

/** Packet defragmenter that buffers incoming bytes and reconstructs complete messages */
export class PacketDefragmenter {
  private state: DefragmentationState;
  private completedMessages: CompleteMessage[];

  /** Create a new packet defragmenter */
  constructor() {
    this.state = { type: "WaitingForNewMessage" };
    this.completedMessages = [];
  }

  /** Process incoming bytes and potentially produce complete messages */
  processBytes(bytes: Uint8Array): void {
    let offset = 0;

    while (offset < bytes.length) {
      if (this.state.type === "WaitingForNewMessage") {
        // Need at least 4 bytes to read the length
        if (bytes.length - offset < 4) {
          // Not enough bytes to read length, wait for more data
          break;
        }

        // Read the length from the first 4 bytes
        const dataView = new DataView(bytes.buffer, bytes.byteOffset + offset, 4);
        const expectedLength = dataView.getUint32(0, false); // big-endian
        offset += 4;

        // Validate message length (prevent excessive memory allocation)
        if (expectedLength > 100_000_000) {
          // 100MB limit
          throw new Error(`Message too large: ${expectedLength} bytes`);
        }

        // Special case for zero-length messages
        if (expectedLength === 0) {
          this.completedMessages.push({ data: new Uint8Array(0) });
          // Stay in WaitingForNewMessage state for next message
          continue;
        }

        // Transition to waiting for data
        this.state = {
          type: "WaitingForData",
          expectedLength,
          buffer: new Uint8Array(0),
        };
      }

      if (this.state.type === "WaitingForData") {
        const bytesNeeded = this.state.expectedLength - this.state.buffer.length;
        const bytesAvailable = bytes.length - offset;
        const bytesToCopy = Math.min(bytesNeeded, bytesAvailable);

        // Extend buffer with new bytes
        const newBuffer = new Uint8Array(this.state.buffer.length + bytesToCopy);
        newBuffer.set(this.state.buffer, 0);
        newBuffer.set(bytes.slice(offset, offset + bytesToCopy), this.state.buffer.length);
        this.state.buffer = newBuffer;
        offset += bytesToCopy;

        // If we have the complete message, add it to completed messages
        if (this.state.buffer.length === this.state.expectedLength) {
          this.completedMessages.push({ data: new Uint8Array(this.state.buffer) });

          // Reset state for next message
          this.state = { type: "WaitingForNewMessage" };
        }
      }
    }
  }

  /** Get the next complete message if available */
  nextMessage(): CompleteMessage | null {
    return this.completedMessages.shift() || null;
  }
}
