# 构建阶段
FROM golang:1.23-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git make

WORKDIR /build

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o api2openclaw cmd/api2openclaw/main.go

# 运行阶段
FROM alpine:latest

# 安装 ca 证书（用于 HTTPS 请求）
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/api2openclaw .

# 复制配置文件
COPY configs/ ./configs/

# 创建日志目录
RUN mkdir -p /var/log/api2openclaw

# 暴露端口
EXPOSE 8080 9090

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 运行应用
ENTRYPOINT ["./api2openclaw"]
CMD ["-config", "configs/config.yaml"]
