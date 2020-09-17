ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:glibc
LABEL maintainer="Stany MARCEL <stanypub@gmail.com>"

ARG ARCH="amd64"
ARG OS="linux"
COPY .build/${OS}-${ARCH}/govc_exporter /bin/govc_exporter

EXPOSE      9752
USER        nobody
ENTRYPOINT  [ "/bin/govc_exporter" ]
