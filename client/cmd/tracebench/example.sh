#!/usr/bin/env bash

go run ./  \
  -timeout 10 \
  -scale-rate 10 \
  -scale-tasklen 0.1 \
  -broker http://localhost:4080 \
    21
