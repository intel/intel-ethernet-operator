# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2020-2023 Intel Corporation

# Build the manager binary
FROM docker.io/golang:alpine3.18 as builder

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

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

FROM registry.access.redhat.com/ubi9/ubi-micro:9.2-9

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
COPY --from=builder /workspace/manager .
COPY assets/ assets/

USER root
RUN mkdir /licenses
COPY LICENSE /licenses/LICENSE.txt
RUN chown 1001 /licenses
USER 1001

ENTRYPOINT ["/manager"]
