# Build the manager binary
FROM arm32v6/golang:1.14-alpine3.11 as builder

RUN apk add git
RUN go get -u github.com/go-bindata/go-bindata/...

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY internal/ internal/

# Generate
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm GO111MODULE=on go generate
# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm GO111MODULE=on go build -a -o controller main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
#FROM gcr.io/distroless/static:nonroot
FROM discolix/static:nonroot
WORKDIR /
COPY --from=builder /workspace/controller .
USER root:root

ENTRYPOINT ["/controller"]