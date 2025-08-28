#!/usr/bin/env bash
# https://docs.deno.com/examples/google_cloud_run_tutorial/
set -eu

# always change to $this directory, and then one up
cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")" )" && cd ..

# build a fresh container for faas offload
docker build --target faas -t ansemjo/wasimoff:faas .

# tag it in appropriate registry and push
IMAGE=europe-west10-docker.pkg.dev/wasimoff-faas-offload/container/cloudrunner
docker tag ansemjo/wasimoff:faas "$IMAGE"
docker push "$IMAGE"

# redeploy the function with fresh container
gcloud run deploy wasimoff-runner \
  --image="$IMAGE" \
  --region=europe-west10 \
  --no-allow-unauthenticated \
  --memory=1024Mi --cpu=2 --port=8000 \
  --min=0 --max-instances=2 \
  --concurrency=default \
  --update-env-vars=BROKER_ORIGIN=https://wasi.team
