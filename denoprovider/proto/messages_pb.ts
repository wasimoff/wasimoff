// best practices: https://protobuf.dev/programming-guides/dos-donts/

// @generated by protoc-gen-es v2.0.0 with parameter "target=ts,json_types=true"
// @generated from file messages.proto (edition 2023)
/* eslint-disable */

import type { GenEnum, GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv1";
import { enumDesc, fileDesc, messageDesc, serviceDesc } from "@bufbuild/protobuf/codegenv1";
import type { Any, AnyJson } from "@bufbuild/protobuf/wkt";
import { file_google_protobuf_any } from "@bufbuild/protobuf/wkt";
import type { Message } from "@bufbuild/protobuf";

/**
 * Describes the file messages.proto.
 */
export const file_messages: GenFile = /*@__PURE__*/
  fileDesc("Cg5tZXNzYWdlcy5wcm90byK5AQoIRW52ZWxvcGUSEAoIc2VxdWVuY2UYASABKAQSIwoEdHlwZRgCIAEoDjIVLkVudmVsb3BlLk1lc3NhZ2VUeXBlEg0KBWVycm9yGAMgASgJEiUKB3BheWxvYWQYBCABKAsyFC5nb29nbGUucHJvdG9idWYuQW55IkAKC01lc3NhZ2VUeXBlEgsKB1VOS05PV04QABILCgdSZXF1ZXN0EAESDAoIUmVzcG9uc2UQAhIJCgVFdmVudBADIksKBFBpbmcSIgoJZGlyZWN0aW9uGAEgASgOMg8uUGluZy5EaXJlY3Rpb24iHwoJRGlyZWN0aW9uEggKBFBpbmcQABIICgRQb25nEAEiOQoMVGFza01ldGFkYXRhEgoKAmlkGAEgASgJEg4KBmNsaWVudBgCIAEoCRINCgVpbmRleBgDIAEoBCI6CgpFeGVjdXRhYmxlEhMKCXJlZmVyZW5jZRgBIAEoCUgAEg0KA3JhdxgCIAEoDEgAQggKBmJpbmFyeSKnAQoPRXhlY3V0ZVdhc2lBcmdzEhsKBHRhc2sYASABKAsyDS5UYXNrTWV0YWRhdGESGwoGYmluYXJ5GAIgASgLMgsuRXhlY3V0YWJsZRIMCgRhcmdzGAMgAygJEgwKBGVudnMYBCADKAkSDQoFc3RkaW4YBSABKAwSDgoGbG9hZGZzGAYgAygJEhAKCGRhdGFmaWxlGAcgASgJEg0KBXRyYWNlGAggASgIInUKEUV4ZWN1dGVXYXNpUmVzdWx0Eg4KBnN0YXR1cxgBIAEoBRIOCgZzdGRvdXQYAiABKAwSDgoGc3RkZXJyGAMgASgMEhAKCGRhdGFmaWxlGAQgASgMEh4KBXRyYWNlGAUgASgLMg8uRXhlY3V0aW9uVHJhY2UiEAoORXhlY3V0aW9uVHJhY2UiXgoIRmlsZVN0YXQSEAoIZmlsZW5hbWUYASABKAkSEwoLY29udGVudHR5cGUYAiABKAkSDgoGbGVuZ3RoGAMgASgEEg0KBWVwb2NoGAUgASgDEgwKBGhhc2gYBCABKAwiEQoPRmlsZUxpc3RpbmdBcmdzIi0KEUZpbGVMaXN0aW5nUmVzdWx0EhgKBWZpbGVzGAEgAygLMgkuRmlsZVN0YXQiKAoNRmlsZVByb2JlQXJncxIXCgRzdGF0GAEgASgLMgkuRmlsZVN0YXQiHQoPRmlsZVByb2JlUmVzdWx0EgoKAm9rGAEgASgIIjcKDkZpbGVVcGxvYWRBcmdzEhcKBHN0YXQYASABKAsyCS5GaWxlU3RhdBIMCgRmaWxlGAIgASgMIh4KEEZpbGVVcGxvYWRSZXN1bHQSCgoCb2sYASABKAgiHwoMR2VuZXJpY0V2ZW50Eg8KB21lc3NhZ2UYASABKAkiYwoMUHJvdmlkZXJJbmZvEgwKBG5hbWUYASABKAkSEAoIcGxhdGZvcm0YAiABKAkSEQoJdXNlcmFnZW50GAMgASgJEiAKBHBvb2wYBCABKAsyEi5Qcm92aWRlclJlc291cmNlcyI3ChFQcm92aWRlclJlc291cmNlcxITCgtjb25jdXJyZW5jeRgBIAEoDRINCgV0YXNrcxgCIAEoDSpcCgtTdWJwcm90b2NvbBILCgdVTktOT1dOEAASIQodd2FzaW1vZmZfcHJvdmlkZXJfdjFfcHJvdG9idWYQARIdChl3YXNpbW9mZl9wcm92aWRlcl92MV9qc29uEAIy1QEKCFByb3ZpZGVyEjMKC0V4ZWN1dGVXYXNpEhAuRXhlY3V0ZVdhc2lBcmdzGhIuRXhlY3V0ZVdhc2lSZXN1bHQSLQoJRmlsZVByb2JlEg4uRmlsZVByb2JlQXJncxoQLkZpbGVQcm9iZVJlc3VsdBIzCgtGaWxlTGlzdGluZxIQLkZpbGVMaXN0aW5nQXJncxoSLkZpbGVMaXN0aW5nUmVzdWx0EjAKCkZpbGVVcGxvYWQSDy5GaWxlVXBsb2FkQXJncxoRLkZpbGVVcGxvYWRSZXN1bHRiCGVkaXRpb25zcOgH", [file_google_protobuf_any]);

/**
 * Envelope is a generic message wrapper with a sequence counter and message type.
 * The payload contains a { Request, Response, Event }.
 *
 * @generated from message Envelope
 */
export type Envelope = Message<"Envelope"> & {
  /**
   * The sequence number is incremented for each message but Request and Event
   * count independently. Responses must always reuse the Request's sequence
   * number so they can be routed to the caller correctly.
   *
   * @generated from field: uint64 sequence = 1;
   */
  sequence: bigint;

  /**
   * The message type indicates the payload contents: { Request, Response, Event }.
   *
   * @generated from field: Envelope.MessageType type = 2;
   */
  type: Envelope_MessageType;

  /**
   * The presence of an error indicates that something went wrong with the call
   * in general (like a server "oops"). Otherwise, the called function should
   * encode specific errors within the payload.
   *
   * @generated from field: string error = 3;
   */
  error: string;

  /**
   * The payload itself. Needs to be (un)packed with `anypb`.
   *
   * @generated from field: google.protobuf.Any payload = 4;
   */
  payload?: Any;
};

/**
 * JSON type for the message Envelope.
 */
export type EnvelopeJson = {
  /**
   * @generated from field: uint64 sequence = 1;
   */
  sequence?: string;

  /**
   * @generated from field: Envelope.MessageType type = 2;
   */
  type?: Envelope_MessageTypeJson;

  /**
   * @generated from field: string error = 3;
   */
  error?: string;

  /**
   * @generated from field: google.protobuf.Any payload = 4;
   */
  payload?: AnyJson;
};

/**
 * Describes the message Envelope.
 * Use `create(EnvelopeSchema)` to create a new message.
 */
export const EnvelopeSchema: GenMessage<Envelope, EnvelopeJson> = /*@__PURE__*/
  messageDesc(file_messages, 0);

/**
 * @generated from enum Envelope.MessageType
 */
export enum Envelope_MessageType {
  /**
   * @generated from enum value: UNKNOWN = 0;
   */
  UNKNOWN = 0,

  /**
   * @generated from enum value: Request = 1;
   */
  Request = 1,

  /**
   * @generated from enum value: Response = 2;
   */
  Response = 2,

  /**
   * @generated from enum value: Event = 3;
   */
  Event = 3,
}

/**
 * JSON type for the enum Envelope.MessageType.
 */
export type Envelope_MessageTypeJson = "UNKNOWN" | "Request" | "Response" | "Event";

/**
 * Describes the enum Envelope.MessageType.
 */
export const Envelope_MessageTypeSchema: GenEnum<Envelope_MessageType, Envelope_MessageTypeJson> = /*@__PURE__*/
  enumDesc(file_messages, 0, 0);

/**
 * Ping stub, if the transport does not provide them. WebSocket does have its
 * own mechanism. On WebTransport, you should use a separate stream to avoid re-
 * introducing head-of-line blocking with the other RPC requests.
 *
 * @generated from message Ping
 */
export type Ping = Message<"Ping"> & {
  /**
   * @generated from field: Ping.Direction direction = 1;
   */
  direction: Ping_Direction;
};

/**
 * JSON type for the message Ping.
 */
export type PingJson = {
  /**
   * @generated from field: Ping.Direction direction = 1;
   */
  direction?: Ping_DirectionJson;
};

/**
 * Describes the message Ping.
 * Use `create(PingSchema)` to create a new message.
 */
export const PingSchema: GenMessage<Ping, PingJson> = /*@__PURE__*/
  messageDesc(file_messages, 1);

/**
 * @generated from enum Ping.Direction
 */
export enum Ping_Direction {
  /**
   * @generated from enum value: Ping = 0;
   */
  Ping = 0,

  /**
   * @generated from enum value: Pong = 1;
   */
  Pong = 1,
}

/**
 * JSON type for the enum Ping.Direction.
 */
export type Ping_DirectionJson = "Ping" | "Pong";

/**
 * Describes the enum Ping.Direction.
 */
export const Ping_DirectionSchema: GenEnum<Ping_Direction, Ping_DirectionJson> = /*@__PURE__*/
  enumDesc(file_messages, 1, 0);

/**
 * TaskMetadata contains some information about the originating task for logging
 *
 * @generated from message TaskMetadata
 */
export type TaskMetadata = Message<"TaskMetadata"> & {
  /**
   * overall job ID
   *
   * @generated from field: string id = 1;
   */
  id: string;

  /**
   * info about the requesting client
   *
   * @generated from field: string client = 2;
   */
  client: string;

  /**
   * index within a job with multiple tasks
   *
   * @generated from field: uint64 index = 3;
   */
  index: bigint;
};

/**
 * JSON type for the message TaskMetadata.
 */
export type TaskMetadataJson = {
  /**
   * @generated from field: string id = 1;
   */
  id?: string;

  /**
   * @generated from field: string client = 2;
   */
  client?: string;

  /**
   * @generated from field: uint64 index = 3;
   */
  index?: string;
};

/**
 * Describes the message TaskMetadata.
 * Use `create(TaskMetadataSchema)` to create a new message.
 */
export const TaskMetadataSchema: GenMessage<TaskMetadata, TaskMetadataJson> = /*@__PURE__*/
  messageDesc(file_messages, 2);

/**
 * Executable can be either a string reference or the raw binary itself
 *
 * @generated from message Executable
 */
export type Executable = Message<"Executable"> & {
  /**
   * @generated from oneof Executable.binary
   */
  binary: {
    /**
     * @generated from field: string reference = 1;
     */
    value: string;
    case: "reference";
  } | {
    /**
     * @generated from field: bytes raw = 2;
     */
    value: Uint8Array;
    case: "raw";
  } | { case: undefined; value?: undefined };
};

/**
 * JSON type for the message Executable.
 */
export type ExecutableJson = {
  /**
   * @generated from field: string reference = 1;
   */
  reference?: string;

  /**
   * @generated from field: bytes raw = 2;
   */
  raw?: string;
};

/**
 * Describes the message Executable.
 * Use `create(ExecutableSchema)` to create a new message.
 */
export const ExecutableSchema: GenMessage<Executable, ExecutableJson> = /*@__PURE__*/
  messageDesc(file_messages, 3);

/**
 * ExecuteWasi runs a webassembly/wasi binary on the Provider
 *
 * @generated from message ExecuteWasiArgs
 */
export type ExecuteWasiArgs = Message<"ExecuteWasiArgs"> & {
  /**
   * @generated from field: TaskMetadata task = 1;
   */
  task?: TaskMetadata;

  /**
   * @generated from field: Executable binary = 2;
   */
  binary?: Executable;

  /**
   * @generated from field: repeated string args = 3;
   */
  args: string[];

  /**
   * @generated from field: repeated string envs = 4;
   */
  envs: string[];

  /**
   * @generated from field: bytes stdin = 5;
   */
  stdin: Uint8Array;

  /**
   * @generated from field: repeated string loadfs = 6;
   */
  loadfs: string[];

  /**
   * @generated from field: string datafile = 7;
   */
  datafile: string;

  /**
   * @generated from field: bool trace = 8;
   */
  trace: boolean;
};

/**
 * JSON type for the message ExecuteWasiArgs.
 */
export type ExecuteWasiArgsJson = {
  /**
   * @generated from field: TaskMetadata task = 1;
   */
  task?: TaskMetadataJson;

  /**
   * @generated from field: Executable binary = 2;
   */
  binary?: ExecutableJson;

  /**
   * @generated from field: repeated string args = 3;
   */
  args?: string[];

  /**
   * @generated from field: repeated string envs = 4;
   */
  envs?: string[];

  /**
   * @generated from field: bytes stdin = 5;
   */
  stdin?: string;

  /**
   * @generated from field: repeated string loadfs = 6;
   */
  loadfs?: string[];

  /**
   * @generated from field: string datafile = 7;
   */
  datafile?: string;

  /**
   * @generated from field: bool trace = 8;
   */
  trace?: boolean;
};

/**
 * Describes the message ExecuteWasiArgs.
 * Use `create(ExecuteWasiArgsSchema)` to create a new message.
 */
export const ExecuteWasiArgsSchema: GenMessage<ExecuteWasiArgs, ExecuteWasiArgsJson> = /*@__PURE__*/
  messageDesc(file_messages, 4);

/**
 * @generated from message ExecuteWasiResult
 */
export type ExecuteWasiResult = Message<"ExecuteWasiResult"> & {
  /**
   * @generated from field: int32 status = 1;
   */
  status: number;

  /**
   * @generated from field: bytes stdout = 2;
   */
  stdout: Uint8Array;

  /**
   * @generated from field: bytes stderr = 3;
   */
  stderr: Uint8Array;

  /**
   * @generated from field: bytes datafile = 4;
   */
  datafile: Uint8Array;

  /**
   * @generated from field: ExecutionTrace trace = 5;
   */
  trace?: ExecutionTrace;
};

/**
 * JSON type for the message ExecuteWasiResult.
 */
export type ExecuteWasiResultJson = {
  /**
   * @generated from field: int32 status = 1;
   */
  status?: number;

  /**
   * @generated from field: bytes stdout = 2;
   */
  stdout?: string;

  /**
   * @generated from field: bytes stderr = 3;
   */
  stderr?: string;

  /**
   * @generated from field: bytes datafile = 4;
   */
  datafile?: string;

  /**
   * @generated from field: ExecutionTrace trace = 5;
   */
  trace?: ExecutionTraceJson;
};

/**
 * Describes the message ExecuteWasiResult.
 * Use `create(ExecuteWasiResultSchema)` to create a new message.
 */
export const ExecuteWasiResultSchema: GenMessage<ExecuteWasiResult, ExecuteWasiResultJson> = /*@__PURE__*/
  messageDesc(file_messages, 5);

/**
 * TODO
 *
 * @generated from message ExecutionTrace
 */
export type ExecutionTrace = Message<"ExecutionTrace"> & {
};

/**
 * JSON type for the message ExecutionTrace.
 */
export type ExecutionTraceJson = {
};

/**
 * Describes the message ExecutionTrace.
 * Use `create(ExecutionTraceSchema)` to create a new message.
 */
export const ExecutionTraceSchema: GenMessage<ExecutionTrace, ExecutionTraceJson> = /*@__PURE__*/
  messageDesc(file_messages, 6);

/**
 * FileStat contains metadata about a file for identification in other messages
 *
 * @generated from message FileStat
 */
export type FileStat = Message<"FileStat"> & {
  /**
   * @generated from field: string filename = 1;
   */
  filename: string;

  /**
   * @generated from field: string contenttype = 2;
   */
  contenttype: string;

  /**
   * @generated from field: uint64 length = 3;
   */
  length: bigint;

  /**
   * @generated from field: int64 epoch = 5;
   */
  epoch: bigint;

  /**
   * @generated from field: bytes hash = 4;
   */
  hash: Uint8Array;
};

/**
 * JSON type for the message FileStat.
 */
export type FileStatJson = {
  /**
   * @generated from field: string filename = 1;
   */
  filename?: string;

  /**
   * @generated from field: string contenttype = 2;
   */
  contenttype?: string;

  /**
   * @generated from field: uint64 length = 3;
   */
  length?: string;

  /**
   * @generated from field: int64 epoch = 5;
   */
  epoch?: string;

  /**
   * @generated from field: bytes hash = 4;
   */
  hash?: string;
};

/**
 * Describes the message FileStat.
 * Use `create(FileStatSchema)` to create a new message.
 */
export const FileStatSchema: GenMessage<FileStat, FileStatJson> = /*@__PURE__*/
  messageDesc(file_messages, 7);

/**
 * FileListing asks for a listing of all available files on Provider
 *
 * empty
 *
 * @generated from message FileListingArgs
 */
export type FileListingArgs = Message<"FileListingArgs"> & {
};

/**
 * JSON type for the message FileListingArgs.
 */
export type FileListingArgsJson = {
};

/**
 * Describes the message FileListingArgs.
 * Use `create(FileListingArgsSchema)` to create a new message.
 */
export const FileListingArgsSchema: GenMessage<FileListingArgs, FileListingArgsJson> = /*@__PURE__*/
  messageDesc(file_messages, 8);

/**
 * @generated from message FileListingResult
 */
export type FileListingResult = Message<"FileListingResult"> & {
  /**
   * @generated from field: repeated FileStat files = 1;
   */
  files: FileStat[];
};

/**
 * JSON type for the message FileListingResult.
 */
export type FileListingResultJson = {
  /**
   * @generated from field: repeated FileStat files = 1;
   */
  files?: FileStatJson[];
};

/**
 * Describes the message FileListingResult.
 * Use `create(FileListingResultSchema)` to create a new message.
 */
export const FileListingResultSchema: GenMessage<FileListingResult, FileListingResultJson> = /*@__PURE__*/
  messageDesc(file_messages, 9);

/**
 * FileProbe checks if a certain file exists on provider
 *
 * @generated from message FileProbeArgs
 */
export type FileProbeArgs = Message<"FileProbeArgs"> & {
  /**
   * @generated from field: FileStat stat = 1;
   */
  stat?: FileStat;
};

/**
 * JSON type for the message FileProbeArgs.
 */
export type FileProbeArgsJson = {
  /**
   * @generated from field: FileStat stat = 1;
   */
  stat?: FileStatJson;
};

/**
 * Describes the message FileProbeArgs.
 * Use `create(FileProbeArgsSchema)` to create a new message.
 */
export const FileProbeArgsSchema: GenMessage<FileProbeArgs, FileProbeArgsJson> = /*@__PURE__*/
  messageDesc(file_messages, 10);

/**
 * @generated from message FileProbeResult
 */
export type FileProbeResult = Message<"FileProbeResult"> & {
  /**
   * @generated from field: bool ok = 1;
   */
  ok: boolean;
};

/**
 * JSON type for the message FileProbeResult.
 */
export type FileProbeResultJson = {
  /**
   * @generated from field: bool ok = 1;
   */
  ok?: boolean;
};

/**
 * Describes the message FileProbeResult.
 * Use `create(FileProbeResultSchema)` to create a new message.
 */
export const FileProbeResultSchema: GenMessage<FileProbeResult, FileProbeResultJson> = /*@__PURE__*/
  messageDesc(file_messages, 11);

/**
 * FileUpload pushes a file to the Provider.
 *
 * @generated from message FileUploadArgs
 */
export type FileUploadArgs = Message<"FileUploadArgs"> & {
  /**
   * @generated from field: FileStat stat = 1;
   */
  stat?: FileStat;

  /**
   * @generated from field: bytes file = 2;
   */
  file: Uint8Array;
};

/**
 * JSON type for the message FileUploadArgs.
 */
export type FileUploadArgsJson = {
  /**
   * @generated from field: FileStat stat = 1;
   */
  stat?: FileStatJson;

  /**
   * @generated from field: bytes file = 2;
   */
  file?: string;
};

/**
 * Describes the message FileUploadArgs.
 * Use `create(FileUploadArgsSchema)` to create a new message.
 */
export const FileUploadArgsSchema: GenMessage<FileUploadArgs, FileUploadArgsJson> = /*@__PURE__*/
  messageDesc(file_messages, 12);

/**
 * @generated from message FileUploadResult
 */
export type FileUploadResult = Message<"FileUploadResult"> & {
  /**
   * @generated from field: bool ok = 1;
   */
  ok: boolean;
};

/**
 * JSON type for the message FileUploadResult.
 */
export type FileUploadResultJson = {
  /**
   * @generated from field: bool ok = 1;
   */
  ok?: boolean;
};

/**
 * Describes the message FileUploadResult.
 * Use `create(FileUploadResultSchema)` to create a new message.
 */
export const FileUploadResultSchema: GenMessage<FileUploadResult, FileUploadResultJson> = /*@__PURE__*/
  messageDesc(file_messages, 13);

/**
 * Generic is just a generic piece of text for debugging
 *
 * @generated from message GenericEvent
 */
export type GenericEvent = Message<"GenericEvent"> & {
  /**
   * @generated from field: string message = 1;
   */
  message: string;
};

/**
 * JSON type for the message GenericEvent.
 */
export type GenericEventJson = {
  /**
   * @generated from field: string message = 1;
   */
  message?: string;
};

/**
 * Describes the message GenericEvent.
 * Use `create(GenericEventSchema)` to create a new message.
 */
export const GenericEventSchema: GenMessage<GenericEvent, GenericEventJson> = /*@__PURE__*/
  messageDesc(file_messages, 14);

/**
 * ProviderInfo is sent once at the beginning to identify the Provider
 *
 * @generated from message ProviderInfo
 */
export type ProviderInfo = Message<"ProviderInfo"> & {
  /**
   * a logging-friendly name of the provider // TODO: make a persistent ID?
   *
   * @generated from field: string name = 1;
   */
  name: string;

  /**
   * like the navigator.platform in browser
   *
   * @generated from field: string platform = 2;
   */
  platform: string;

  /**
   * like the navigator.useragent in browser
   *
   * @generated from field: string useragent = 3;
   */
  useragent: string;

  /**
   * @generated from field: ProviderResources pool = 4;
   */
  pool?: ProviderResources;
};

/**
 * JSON type for the message ProviderInfo.
 */
export type ProviderInfoJson = {
  /**
   * @generated from field: string name = 1;
   */
  name?: string;

  /**
   * @generated from field: string platform = 2;
   */
  platform?: string;

  /**
   * @generated from field: string useragent = 3;
   */
  useragent?: string;

  /**
   * @generated from field: ProviderResources pool = 4;
   */
  pool?: ProviderResourcesJson;
};

/**
 * Describes the message ProviderInfo.
 * Use `create(ProviderInfoSchema)` to create a new message.
 */
export const ProviderInfoSchema: GenMessage<ProviderInfo, ProviderInfoJson> = /*@__PURE__*/
  messageDesc(file_messages, 15);

/**
 * ProviderResources is information about the available resources in Worker pool
 *
 * @generated from message ProviderResources
 */
export type ProviderResources = Message<"ProviderResources"> & {
  /**
   * maximum possible concurrency
   *
   * @generated from field: uint32 concurrency = 1;
   */
  concurrency: number;

  /**
   * currently active tasks
   *
   * @generated from field: uint32 tasks = 2;
   */
  tasks: number;
};

/**
 * JSON type for the message ProviderResources.
 */
export type ProviderResourcesJson = {
  /**
   * @generated from field: uint32 concurrency = 1;
   */
  concurrency?: number;

  /**
   * @generated from field: uint32 tasks = 2;
   */
  tasks?: number;
};

/**
 * Describes the message ProviderResources.
 * Use `create(ProviderResourcesSchema)` to create a new message.
 */
export const ProviderResourcesSchema: GenMessage<ProviderResources, ProviderResourcesJson> = /*@__PURE__*/
  messageDesc(file_messages, 16);

/**
 * Subprotocol is used to identify the concrete encoding on the wire.
 *
 * @generated from enum Subprotocol
 */
export enum Subprotocol {
  /**
   * @generated from enum value: UNKNOWN = 0;
   */
  UNKNOWN = 0,

  /**
   * binary messages with Protobuf encoding
   *
   * @generated from enum value: wasimoff_provider_v1_protobuf = 1;
   */
  wasimoff_provider_v1_protobuf = 1,

  /**
   * text messages with JSON encoding
   *
   * @generated from enum value: wasimoff_provider_v1_json = 2;
   */
  wasimoff_provider_v1_json = 2,
}

/**
 * JSON type for the enum Subprotocol.
 */
export type SubprotocolJson = "UNKNOWN" | "wasimoff_provider_v1_protobuf" | "wasimoff_provider_v1_json";

/**
 * Describes the enum Subprotocol.
 */
export const SubprotocolSchema: GenEnum<Subprotocol, SubprotocolJson> = /*@__PURE__*/
  enumDesc(file_messages, 0);

/**
 * We aren't using gRPC (yet) but we can codify the expected message pairs anyway.
 * This service lists the requests that a Broker can send to the Provider, i.e. the
 * Provider (the browser) takes the role of a server here!
 *
 * @generated from service Provider
 */
export const Provider: GenService<{
  /**
   * execute
   *
   * rpc ExecuteWasm (ExecuteWasmArgs) returns (ExecuteWasmResult); // TODO
   *
   * @generated from rpc Provider.ExecuteWasi
   */
  executeWasi: {
    methodKind: "unary";
    input: typeof ExecuteWasiArgsSchema;
    output: typeof ExecuteWasiResultSchema;
  },
  /**
   * filesystem
   *
   * @generated from rpc Provider.FileProbe
   */
  fileProbe: {
    methodKind: "unary";
    input: typeof FileProbeArgsSchema;
    output: typeof FileProbeResultSchema;
  },
  /**
   * @generated from rpc Provider.FileListing
   */
  fileListing: {
    methodKind: "unary";
    input: typeof FileListingArgsSchema;
    output: typeof FileListingResultSchema;
  },
  /**
   * @generated from rpc Provider.FileUpload
   */
  fileUpload: {
    methodKind: "unary";
    input: typeof FileUploadArgsSchema;
    output: typeof FileUploadResultSchema;
  },
}> = /*@__PURE__*/
  serviceDesc(file_messages, 0);

