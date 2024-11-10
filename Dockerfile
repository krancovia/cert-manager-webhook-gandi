####################################################################################################
# build
####################################################################################################
FROM --platform=$BUILDPLATFORM golang:1.23.3-bookworm AS build

ARG TARGETOS
ARG TARGETARCH

ARG VERSION_PACKAGE=github.com/krancouvia/cert-manager-webhook-gandi/internal/version

ARG CGO_ENABLED=0

WORKDIR /cert-manager-webhook-gandi
COPY ["go.mod", "go.sum", "./"]
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/

ARG VERSION
ARG GIT_COMMIT
ARG GIT_TREE_STATE

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
      -ldflags "-w -X ${VERSION_PACKAGE}.version=${VERSION} -X ${VERSION_PACKAGE}.buildDate=$(date -u +'%Y-%m-%dT%H:%M:%SZ') -X ${VERSION_PACKAGE}.gitCommit=${GIT_COMMIT} -X ${VERSION_PACKAGE}.gitTreeState=${GIT_TREE_STATE}" \
      -o bin/cert-manager-webhook-gandi \
      ./cmd

####################################################################################################
# dev
# - relies on go build that runs on host
# - supports development
# - not used for official image builds
####################################################################################################
FROM --platform=$BUILDPLATFORM golang:1.23.3-bookworm AS dev

COPY bin/cert-manager-webhook-gandi /usr/local/bin/cert-manager-webhook-gandi

RUN adduser -D -H -u 1000 nonroot
USER 1000:0

CMD ["/usr/local/bin/cert-manager-webhook-gandi"]

####################################################################################################
# final
# - the official image we publish
# - purposefully last so that it is the default target when building
####################################################################################################
FROM cgr.dev/chainguard/static:latest AS final

ENV HOME /home/nonroot

COPY --from=build /cert-manager-webhook-gandi/ /usr/local/bin/

CMD ["/usr/local/bin/cert-manager-webhook-gandi"]
