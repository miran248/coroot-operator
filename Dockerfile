FROM golang:1.23 AS builder

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY main.go main.go
COPY api/ api/
COPY controller/ controller/

RUN CGO_ENABLED=0 go build -a -o coroot-operator main.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/coroot-operator /usr/bin/coroot-operator
USER 65534:65534

ENTRYPOINT ["coroot-operator"]
