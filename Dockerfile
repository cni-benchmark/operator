ARG DOCKERHUB_MIRROR=""
ARG IMAGE_UBUNTU=ubuntu:25.04@sha256:008b026f11c0b5653d564d0c9877a116770f06dfbdb36ca75c46fd593d863cbc
ARG IMAGE_GOLANG=golang:1.23.6-alpine3.21@sha256:2c49857f2295e89b23b28386e57e018a86620a8fede5003900f2d138ba9c4037

#  ┌─┐┌─┐┬  ┬─┐┌┐┐┌─┐
#  │ ┬│ ││  │─┤││││ ┬
#  ┘─┘┘─┘┘─┘┘ ┘┘└┘┘─┘

FROM ${DOCKERHUB_MIRROR}${IMAGE_GOLANG} AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /app
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY ./cmd ./
COPY ./pkg ./
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o /build cmd/main.go

#  ┬ ┐┬─┐┬ ┐┌┐┐┌┐┐┬ ┐
#  │ ││─││ ││││ │ │ │
#  ┘─┘┘─┘┘─┘┘└┘ ┘ ┘─┘

FROM ${DOCKERHUB_MIRROR}${IMAGE_UBUNTU} AS runner
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update -qqy \
 && apt-get install -qqy iperf3 \
 && iperf3 --version \
 && apt-get clean \
 && rm -rf /var/cache/apt
COPY --from=builder --chown=root:root --chmod=0755 /build /cni-benchmark
ENTRYPOINT ["/cni-benchmark"]
USER 65534:65534
