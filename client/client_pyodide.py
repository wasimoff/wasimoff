# https://protobuf.dev/getting-started/pythontutorial/
# https://www.python-httpx.org/quickstart/

import httpx, cloudpickle

# import the protobuf definitions from parent directory
import sys, os
parent = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
sys.path.insert(0, parent)
from proto.v1.messages_pb2 import Task

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

def lsdir(path):
  import os
  try:
    for root, dirs, files in os.walk(path):
      print(f"\n{root}/")
      for entry in dirs + files:
        m = os.stat(os.path.join(root, entry))
        perms = oct(m.st_mode)[-3:]
        print(f" {m.st_size:10d}  {perms}  {entry}")
  except Exception as e:
    print(f"Oops: {e}")

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
task = [ lsdir, ["/mnt"], { } ]

# construct a zip file in memory
import zipfile, io
buf = io.BytesIO()
zf = zipfile.ZipFile(buf, "w", zipfile.ZIP_DEFLATED)
with zf.open("hello.txt", mode="w") as f:
  f.write(b"Hello, World!")
zf.close()

# build the pyodide request
req = Task.Pyodide.Request()
req.params.script = "print('Ooops, ran the script instead.')"
req.params.pickle = cloudpickle.dumps(task)
req.params.packages.extend([ "numpy", "cloudpickle" ])
req.params.stdin = b"Hello, World!"
req.params.envs.extend([ "TESTING=PROJECT=wasimoff", "HELLO=ok" ])
req.params.rootfs.blob = bytes(buf.getbuffer())
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
