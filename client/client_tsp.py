# https://protobuf.dev/getting-started/pythontutorial/
# https://www.python-httpx.org/quickstart/

import httpx
from python.proto.v1.messages_pb2 import Task

# build the request
req = Task.Wasip1.Request()
req.params.binary.ref = "tsp.wasm"
req.params.args.extend(["tsp.wasm", "rand", "10"])
print(req)

# instantiate the client
client = httpx.Client(base_url="http://localhost:4080/api/client/")

# make the request
response = client.post("/wasimoff.v1.Tasks/RunWasip1",
  headers={ "content-type": "application/proto" },
  content=req.SerializeToString(),
)

# parse the response
res = Task.Wasip1.Response()
res.ParseFromString(response.content)
print(res)
