# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Multi-stage build to create the binary
FROM golang:1.24 AS builder
WORKDIR /go/src/dynamodb-adapter

# Force the go compiler to use modules
ENV GO111MODULE=on

COPY . .

#This is the 'magic' step that will download all the dependencies that are specified in
# the go.mod and go.sum file.
# Because of how the layer caching system works in Docker, the  go mod download 
# command will _ only_ be re-run when the go.mod or go.sum file change 
# (or when we add another docker instruction this line)
RUN go mod download

# Build the package
ARG PROXY_RELEASE_VERSION
RUN go build \
  -ldflags "-X github.com/cloudspannerecosystem/dynamodb-adapter/config.proxyReleaseVersion=${PROXY_RELEASE_VERSION}" \
  -o dynamodb-adapter

# Multi-stage build to create a minimal runtime image
# Run the executable
FROM debian:bookworm-slim
WORKDIR /app
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /go/src/dynamodb-adapter/dynamodb-adapter .
COPY config.yaml .
CMD ["./dynamodb-adapter"]
