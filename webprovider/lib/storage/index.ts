import { LRUCache } from "lru-cache";
import { MemoryFileSystem } from "./fs_memory";
import { EventEmitter } from "@wasimoff/func/eventemitter";

const logprefix = ["%c[ProviderStorage]", "color: purple;"];

// used for cache eviction policies
const minutes = (n: number) => n * 60 * 1000; // in milliseconds
const megabytes = (n: number) => n * 1024 * 1024; // in bytes

/** ProviderStorage is an abstract interface to store and retrieve WebAssembly
 * executables and packed assets. It can for example be backed by a simple
 * in-memory cache or the Origin-Private Filesystem (OPFS) in browsers. */
export class ProviderStorage {
  // underlying filesystem implementation
  filesystem: ProviderStorageFileSystem;

  // base origin for the remote fetching
  public origin?: string;

  public updates = new EventEmitter<{ added?: string[]; removed?: string[] }>();

  // cache compiled webassembly modules
  private wasmCache: LRUCache<string, { size: number; module: WebAssembly.Module }>;

  // cache zip archives for rootfs
  private zipCache: LRUCache<string, ArrayBuffer>;

  constructor(filesystem?: ProviderStorageFileSystem, fetchOrigin?: string) {
    // if no filesystem was given, instantiate a MemoryFilesystem
    if (filesystem === undefined) this.filesystem = new MemoryFileSystem();
    else this.filesystem = filesystem;
    console.debug(...logprefix, `opened storage with`, this.filesystem.constructor.name);

    this.wasmCache = new LRUCache<string, { size: number; module: WebAssembly.Module }>({
      // eviction policy
      ttl: minutes(30),
      max: 64, // items
      //? larger items can be fetched but won't be cached
      maxSize: megabytes(128),
      sizeCalculation: (item, _) => item.size,

      fetchMethod: async (filename) => {
        let file = await this.getFile(filename);
        if (file === undefined) return undefined;
        let buf = await file.arrayBuffer();
        return {
          size: buf.byteLength,
          module: await WebAssembly.compile(buf),
        };
      },
    });

    this.zipCache = new LRUCache<string, ArrayBuffer>({
      // eviction policy
      ttl: minutes(30),
      max: 64, // items
      maxSize: megabytes(128),
      sizeCalculation: (item, _) => item.byteLength,

      fetchMethod: async (filename) => {
        let file = await this.getFile(filename);
        if (file === undefined) return undefined;
        return await file.arrayBuffer();
      },
    });

    this.origin = fetchOrigin;
  }

  // fetch a file from the backend
  private async fetchFile(filename: string): Promise<File | undefined> {
    // request the file from broker
    console.warn(...logprefix, `file ${filename} not found locally, fetch from broker`);
    let response = await fetch(`${this.origin}/api/storage/${filename}`);
    if (!response.ok) return undefined;

    // store fetched file to filesystem
    let buf = (await response.bytes()) as Uint8Array<ArrayBuffer>;
    let media = response.headers.get("content-type") || "";
    let name = response.headers.get("x-wasimoff-ref") || (await getRef(buf));
    let file = new File([buf], name, { type: media });
    await this.filesystem.put(name, file);

    // emit event for broker
    this.updates.emit({ added: [name] });
    return file;
  }

  // TODO: emitting events for removed files requires shimming the FileSystem functions

  // either return a file from filesystem or attempt to fetch it remotely
  private async getFile(filename: string): Promise<File | undefined> {
    let file = await this.filesystem.get(filename);
    if (!file && this.origin) file = await this.fetchFile(filename);
    return file;
  }

  /** Get a WebAssembly module compiled from a stored executable. */
  async getWasmModule(filename: string): Promise<WebAssembly.Module | undefined> {
    return (await this.wasmCache.fetch(filename))?.module;
  }

  /** Get a ZIP archive for rootfs usage. */
  async getZipArchive(filename: string): Promise<ArrayBuffer | undefined> {
    return this.zipCache.fetch(filename);
  }
}

/** ProviderStorageFileSystem is an underlying structure, which actually holds the
 * files. It minimally needs to support list, has, get, put and delete operations,
 * like a Map<string, File>. */
export interface ProviderStorageFileSystem {
  /** Return the currently opened path. */
  readonly path: string;

  /** List all files in this Filesystem. */
  list(): Promise<string[]>;

  /** Check if a file exists in Filesystem. */
  has(filename: string): Promise<boolean>;

  /** Get a specific file from Filesystem. */
  get(filename: string): Promise<File | undefined>;

  /** Save a new file to the Filesystem. */
  put(filename: string, file: File): Promise<File>;

  /** Remove a file from the Filesystem. */
  delete(filename: string): Promise<boolean>;
}

/** Return the SHA-256 digest of a file. This can be used to check for an exact match
 * without actually transferring the file's contents. */
export async function digest(buf: Uint8Array<ArrayBuffer>): Promise<Uint8Array> {
  if (crypto.subtle) {
    return new Uint8Array(await crypto.subtle.digest("SHA-256", buf));
  } else return new Uint8Array(32); // will always re-transfer
}

/** Check if a filename is a SHA256 content address aka. ref. */
export function isRef(filename: string): boolean {
  return filename.match(/^sha256:[0-9a-f]{64}$/i) !== null;
}

export async function getRef(buf: Uint8Array<ArrayBuffer>): Promise<string> {
  if (!crypto.subtle) throw "cannot compute digest in an insecure context";
  let hash = await digest(buf);
  let hex = [...hash].map((d) => d.toString(16).padStart(2, "0")).join("");
  return `sha256:${hex}`;
}

/** Return a bytelength in human-readable unit. */
export function filesize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(2)} KiB`;
  if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(2)} MiB`;
  return `${(bytes / 1024 ** 3).toFixed(2)} GiB`;
}
