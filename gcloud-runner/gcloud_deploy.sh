#!/usr/bin/env bash
set -x

gcloud run deploy \
  wasimoff-runner \
  --source=. \
  --platform managed \
  --no-allow-unauthenticated \
  --function=wasimoff \
  --min=0 \
  --concurrency=5 \
  --max-instances=1
