/** Maximum fragment size in bytes (configurable constant) */
export const MAX_FRAGMENT_SIZE: number = 1024;

/** Represents a complete message ready to be processed */
export interface CompleteMessage {
  data: Uint8Array;
}

/** Represents a fragment of a message */
export interface Fragment {
  data: Uint8Array;
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
  const lengthArray = new Uint8Array(lengthBytes);
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
  | { type: "WaitingForLength"; buffer: Uint8Array }
  | { type: "WaitingForData"; expectedLength: number; buffer: Uint8Array };

/** Packet defragmenter that buffers incoming bytes and reconstructs complete messages */
export class PacketDefragmenter {
  private state: DefragmentationState;
  private completedMessages: CompleteMessage[];

  /** Create a new packet defragmenter */
  constructor() {
    this.state = { type: "WaitingForLength", buffer: new Uint8Array(0) };
    this.completedMessages = [];
  }

  /** Process incoming bytes and potentially produce complete messages */
  processBytes(bytes: Uint8Array): void {
    let offset = 0;

    while (offset < bytes.length) {
      if (this.state.type === "WaitingForLength") {
        // We need 4 bytes for the length
        const bytesNeeded = 4 - this.state.buffer.length;
        const bytesAvailable = bytes.length - offset;
        const bytesToCopy = Math.min(bytesNeeded, bytesAvailable);

        // Extend buffer with new bytes
        const newBuffer = new Uint8Array(this.state.buffer.length + bytesToCopy);
        newBuffer.set(this.state.buffer, 0);
        newBuffer.set(bytes.slice(offset, offset + bytesToCopy), this.state.buffer.length);
        this.state.buffer = newBuffer;
        offset += bytesToCopy;

        // If we have 4 bytes, we can read the length
        if (this.state.buffer.length === 4) {
          const dataView = new DataView(this.state.buffer.buffer, this.state.buffer.byteOffset, 4);
          const expectedLength = dataView.getUint32(0, false); // big-endian

          // Validate message length (prevent excessive memory allocation)
          if (expectedLength > 100_000_000) {
            // 100MB limit
            throw new Error(`Message too large: ${expectedLength} bytes`);
          }

          // Special case for zero-length messages
          if (expectedLength === 0) {
            this.completedMessages.push({ data: new Uint8Array(0) });
            // Reset state for next message
            this.state = { type: "WaitingForLength", buffer: new Uint8Array(0) };
          } else {
            this.state = {
              type: "WaitingForData",
              expectedLength,
              buffer: new Uint8Array(0),
            };
          }
        }
      } else if (this.state.type === "WaitingForData") {
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
          this.completedMessages.push({ data: this.state.buffer });

          // Reset state for next message
          this.state = { type: "WaitingForLength", buffer: new Uint8Array(0) };
        }
      }
    }
  }

  /** Get the next complete message if available */
  nextMessage(): CompleteMessage | null {
    return this.completedMessages.shift() || null;
  }

  /** Check if there are any complete messages ready */
  hasMessages(): boolean {
    return this.completedMessages.length > 0;
  }

  /** Get the number of complete messages ready */
  messageCount(): number {
    return this.completedMessages.length;
  }

  /** Reset the defragmenter state (useful for connection reset) */
  reset(): void {
    this.state = { type: "WaitingForLength", buffer: new Uint8Array(0) };
    this.completedMessages = [];
  }
}
