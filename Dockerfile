FROM golang:1.23 AS builder

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY main.go main.go
COPY api/ api/
COPY controller/ controller/

RUN CGO_ENABLED=0 go build -a -o coroot-operator main.go

FROM registry.access.redhat.com/ubi9/ubi

ARG VERSION=unknown
LABEL name="coroot-operator" \
      vendor="Coroot, Inc." \
      version=${VERSION} \
      summary="Coroot Operator."

COPY LICENSE /licenses/LICENSE

WORKDIR /
COPY --from=builder /workspace/coroot-operator /usr/bin/coroot-operator
USER 65534:65534
ENTRYPOINT ["coroot-operator"]
