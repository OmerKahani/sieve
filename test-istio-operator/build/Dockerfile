# Build the manager binary
FROM golang:1.15 as builder

WORKDIR /workspace

COPY . ./

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager github.com/banzaicloud/istio-operator/cmd/manager

# Copy the controller-manager into a thin image
FROM alpine:latest

WORKDIR /
COPY --from=builder /workspace/manager .

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories
RUN apk update
RUN apk upgrade
RUN apk add bash

# ENTRYPOINT ["/manager"]
CMD ["sleep", "infinity"]
