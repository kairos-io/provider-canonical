VERSION 0.6
FROM alpine

ARG CANONICAL_VERSION=latest
ARG BASE_IMAGE=quay.io/kairos/opensuse:leap-15.5-core-amd64-generic-v2.4.3
ARG IMAGE_REPOSITORY=quay.io/kairos
ARG RELEASE_VERSION=0.4.0

ARG LUET_VERSION=0.35.1
ARG GOLINT_VERSION=v1.61.0
ARG GOLANG_VERSION=1.23

ARG CANONICAL_VERSION=latest
ARG BASE_IMAGE_NAME=$(echo $BASE_IMAGE | grep -o [^/]*: | rev | cut -c2- | rev)
ARG BASE_IMAGE_TAG=$(echo $BASE_IMAGE | grep -o :.* | cut -c2-)
ARG CANONICAL_VERSION_TAG=$(echo $CANONICAL_VERSION | sed s/+/-/)
ARG FIPS_ENABLED=false
ARG PROVIDER_IMAGE_NAME=canonical

lint:
    FROM golang:$GOLANG_VERSION
    RUN wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s ${GOLINT_VERSION}
    WORKDIR /build
    COPY . .
    RUN golangci-lint run --timeout=5m

go-deps:
    FROM us-docker.pkg.dev/palette-images/build-base-images/golang:${GOLANG_VERSION}-alpine
    WORKDIR /build
    COPY go.mod go.sum ./
    RUN go mod download
    RUN apk update
    SAVE ARTIFACT go.mod AS LOCAL go.mod
    SAVE ARTIFACT go.sum AS LOCAL go.sum

BUILD_GOLANG:
    COMMAND
    WORKDIR /build
    COPY . ./
    ARG BIN
    ARG SRC

    ARG VERSION

    ENV GO_LDFLAGS=" -X github.com/kairos-io/provider-canonical/pkg/version.Version=${VERSION} -w -s"

    IF $FIPS_ENABLED
        RUN go-build-fips.sh -a -o ${BIN} ./${SRC}
        RUN assert-fips.sh ${BIN}
        RUN assert-static.sh ${BIN}
    ELSE
        RUN go-build-static.sh -a -o ${BIN} ./${SRC}
    END

    SAVE ARTIFACT ${BIN} ${BIN} AS LOCAL build/${BIN}

VERSION:
    COMMAND
    FROM alpine
    RUN apk add git

    COPY .git/ .git

    RUN echo $(git describe --exact-match --tags || echo "v0.0.0-$(git rev-parse --short=8 HEAD)") > VERSION

    SAVE ARTIFACT VERSION VERSION

build-provider:
    DO +VERSION
    ARG VERSION=$(cat VERSION)

    FROM +go-deps
    DO +BUILD_GOLANG --BIN=agent-provider-canonical --SRC=main.go --VERSION=$VERSION

    SAVE ARTIFACT agent-provider-canonical

build-provider-package:
    DO +VERSION
    ARG TARGETARCH
    ARG VERSION=$(cat VERSION)
    FROM scratch
    COPY +build-provider/agent-provider-canonical /system/providers/agent-provider-canonical
    COPY scripts/ /opt/canonical/scripts/
    SAVE IMAGE --push $IMAGE_REPOSITORY/provider-canonical:${VERSION}-${TARGETARCH}

provider-package-merge:
    BUILD --platform=linux/amd64 --platform=linux/arm64 +provider-package-pull