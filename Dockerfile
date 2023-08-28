# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2020-2023 Intel Corporation

# Build the manager binary
FROM golang:alpine3.18 as builder

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

COPY main.go main.go
COPY apis/ apis/

COPY controllers/ controllers/
COPY pkg/ pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

# Install packages in clean filesystem
FROM registry.access.redhat.com/ubi9/ubi:9.2-489 AS package_installer
RUN mkdir -p /mnt/rootfs && \
    yum install --installroot /mnt/rootfs coreutils-single glibc-minimal-langpack kmod \
        --releasever 9 --setopt install_weak_deps=false --nodocs -y && \
    yum --installroot /mnt/rootfs clean all && \
    rm -rf /mnt/rootfs/var/cache/* /mnt/rootfs/var/log/dnf* /mnt/rootfs/var/log/yum.*

FROM registry.access.redhat.com/ubi9/ubi-micro:9.2-5

ARG VERSION
### Required OpenShift Labels
LABEL name="Intel Ethernet Operator" \
    vendor="Intel Corporation" \
    version=$VERSION \
    release="1" \
    summary="Intel Ethernet Operator for E810 NICs" \
    description="The role of the Intel Ethernet Operator is to orchestrate and manage the configuration of the capabilities exposed by the Intel E810 Series network interface cards (NICs)"

USER 1001
WORKDIR /
COPY --from=package_installer /mnt/rootfs/ /
COPY --from=builder /workspace/manager .
COPY assets/ assets/

ENTRYPOINT ["/manager"]
