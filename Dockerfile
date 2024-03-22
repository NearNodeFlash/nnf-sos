# Copyright 2020-2023 Hewlett Packard Enterprise Development LP
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

FROM golang:1.20 AS builder

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
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager cmd/main.go

ENTRYPOINT ["/bin/sh"]

FROM builder as testing
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
