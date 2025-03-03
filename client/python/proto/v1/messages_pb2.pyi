from google.protobuf import any_pb2 as _any_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Subprotocol(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    UNKNOWN: _ClassVar[Subprotocol]
    wasimoff_provider_v1_protobuf: _ClassVar[Subprotocol]
    wasimoff_provider_v1_json: _ClassVar[Subprotocol]
UNKNOWN: Subprotocol
wasimoff_provider_v1_protobuf: Subprotocol
wasimoff_provider_v1_json: Subprotocol

class Envelope(_message.Message):
    __slots__ = ("sequence", "type", "error", "payload")
    class MessageType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        UNKNOWN: _ClassVar[Envelope.MessageType]
        Request: _ClassVar[Envelope.MessageType]
        Response: _ClassVar[Envelope.MessageType]
        Event: _ClassVar[Envelope.MessageType]
    UNKNOWN: Envelope.MessageType
    Request: Envelope.MessageType
    Response: Envelope.MessageType
    Event: Envelope.MessageType
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    sequence: int
    type: Envelope.MessageType
    error: str
    payload: _any_pb2.Any
    def __init__(self, sequence: _Optional[int] = ..., type: _Optional[_Union[Envelope.MessageType, str]] = ..., error: _Optional[str] = ..., payload: _Optional[_Union[_any_pb2.Any, _Mapping]] = ...) -> None: ...

class Task(_message.Message):
    __slots__ = ()
    class Metadata(_message.Message):
        __slots__ = ("id", "requester", "provider")
        ID_FIELD_NUMBER: _ClassVar[int]
        REQUESTER_FIELD_NUMBER: _ClassVar[int]
        PROVIDER_FIELD_NUMBER: _ClassVar[int]
        id: str
        requester: str
        provider: str
        def __init__(self, id: _Optional[str] = ..., requester: _Optional[str] = ..., provider: _Optional[str] = ...) -> None: ...
    class QoS(_message.Message):
        __slots__ = ("priority", "deadline")
        PRIORITY_FIELD_NUMBER: _ClassVar[int]
        DEADLINE_FIELD_NUMBER: _ClassVar[int]
        priority: bool
        deadline: _timestamp_pb2.Timestamp
        def __init__(self, priority: bool = ..., deadline: _Optional[_Union[_timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
    class Cancel(_message.Message):
        __slots__ = ("id", "reason")
        ID_FIELD_NUMBER: _ClassVar[int]
        REASON_FIELD_NUMBER: _ClassVar[int]
        id: str
        reason: str
        def __init__(self, id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...
    class Wasip1(_message.Message):
        __slots__ = ()
        class Params(_message.Message):
            __slots__ = ("binary", "args", "envs", "stdin", "rootfs", "artifacts")
            BINARY_FIELD_NUMBER: _ClassVar[int]
            ARGS_FIELD_NUMBER: _ClassVar[int]
            ENVS_FIELD_NUMBER: _ClassVar[int]
            STDIN_FIELD_NUMBER: _ClassVar[int]
            ROOTFS_FIELD_NUMBER: _ClassVar[int]
            ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
            binary: File
            args: _containers.RepeatedScalarFieldContainer[str]
            envs: _containers.RepeatedScalarFieldContainer[str]
            stdin: bytes
            rootfs: File
            artifacts: _containers.RepeatedScalarFieldContainer[str]
            def __init__(self, binary: _Optional[_Union[File, _Mapping]] = ..., args: _Optional[_Iterable[str]] = ..., envs: _Optional[_Iterable[str]] = ..., stdin: _Optional[bytes] = ..., rootfs: _Optional[_Union[File, _Mapping]] = ..., artifacts: _Optional[_Iterable[str]] = ...) -> None: ...
        class Output(_message.Message):
            __slots__ = ("status", "stdout", "stderr", "artifacts")
            STATUS_FIELD_NUMBER: _ClassVar[int]
            STDOUT_FIELD_NUMBER: _ClassVar[int]
            STDERR_FIELD_NUMBER: _ClassVar[int]
            ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
            status: int
            stdout: bytes
            stderr: bytes
            artifacts: File
            def __init__(self, status: _Optional[int] = ..., stdout: _Optional[bytes] = ..., stderr: _Optional[bytes] = ..., artifacts: _Optional[_Union[File, _Mapping]] = ...) -> None: ...
        class Request(_message.Message):
            __slots__ = ("info", "qos", "params")
            INFO_FIELD_NUMBER: _ClassVar[int]
            QOS_FIELD_NUMBER: _ClassVar[int]
            PARAMS_FIELD_NUMBER: _ClassVar[int]
            info: Task.Metadata
            qos: Task.QoS
            params: Task.Wasip1.Params
            def __init__(self, info: _Optional[_Union[Task.Metadata, _Mapping]] = ..., qos: _Optional[_Union[Task.QoS, _Mapping]] = ..., params: _Optional[_Union[Task.Wasip1.Params, _Mapping]] = ...) -> None: ...
        class JobRequest(_message.Message):
            __slots__ = ("info", "qos", "parent", "tasks")
            INFO_FIELD_NUMBER: _ClassVar[int]
            QOS_FIELD_NUMBER: _ClassVar[int]
            PARENT_FIELD_NUMBER: _ClassVar[int]
            TASKS_FIELD_NUMBER: _ClassVar[int]
            info: Task.Metadata
            qos: Task.QoS
            parent: Task.Wasip1.Params
            tasks: _containers.RepeatedCompositeFieldContainer[Task.Wasip1.Params]
            def __init__(self, info: _Optional[_Union[Task.Metadata, _Mapping]] = ..., qos: _Optional[_Union[Task.QoS, _Mapping]] = ..., parent: _Optional[_Union[Task.Wasip1.Params, _Mapping]] = ..., tasks: _Optional[_Iterable[_Union[Task.Wasip1.Params, _Mapping]]] = ...) -> None: ...
        class Response(_message.Message):
            __slots__ = ("info", "error", "ok")
            INFO_FIELD_NUMBER: _ClassVar[int]
            ERROR_FIELD_NUMBER: _ClassVar[int]
            OK_FIELD_NUMBER: _ClassVar[int]
            info: Task.Metadata
            error: str
            ok: Task.Wasip1.Output
            def __init__(self, info: _Optional[_Union[Task.Metadata, _Mapping]] = ..., error: _Optional[str] = ..., ok: _Optional[_Union[Task.Wasip1.Output, _Mapping]] = ...) -> None: ...
        class JobResponse(_message.Message):
            __slots__ = ("info", "error", "tasks")
            INFO_FIELD_NUMBER: _ClassVar[int]
            ERROR_FIELD_NUMBER: _ClassVar[int]
            TASKS_FIELD_NUMBER: _ClassVar[int]
            info: Task.Metadata
            error: str
            tasks: _containers.RepeatedCompositeFieldContainer[Task.Wasip1.Response]
            def __init__(self, info: _Optional[_Union[Task.Metadata, _Mapping]] = ..., error: _Optional[str] = ..., tasks: _Optional[_Iterable[_Union[Task.Wasip1.Response, _Mapping]]] = ...) -> None: ...
        def __init__(self) -> None: ...
    class Pyodide(_message.Message):
        __slots__ = ()
        class Params(_message.Message):
            __slots__ = ("script", "packages", "pickle")
            SCRIPT_FIELD_NUMBER: _ClassVar[int]
            PACKAGES_FIELD_NUMBER: _ClassVar[int]
            PICKLE_FIELD_NUMBER: _ClassVar[int]
            script: str
            packages: _containers.RepeatedScalarFieldContainer[str]
            pickle: bytes
            def __init__(self, script: _Optional[str] = ..., packages: _Optional[_Iterable[str]] = ..., pickle: _Optional[bytes] = ...) -> None: ...
        class Output(_message.Message):
            __slots__ = ("pickle", "stdout", "stderr", "version")
            PICKLE_FIELD_NUMBER: _ClassVar[int]
            STDOUT_FIELD_NUMBER: _ClassVar[int]
            STDERR_FIELD_NUMBER: _ClassVar[int]
            VERSION_FIELD_NUMBER: _ClassVar[int]
            pickle: bytes
            stdout: bytes
            stderr: bytes
            version: str
            def __init__(self, pickle: _Optional[bytes] = ..., stdout: _Optional[bytes] = ..., stderr: _Optional[bytes] = ..., version: _Optional[str] = ...) -> None: ...
        class Request(_message.Message):
            __slots__ = ("info", "qos", "params")
            INFO_FIELD_NUMBER: _ClassVar[int]
            QOS_FIELD_NUMBER: _ClassVar[int]
            PARAMS_FIELD_NUMBER: _ClassVar[int]
            info: Task.Metadata
            qos: Task.QoS
            params: Task.Pyodide.Params
            def __init__(self, info: _Optional[_Union[Task.Metadata, _Mapping]] = ..., qos: _Optional[_Union[Task.QoS, _Mapping]] = ..., params: _Optional[_Union[Task.Pyodide.Params, _Mapping]] = ...) -> None: ...
        class JobRequest(_message.Message):
            __slots__ = ("info", "qos", "parent", "tasks")
            INFO_FIELD_NUMBER: _ClassVar[int]
            QOS_FIELD_NUMBER: _ClassVar[int]
            PARENT_FIELD_NUMBER: _ClassVar[int]
            TASKS_FIELD_NUMBER: _ClassVar[int]
            info: Task.Metadata
            qos: Task.QoS
            parent: Task.Pyodide.Params
            tasks: _containers.RepeatedCompositeFieldContainer[Task.Pyodide.Params]
            def __init__(self, info: _Optional[_Union[Task.Metadata, _Mapping]] = ..., qos: _Optional[_Union[Task.QoS, _Mapping]] = ..., parent: _Optional[_Union[Task.Pyodide.Params, _Mapping]] = ..., tasks: _Optional[_Iterable[_Union[Task.Pyodide.Params, _Mapping]]] = ...) -> None: ...
        class Response(_message.Message):
            __slots__ = ("info", "error", "ok")
            INFO_FIELD_NUMBER: _ClassVar[int]
            ERROR_FIELD_NUMBER: _ClassVar[int]
            OK_FIELD_NUMBER: _ClassVar[int]
            info: Task.Metadata
            error: str
            ok: Task.Pyodide.Output
            def __init__(self, info: _Optional[_Union[Task.Metadata, _Mapping]] = ..., error: _Optional[str] = ..., ok: _Optional[_Union[Task.Pyodide.Output, _Mapping]] = ...) -> None: ...
        class JobResponse(_message.Message):
            __slots__ = ("info", "error", "tasks")
            INFO_FIELD_NUMBER: _ClassVar[int]
            ERROR_FIELD_NUMBER: _ClassVar[int]
            TASKS_FIELD_NUMBER: _ClassVar[int]
            info: Task.Metadata
            error: str
            tasks: _containers.RepeatedCompositeFieldContainer[Task.Pyodide.Response]
            def __init__(self, info: _Optional[_Union[Task.Metadata, _Mapping]] = ..., error: _Optional[str] = ..., tasks: _Optional[_Iterable[_Union[Task.Pyodide.Response, _Mapping]]] = ...) -> None: ...
        def __init__(self) -> None: ...
    def __init__(self) -> None: ...

class File(_message.Message):
    __slots__ = ("ref", "media", "blob")
    REF_FIELD_NUMBER: _ClassVar[int]
    MEDIA_FIELD_NUMBER: _ClassVar[int]
    BLOB_FIELD_NUMBER: _ClassVar[int]
    ref: str
    media: str
    blob: bytes
    def __init__(self, ref: _Optional[str] = ..., media: _Optional[str] = ..., blob: _Optional[bytes] = ...) -> None: ...

class Filesystem(_message.Message):
    __slots__ = ()
    class Listing(_message.Message):
        __slots__ = ()
        class Request(_message.Message):
            __slots__ = ()
            def __init__(self) -> None: ...
        class Response(_message.Message):
            __slots__ = ("files",)
            FILES_FIELD_NUMBER: _ClassVar[int]
            files: _containers.RepeatedScalarFieldContainer[str]
            def __init__(self, files: _Optional[_Iterable[str]] = ...) -> None: ...
        def __init__(self) -> None: ...
    class Probe(_message.Message):
        __slots__ = ()
        class Request(_message.Message):
            __slots__ = ("file",)
            FILE_FIELD_NUMBER: _ClassVar[int]
            file: str
            def __init__(self, file: _Optional[str] = ...) -> None: ...
        class Response(_message.Message):
            __slots__ = ("ok",)
            OK_FIELD_NUMBER: _ClassVar[int]
            ok: bool
            def __init__(self, ok: bool = ...) -> None: ...
        def __init__(self) -> None: ...
    class Upload(_message.Message):
        __slots__ = ()
        class Request(_message.Message):
            __slots__ = ("upload",)
            UPLOAD_FIELD_NUMBER: _ClassVar[int]
            upload: File
            def __init__(self, upload: _Optional[_Union[File, _Mapping]] = ...) -> None: ...
        class Response(_message.Message):
            __slots__ = ("ref",)
            REF_FIELD_NUMBER: _ClassVar[int]
            ref: str
            def __init__(self, ref: _Optional[str] = ...) -> None: ...
        def __init__(self) -> None: ...
    class Download(_message.Message):
        __slots__ = ()
        class Request(_message.Message):
            __slots__ = ("file",)
            FILE_FIELD_NUMBER: _ClassVar[int]
            file: str
            def __init__(self, file: _Optional[str] = ...) -> None: ...
        class Response(_message.Message):
            __slots__ = ("download", "err")
            DOWNLOAD_FIELD_NUMBER: _ClassVar[int]
            ERR_FIELD_NUMBER: _ClassVar[int]
            download: File
            err: str
            def __init__(self, download: _Optional[_Union[File, _Mapping]] = ..., err: _Optional[str] = ...) -> None: ...
        def __init__(self) -> None: ...
    def __init__(self) -> None: ...

class Event(_message.Message):
    __slots__ = ()
    class GenericMessage(_message.Message):
        __slots__ = ("message",)
        MESSAGE_FIELD_NUMBER: _ClassVar[int]
        message: str
        def __init__(self, message: _Optional[str] = ...) -> None: ...
    class ProviderHello(_message.Message):
        __slots__ = ("name", "useragent")
        NAME_FIELD_NUMBER: _ClassVar[int]
        USERAGENT_FIELD_NUMBER: _ClassVar[int]
        name: str
        useragent: str
        def __init__(self, name: _Optional[str] = ..., useragent: _Optional[str] = ...) -> None: ...
    class ProviderResources(_message.Message):
        __slots__ = ("concurrency", "tasks")
        CONCURRENCY_FIELD_NUMBER: _ClassVar[int]
        TASKS_FIELD_NUMBER: _ClassVar[int]
        concurrency: int
        tasks: int
        def __init__(self, concurrency: _Optional[int] = ..., tasks: _Optional[int] = ...) -> None: ...
    class ClusterInfo(_message.Message):
        __slots__ = ("providers",)
        PROVIDERS_FIELD_NUMBER: _ClassVar[int]
        providers: int
        def __init__(self, providers: _Optional[int] = ...) -> None: ...
    class Throughput(_message.Message):
        __slots__ = ("overall", "yours")
        OVERALL_FIELD_NUMBER: _ClassVar[int]
        YOURS_FIELD_NUMBER: _ClassVar[int]
        overall: float
        yours: float
        def __init__(self, overall: _Optional[float] = ..., yours: _Optional[float] = ...) -> None: ...
    class FileSystemUpdate(_message.Message):
        __slots__ = ("added", "removed")
        ADDED_FIELD_NUMBER: _ClassVar[int]
        REMOVED_FIELD_NUMBER: _ClassVar[int]
        added: _containers.RepeatedScalarFieldContainer[str]
        removed: _containers.RepeatedScalarFieldContainer[str]
        def __init__(self, added: _Optional[_Iterable[str]] = ..., removed: _Optional[_Iterable[str]] = ...) -> None: ...
    def __init__(self) -> None: ...
