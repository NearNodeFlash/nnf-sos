# Copyright 2020-2024 Hewlett Packard Enterprise Development LP
# Other additional copyright holders may be indicated within.
#
# The entirety of this work is licensed under the Apache License,
# Version 2.0 (the "License"); you may not use this file except
# in compliance with the License.
#
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.21 AS builder

ARG TARGETARCH
ARG TARGETOS

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Copy the go source
COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/
COPY pkg/ pkg/
COPY vendor/ vendor/
COPY config/ config/

# Build
# the GOARCH has a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go

ENTRYPOINT ["/bin/sh"]

FROM builder AS testing
WORKDIR /workspace

ARG FAILFAST
COPY hack/ hack/
COPY test-tools.sh .
COPY Makefile .

RUN echo "building test target after copy" && pwd && ls -al

ENTRYPOINT ["make", "test"]

# The final application stage.
FROM redhat/ubi8-minimal

WORKDIR /
# Retrieve executable from previous layer
COPY --from=builder /workspace/manager .

ENTRYPOINT ["/manager"]
