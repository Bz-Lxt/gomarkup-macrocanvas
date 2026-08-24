# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates git
ENV GOPROXY=https://goproxy.cn,direct \
    GOTOOLCHAIN=auto \
    CGO_ENABLED=0 \
    GOOS=linux
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/macrocanvas ./cmd/server

FROM alpine:3.20
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata wget \
    && mkdir -p /data
ENV TZ=Asia/Shanghai
COPY --from=build /out/macrocanvas /usr/local/bin/macrocanvas
EXPOSE 8080
USER root
ENTRYPOINT ["/usr/local/bin/macrocanvas"]
