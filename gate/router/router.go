package router

import (
	"common/config"
	"gate/api"
	"gate/auth"
	"gate/rpc"
	"github.com/gin-gonic/gin"
)

// router 用于定义和注册 HTTP 路由，配置日志级别、跨域中间件，并初始化 gRPC 客户端。
// Gin 是一个 HTTP Web 框架, 处理 GET、POST、PUT、DELETE 等 HTTP 方法。

// RegisterRouter 注册路由
func RegisterRouter() *gin.Engine {
	// 根据配置设置 Gin 框架的运行模式
	if config.Conf.Log.Level == "DEBUG" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化 gRPC 客户端，⭐gate 是作为 gRPC 的客户端，去调用 user 的 gRPC 服务
	rpc.Init()

	// 创建一个默认的 Gin 引擎实例
	r := gin.Default()

	// 使用跨域中间件解决跨域问题
	r.Use(auth.Cors())

	// 创建一个新的用户处理器实例
	userHandler := api.NewUserHandler()

	// 在 Gin 框架中注册一个 POST 请求的路由
	// 将路径为 /register 的 POST 请求映射到 userHandler.Register 处理函数
	r.POST("/register", userHandler.Register)

	return r
}
