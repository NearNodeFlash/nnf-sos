# Copyright 2020, 2021, 2022 Hewlett Packard Enterprise Development LP
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

FROM golang:1.17 AS builder

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY vendor/ vendor/
COPY config/ config/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

ENTRYPOINT ["/bin/sh"]

FROM builder as testing
WORKDIR /workspace

ARG FAILFAST
COPY hack/ hack/
COPY test-tools.sh .
COPY initiateContainerTest.sh .
COPY Makefile .

RUN echo "building test target after copy" && pwd && ls -al

RUN go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest && \
    make manifests && make generate && make fmt &&  make vet && \
    mkdir -p /workspace/testbin && /bin/bash -c "test -f /workspace/testbin/setup-envtest.sh || curl -sSLo /workspace/testbin/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh" && \
    /bin/bash -c "source /workspace/testbin/setup-envtest.sh; fetch_envtest_tools /workspace/testbin; setup_envtest_env /workspace/testbin"

ENTRYPOINT ["bash", "/workspace/initiateContainerTest.sh"]

# The final application stage.
FROM redhat/ubi8-minimal

WORKDIR /
# Retrieve executable from previous layer
COPY --from=builder /workspace/manager .

ENTRYPOINT ["/manager"]
