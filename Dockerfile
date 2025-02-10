ARG DOCKERHUB_MIRROR=""
ARG IMAGE_UBUNTU=ubuntu:25.04@sha256:008b026f11c0b5653d564d0c9877a116770f06dfbdb36ca75c46fd593d863cbc

FROM ${DOCKERHUB_MIRROR}${IMAGE_UBUNTU} AS runner
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update -qqy \
 && apt-get install -qqy iperf3 \
 && iperf3 --version \
 && apt-get clean \
 && rm -rf /var/cache/apt
COPY cni-benchmark-operator /cni-benchmark-operator
ENTRYPOINT ["/cni-benchmark-operator"]
USER 65534:65534
