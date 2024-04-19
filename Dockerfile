# syntax=docker/dockerfile:1

ARG GO_VERSION=1

# setup the base environment.
FROM --platform=${BUILDPLATFORM} golang:${GO_VERSION} AS base

WORKDIR /fn
ENV CGO_ENABLED=0

ARG LDFLAGS="-X 'main.Version=unknown'"

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# build the function.
FROM base AS build
ARG TARGETOS
ARG TARGETARCH
RUN --mount=target=. \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="${LDFLAGS}" -o /function ./

# produce the function image.
FROM gcr.io/distroless/base-debian11 AS image
WORKDIR /
COPY --from=build /function ./
EXPOSE 9443
USER nonroot:nonroot
ENTRYPOINT ["/function"]
