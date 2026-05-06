ARG BUILDER_IMAGE=docker.1ms.run/golang:1.25-alpine
ARG RUNTIME_IMAGE=docker.1ms.run/alpine:3.19

FROM ${BUILDER_IMAGE} AS builder
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories \
    && apk update \
    && apk add --no-cache git ca-certificates
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server/

FROM ${RUNTIME_IMAGE} AS runtime
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories \
    && apk update \
    && apk add --no-cache \
    chromium \
    font-noto-cjk \
    ca-certificates \
    curl \
    git \
    tzdata
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /server /app/server
EXPOSE 8080
CMD ["/app/server"]
