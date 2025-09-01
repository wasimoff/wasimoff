# =========================================================================== #
# ---> build the broker binary
FROM golang:1.24-bookworm AS brokerbuild

# compile the binary
COPY ./ /build
WORKDIR /build/broker
RUN CGO_ENABLED=0 go build -o broker

# =========================================================================== #
# ---> build the webprovider frontend dist
FROM node:22-bookworm AS frontend

# compile the frontend
COPY ./ /build
WORKDIR /build/webprovider
RUN yarn install && yarn build

# =========================================================================== #
# ---> build denoprovider for the terminal
# docker build --target provider -t wasimoff/provider .
FROM denoland/deno:distroless AS provider

# copy files
COPY ./denoprovider  /app/deno
COPY ./webprovider   /app/webprovider

WORKDIR /app/deno

# cache required dependencies
RUN ["deno", "cache", "--sloppy-imports", "main.ts"]

# launch configuration
ENTRYPOINT ["/tini", "--", "deno", "run", \
  "--cached-only", "--no-prompt", "--sloppy-imports", \
  "--allow-env", "--allow-net", \
  "--allow-read=/app,/deno-dir/npm/registry.npmjs.org/pyodide/", \
  "--allow-write=/deno-dir/npm/registry.npmjs.org/pyodide/", \
  "main.ts"]

# =========================================================================== #
# ---> build a deno image for google cloud run
# docker build --target faas -t wasimoff/faas .
FROM provider AS faas

# install and cache dependencies
RUN ["deno", "cache", "--sloppy-imports", "cloudrun.ts"]

# launch configuration
ENTRYPOINT ["/tini", "--", "deno", "run", \
  "--cached-only", "--no-prompt", "--sloppy-imports", \
  "--allow-env", "--allow-net", \
  "--allow-read=/app,/deno-dir/npm/registry.npmjs.org/pyodide/", \
  "--allow-write=/deno-dir/npm/registry.npmjs.org/pyodide/", \
  "cloudrun.ts"]

# =========================================================================== #
# ---> combine broker and frontend dist in default container
# docker build --target broker -t wasimoff/broker .
FROM alpine AS broker
COPY --from=brokerbuild  /build/broker/broker /broker
COPY --from=frontend     /build/webprovider/dist /provider
ENTRYPOINT [ "/broker" ]

# needed for healthcheck
RUN apk add --no-cache curl

# :: minimum container configuration ::

# the TCP port to listen on with the HTTP server
ENV WASIMOFF_HTTP_LISTEN=":4080"

# filesystem path to frontend dist to be served
ENV WASIMOFF_STATIC_FILES="/provider"
