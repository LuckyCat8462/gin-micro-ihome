package utils

import (
	"context"
	"net/http"
	"time"

	"gin-micro-ihome/web/conf"

	"github.com/gin-gonic/gin"
)

// RPCTimeout 网关调用微服务的默认超时
func RPCTimeout() time.Duration {
	sec := conf.RPCTimeoutSeconds
	if sec <= 0 {
		sec = 5
	}
	return time.Duration(sec) * time.Second
}

// RPCCtx 带超时的 RPC context（超时后自动取消；适合请求级一次性调用）
func RPCCtx() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), RPCTimeout())
	// 超时到达后释放 timer；调用方无需 defer
	go func() {
		<-ctx.Done()
		cancel()
	}()
	return ctx
}

// AbortRPC 下游 RPC 传输失败时的统一 JSON 响应
func AbortRPC(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"errno":  RECODE_SERVERERR,
		"errmsg": RecodeText(RECODE_SERVERERR),
	})
}

// AbortParam 参数错误统一响应
func AbortParam(ctx *gin.Context, errmsg string) {
	if errmsg == "" {
		errmsg = RecodeText(RECODE_PARAMERR)
	}
	ctx.JSON(http.StatusOK, gin.H{
		"errno":  RECODE_PARAMERR,
		"errmsg": errmsg,
	})
}
