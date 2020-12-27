# Build the manager binary
FROM arm32v6/golang:1.14-alpine3.11 as builder

# install the requirements for using the rpi_ws281x library
RUN apk add git gcc linux-headers scons libc-dev
RUN go get -u github.com/go-bindata/go-bindata/...

# compile and install the rpi_ws281x library into the builder image
WORKDIR /tmp
RUN ln -s /usr/bin/python3 /usr/bin/python
RUN git clone https://github.com/jgarff/rpi_ws281x.git && \
    cd rpi_ws281x && \
    scons
RUN cp /tmp/rpi_ws281x/*.a /usr/local/lib/
RUN find / -name stdio.h
RUN cp /tmp/rpi_ws281x/*.h /usr/include/

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
# Build static binary with the rpi_ws281x library built in
RUN GOOS=linux GOARCH=arm GO111MODULE=on go build -ldflags="-extldflags=-static" -a -o controller main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
#FROM gcr.io/distroless/static:nonroot
FROM discolix/static:nonroot
ARG TWC_BUILD_VERSION
ENV TWC_BUILD_VERSION=${TWC_BUILD_VERSION}
WORKDIR /
COPY --from=builder /workspace/controller .
USER root:root

ENTRYPOINT ["/controller"]