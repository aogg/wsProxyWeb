# 构建阶段
FROM golang:1.24-alpine AS builder

WORKDIR /build

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译二进制文件
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server main.go

# 运行阶段
FROM alpine:latest

WORKDIR /app

# 安装 ca-certificates 用于 HTTPS 请求
RUN apk --no-cache add ca-certificates

# 从构建阶段复制二进制文件
COPY --from=builder /build/server .

# 复制配置文件
COPY src/configs ./src/configs

# 创建运行时目录
RUN mkdir -p runtime/server/logs runtime/server/data

# 暴露端口
EXPOSE 8080

# 运行服务
CMD ["./server"]
