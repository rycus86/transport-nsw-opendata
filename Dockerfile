FROM golang:1.11 as builder

ARG CC=""
ARG CC_PKG=""
ARG CC_GOARCH=""

ADD . /go/src/github.com/rycus86/transport-nsw-opendata
WORKDIR /go/src/github.com/rycus86/transport-nsw-opendata

RUN if [ -n "$CC_PKG" ]; then \
      apt-get update && apt-get install -y $CC_PKG; \
    fi \
    && export CC=$CC \
    && export GOOS=linux \
    && export GOARCH=$CC_GOARCH \
    && export CGO_ENABLED=0 \
    && export GO111MODULE=on \
    && go build -o /var/out/main -v ./cmd

FROM scratch

ARG VERSION="dev"
ARG BUILD_ARCH="unknown"
ARG GIT_COMMIT="unknown"
ARG BUILD_TIMESTAMP="unknown"

ENV VERSION="$VERSION"
ENV BUILD_ARCH="$BUILD_ARCH"
ENV GIT_COMMIT="$GIT_COMMIT"
ENV BUILD_TIMESTAMP="$BUILD_TIMESTAMP"

LABEL maintainer="Viktor Adam <rycus86@gmail.com>"

LABEL com.github.rycus86.transport-nsw-opendata.version="$VERSION"
LABEL com.github.rycus86.transport-nsw-opendata.commit="$GIT_COMMIT"

COPY --from=builder /var/out/main  /tfnsw-server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip /usr/local/go/lib/time/zoneinfo.zip

ENTRYPOINT [ "/tfnsw-server" ]
