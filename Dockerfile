# Go version is pinned here and in go.mod, keep both in sync.
ARG GO_VERSION=1.27.3

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS builder
ARG TARGETOS
ARG TARGETARCH
ARG REVISION=unknown
ARG BRANCH=unknown
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -mod=vendor -trimpath -tags netgo \
    -ldflags "-s -w \
      -X github.com/prometheus/common/version.Version=$(cat VERSION) \
      -X github.com/prometheus/common/version.Revision=${REVISION} \
      -X github.com/prometheus/common/version.Branch=${BRANCH} \
      -X github.com/prometheus/common/version.BuildUser=docker \
      -X github.com/prometheus/common/version.BuildDate=$(date -u +%Y%m%d-%H:%M:%S)" \
    -o /out/govc_exporter .

FROM quay.io/prometheus/busybox:glibc
LABEL maintainer="Stany MARCEL <stanypub@gmail.com>"

COPY --from=builder /out/govc_exporter /bin/govc_exporter

EXPOSE      9752
USER        nobody
ENTRYPOINT  [ "/bin/govc_exporter" ]
