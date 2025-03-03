# https://protobuf.dev/getting-started/pythontutorial/
# https://www.python-httpx.org/quickstart/

import httpx, cloudpickle
from python.proto.v1.messages_pb2 import Task

def hello(name: str):
  print(f"Hello, {name}!")

# build the pyodide request
req = Task.Pyodide.Request()
req.params.script = "print(p('World'))"
req.params.pickle = cloudpickle.dumps(hello)
print(req)

# instantiate the client
client = httpx.Client(base_url="http://localhost:4080/api/client/")

# make the request
response = client.post("/wasimoff.v1.Tasks/RunPyodide",
  headers={ "content-type": "application/proto" },
  content=req.SerializeToString(),
)

# parse the response
res = Task.Pyodide.Response()
res.ParseFromString(response.content)
print(res)
