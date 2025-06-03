#!/usr/bin/env bash
set -eu

# get a current auth token
TOKEN=$(gcloud auth print-identity-token)

# get the function endpoint
URL=$(gcloud run services describe wasimoff-runner --format json | jq -r .status.url)

# use this binary ref for https://wasi.team/api/storage/tsp.wasm
wasm="sha256:d2ee7b6b9507babe659f9fd9356221f989123b02f5bb7130bd95489351ce44ab"

# post the offloading request
exec curl -f -X POST "$URL" \
  -H "Authorization: bearer $TOKEN" \
  -H "Content-Type: application/json" \
  --data-raw "$(jq -n --arg wasm "$wasm" --args \
'{
  "info": {
    "id": "invoke"
  },
  "params": {
    "binary": { "ref": $wasm },
    "args": [ "tsp.wasm", $ARGS.positional[] ]
  }
}' \
  "$@" )"
