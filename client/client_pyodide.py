# https://protobuf.dev/getting-started/pythontutorial/
# https://www.python-httpx.org/quickstart/

import httpx, cloudpickle
from python.proto.v1.messages_pb2 import Task

def hello(name = "Alice", n = 3):
  print(f"Hello, {name}! Your lucky number is {n}.")
  return dict(name=name, n=n)

def reader():
  import sys, os
  print("environment:", os.environ)
  n = 0
  while r := sys.stdin.read(4):
    print("read:", r)
    n += len(r)
  return n

def maths(n):
  # TODO: detect loaded modules in pickled string
  # https://github.com/lithops-cloud/lithops/blob/ceebd0f2d377f583885e7a0c5bbb8bca75bf2d96/lithops/job/serialize.py#L66
  import numpy as np
  mat = np.random.randint(0, 100, (n, n))
  print(mat)
  return mat.mean()

task = [ hello, ["Bob"], dict( n=42 ) ]
task = [ reader, [], {} ]
task = [ maths, [ 10 ], { } ]

# build the pyodide request
req = Task.Pyodide.Request()
req.params.script = "print('Ooops, ran the script instead.')"
req.params.pickle = cloudpickle.dumps(task)
req.params.packages.extend([ "numpy", "cloudpickle" ])
req.params.stdin = b"Hello, World!"
req.params.envs.extend([ "TESTING=PROJECT=wasimoff", "HELLO=ok" ])
print(req)

# instantiate the client
client = httpx.Client(base_url="http://localhost:4080/api/client/")

# make the request
response = client.post("/wasimoff.v1.Tasks/RunPyodide",
  headers={ "content-type": "application/proto" },
  content=req.SerializeToString(),
  timeout=None,
)

# parse the response
res = Task.Pyodide.Response()
res.ParseFromString(response.content)

if res.WhichOneof("result") == "error":
  print(res.error)
else:
  print(res.ok)
  print(res.ok.stdout.decode())

  if res.ok.pickle:
    ret = cloudpickle.loads(res.ok.pickle)
    print("ret:", ret)
