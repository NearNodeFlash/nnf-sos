FROM arti.dev.cray.com/baseos-docker-master-local/centos:centos7 AS base

WORKDIR /

# Install basic dependencies
RUN yum install -y make git gzip wget gcc tar https://packages.endpoint.com/rhel/7/os/x86_64/endpoint-repo-1.7-1.x86_64.rpm

# Retrieve lustre-rpms
WORKDIR /tmp/lustre-rpms

RUN wget https://downloads.whamcloud.com/public/e2fsprogs/1.45.6.wc1/el7/RPMS/x86_64/e2fsprogs-1.45.6.wc1-0.el7.x86_64.rpm \
         https://downloads.whamcloud.com/public/e2fsprogs/1.45.6.wc1/el7/RPMS/x86_64/e2fsprogs-libs-1.45.6.wc1-0.el7.x86_64.rpm \
         https://downloads.whamcloud.com/public/e2fsprogs/1.45.6.wc1/el7/RPMS/x86_64/libcom_err-1.45.6.wc1-0.el7.x86_64.rpm \
         https://downloads.whamcloud.com/public/e2fsprogs/1.45.6.wc1/el7/RPMS/x86_64/libss-1.45.6.wc1-0.el7.x86_64.rpm \
         https://downloads.whamcloud.com/public/lustre/latest-2.12-release/el7/server/RPMS/x86_64/kmod-lustre-2.12.7-1.el7.x86_64.rpm \
         https://downloads.whamcloud.com/public/lustre/latest-2.12-release/el7/server/RPMS/x86_64/lustre-2.12.7-1.el7.x86_64.rpm

WORKDIR /

# Install Golang v1.16
RUN wget https://golang.org/dl/go1.16.7.linux-amd64.tar.gz && tar -xzf go1.16.7.linux-amd64.tar.gz


# Start from scratch to make the base stage for the final application.
# Build it here so it won't be invalidated when we COPY the controller source
# code in the next layer.
FROM arti.dev.cray.com/baseos-docker-master-local/centos:centos7 AS application-base

WORKDIR /
# Retrieve executable from previous layer
COPY --from=base /tmp/lustre-rpms/*.rpm /root/

# Retrieve built rpms from previous layer and install Lustre dependencies
WORKDIR /root/
RUN yum install -y epel-release libyaml net-snmp openmpi sg3_utils && \
    yum clean all && \
    rpm -Uivh e2fsprogs-* libcom_err-* libss-* && \
    rpm -Uivh --nodeps lustre-* kmod-* && \
    rm /root/*.rpm

ENTRYPOINT ["/bin/sh"]


#
# Note: The COPY commands below have the potential to invalidate any layer
# that follows.
#

FROM base as builder

# Set Go environment
ENV GOROOT="/go"
ENV PATH="${PATH}:${GOROOT}/bin" GOPRIVATE="stash.us.cray.com"

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY vendor/ vendor/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

ENTRYPOINT ["/bin/sh"]

FROM builder as testing
WORKDIR /workspace

COPY .dws-operator/ .dws-operator/
COPY hack/ hack/
COPY runContainerTest.sh .
COPY Makefile .

RUN go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest && \
    make manifests && make generate && make fmt &&  make vet && \
    mkdir -p /workspace/testbin && /bin/bash -c "test -f /workspace/testbin/setup-envtest.sh || curl -sSLo /workspace/testbin/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh" && \
    /bin/bash -c "source /workspace/testbin/setup-envtest.sh; fetch_envtest_tools /workspace/testbin; setup_envtest_env /workspace/testbin"

ENTRYPOINT ["sh", "/workspace/runContainerTest.sh"]

# The final application stage.
FROM application-base

WORKDIR /
# Retrieve executable from previous layer
COPY --from=builder /workspace/manager .

ENTRYPOINT ["/manager"]
