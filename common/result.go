package common

import (
	"common/biz"
	"framework/msError"
	"github.com/gin-gonic/gin" // 导入 Gin 框架
	"net/http"                 // 导入 HTTP 包
)

// Result 结构体用于表示 API 响应的结构
type Result struct {
	Code int `json:"code"` // 状态码
	Msg  any `json:"msg"`  // 消息或数据，可以是任意类型
}

// F 函数用于返回错误响应
// 参数 ctx 是 Gin 的上下文，err 是自定义的错误类型
func F(ctx *gin.Context, err *msError.Error) {
	// 使用 Gin 框架的 JSON 方法返回 HTTP 状态码 200 和错误信息
	ctx.JSON(http.StatusOK, Result{
		Code: err.Code,
		Msg:  err.Err.Error(),
	})
}

func Fail(err *msError.Error) Result {
	return Result{
		Code: err.Code,
	}
}

func S(data any) Result {
	return Result{
		Code: biz.OK,
		Msg:  data,
	}
}

// Success 函数用于返回成功响应
func Success(ctx *gin.Context, data any) {
	ctx.JSON(http.StatusOK, Result{
		Code: biz.OK,
		Msg:  data,
	})
}
