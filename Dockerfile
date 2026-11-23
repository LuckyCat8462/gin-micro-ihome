# gin-micro-ihome 基础镜像
# 用法示例:
#   docker build -t ihome-web --build-arg TARGET=web .
#   docker build -t ihome-house --build-arg TARGET=house .
#   docker run --rm -p 8080:8080 ihome-web

# ---------- 构建阶段 ----------
FROM golang:1.24.4-alpine AS builder

WORKDIR /src

# 部分依赖需要 git / gcc
RUN apk add --no-cache git gcc musl-dev

# Go 模块代理（proxy.golang.org 在国内网络不可达，默认 goproxy.cn；可用 --build-arg 覆盖）
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}

# 可选目标: web | getCaptcha | register | user | house | order
ARG TARGET=web

# 先复制依赖清单，利用缓存
COPY go.mod go.sum ./
COPY service/getCaptcha/go.mod service/getCaptcha/go.sum ./service/getCaptcha/
COPY service/register/go.mod service/register/go.sum ./service/register/
COPY service/user/go.mod service/user/go.sum ./service/user/
COPY service/house/go.mod service/house/go.sum ./service/house/
COPY service/order/go.mod service/order/go.sum ./service/order/

# 按目标下载依赖
RUN if [ "$TARGET" = "web" ]; then \
      go mod download; \
    else \
      cd "service/${TARGET}" && go mod download; \
    fi

# 再复制全部源码并编译
COPY . .

RUN set -eux; \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64; \
    if [ "$TARGET" = "web" ]; then \
      go build -ldflags="-s -w" -o /out/app ./web; \
    else \
      cd "service/${TARGET}" && go build -ldflags="-s -w" -o /out/app .; \
    fi

# ---------- 运行阶段 ----------
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

# Web 静态资源（微服务镜像也会带上，体积影响可忽略）
COPY --from=builder /src/web/view ./web/view
COPY --from=builder /out/app ./app

# Web: 8080；微服务默认端口见各服务 conf（12311-12315）
EXPOSE 8080 12311 12312 12313 12314 12315

ENTRYPOINT ["./app"]
