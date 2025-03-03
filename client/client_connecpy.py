# https://protobuf.dev/getting-started/pythontutorial/
# https://www.python-httpx.org/quickstart/

import httpx
from python.proto.v1.messages_pb2 import Task

from connecpy.context import ClientContext
from python.proto.v1.messages_connecpy import AsyncTasksClient

async def main():

  base = "http://localhost:4080/api/client/"
  session = httpx.AsyncClient(base_url=base)
  client = AsyncTasksClient(base, session=session)

  # build the request
  req = Task.Wasip1.Request()
  req.params.binary.ref = "tsp.wasm"
  req.params.args.extend(["tsp.wasm", "rand", "10"])
  print(req)

  response = await client.RunWasip1(
    ctx=ClientContext(),
    request=req,
  )

  # parse the response
  #res = Task.Wasip1.Response()
  #res.ParseFromString(response.content)
  print(response)
