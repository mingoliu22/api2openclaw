# 构建阶段
FROM golang:1.23-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git make

# 设置工作目录
WORKDIR /build

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api2openclaw ./cmd/api2openclaw

# 运行阶段
FROM alpine:latest

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 创建非 root 用户
RUN addgroup -g 1000 -S api2openclaw && \
    adduser -u 1000 -S api2openclaw -G api2openclaw

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/api2openclaw .

# 从构建阶段复制前端文件
COPY --from=builder /build/web/dist /app/web/dist

# 创建配置目录
RUN mkdir -p /app/configs /app/logs

# 复制默认配置
COPY configs/config.yaml /app/configs/

# 更改所有权
RUN chown -R api2openclaw:api2openclaw /app

# 切换到非 root 用户
USER api2openclaw

# 暴露端口
EXPOSE 8080 9090

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 启动应用
ENTRYPOINT ["/app/api2openclaw"]
CMD ["--config", "/app/configs/config.yaml"]
