FROM golang:1.11.4 as builder_base
WORKDIR /src/github.com/mesosphere/installer

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

#This is the ‘magic’ step that will download all the dependencies that are specified in
# the go.mod and go.sum file.
# Because of how the layer caching system works in Docker, the  go mod download
# command will _ only_ be re-run when the go.mod or go.sum file change
# (or when we add another docker instruction this line)
RUN go mod download

# Build with cached dependencies
FROM builder_base as builder
ARG VERSION
ARG BUILD_DATE
ENV CGO_ENABLED=0
WORKDIR /src/github.com/dkoshkin/gofer
COPY . .
RUN make build-binary-local