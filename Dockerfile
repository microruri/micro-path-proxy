# 第一阶段：编译 Go 代码
FROM golang:1.20-alpine AS builder

# 安装根证书，确保 Go 程序通过 HTTPS 请求外界时不会报证书错误
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY go.mod ./
COPY *.go ./

# 静态编译，关闭 CGO，剥离调试信息以大幅缩减体积
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o go-path-proxy main.go

# 第二阶段：构建极其轻量、安全的纯净运行环境
FROM alpine:latest

# 继承上一个阶段的最新根证书
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app
COPY --from=builder /app/go-path-proxy .

# 为了安全起见，建立非 root 用户来运行程序，防止容器逃逸
RUN adduser -D proxyuser
USER proxyuser

# 声明向外暴露的系统端口 (仅作展示，实际依赖于你的命令行参数)
EXPOSE 5000

# 默认将参数抛给被执行的二进制文件
ENTRYPOINT ["./go-path-proxy"]
