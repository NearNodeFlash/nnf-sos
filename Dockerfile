FROM arti.dev.cray.com/baseos-docker-master-local/centos:centos7 AS base

WORKDIR /

# Install basic dependencies
RUN yum install -y wget gcc tar https://packages.endpoint.com/rhel/7/os/x86_64/endpoint-repo-1.7-1.x86_64.rpm && yum -y install git

# Retrieve lustre-rpms
RUN mkdir -p /tmp/lustre-rpms
WORKDIR /tmp/lustre-rpms

RUN wget https://downloads.whamcloud.com/public/e2fsprogs/1.45.6.wc1/el7/RPMS/x86_64/e2fsprogs-1.45.6.wc1-0.el7.x86_64.rpm \
         https://downloads.whamcloud.com/public/e2fsprogs/1.45.6.wc1/el7/RPMS/x86_64/e2fsprogs-libs-1.45.6.wc1-0.el7.x86_64.rpm \
         https://downloads.whamcloud.com/public/e2fsprogs/1.45.6.wc1/el7/RPMS/x86_64/libcom_err-1.45.6.wc1-0.el7.x86_64.rpm \
         https://downloads.whamcloud.com/public/e2fsprogs/1.45.6.wc1/el7/RPMS/x86_64/libss-1.45.6.wc1-0.el7.x86_64.rpm \
         https://downloads.whamcloud.com/public/lustre/latest-2.10-release/el7/server/RPMS/x86_64/lustre-osd-ldiskfs-mount-2.10.8-1.el7.x86_64.rpm \
         https://downloads.whamcloud.com/public/lustre/latest-2.10-release/el7/server/RPMS/x86_64/kmod-lustre-2.10.8-1.el7.x86_64.rpm \
         https://downloads.whamcloud.com/public/lustre/latest-2.10-release/el7/server/RPMS/x86_64/kmod-lustre-osd-ldiskfs-2.10.8-1.el7.x86_64.rpm \
         https://downloads.whamcloud.com/public/lustre/latest-2.10-release/el7/server/RPMS/x86_64/lustre-2.10.8-1.el7.x86_64.rpm \
         https://downloads.whamcloud.com/public/lustre/latest-2.10-release/el7/server/RPMS/x86_64/lustre-iokit-2.10.8-1.el7.x86_64.rpm

WORKDIR /

# Install Golang v1.16
RUN wget https://golang.org/dl/go1.16.7.linux-amd64.tar.gz && tar -xzf go1.16.7.linux-amd64.tar.gz

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
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

ENTRYPOINT ["/bin/sh"]


# Start from scratch
FROM arti.dev.cray.com/baseos-docker-master-local/centos:centos7 AS application

WORKDIR /
# Retrieve executable from previous layer
COPY --from=base /workspace/manager .
COPY --from=base /tmp/lustre-rpms/*.rpm /root/

# Retrieve built rpms from previous layer and install Lustre dependencies
WORKDIR /root/
RUN yum install -y git epel-release libyaml wget net-snmp tar openmpi perl-devel sg3_utils && \
    yum clean all && \
    rpm -Uivh e2fsprogs-* libcom_err-* libss-* && \
    rpm -Uivh --nodeps lustre-* kmod-* && \
    rm /root/*.rpm

ENTRYPOINT ["/manager"]
