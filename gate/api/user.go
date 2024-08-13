package api

import (
	"common"
	"common/biz"
	"common/config"
	"common/jwts"
	"common/logs"
	"context"
	"framework/msError"
	"gate/rpc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"time"
	"user/pb"
)

// 实现了用户注册功能，通过RPC服务注册用户，并生成JWT令牌返回给客户端。

// UserHandler 处理用户相关请求的处理器
type UserHandler struct {
}

// NewUserHandler 创建一个新的UserHandler实例
func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// Register 处理用户注册请求
func (u *UserHandler) Register(ctx *gin.Context) {
	// 接收参数，将JSON请求体绑定到pb.RegisterParams结构体 req
	var req pb.RegisterParams
	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		common.F(ctx, biz.RequestDataError)
		return
	}

	// 调用RPC服务进行注册，将注册请求发送到用户服务
	response, err := rpc.UserClient.Register(context.TODO(), &req)
	if err != nil {
		common.F(ctx, msError.ToError(err))
		return
	}
	uid := response.Uid
	if len(uid) == 0 {
		common.F(ctx, biz.SqlError)
		return
	}
	// 记录注册成功的用户ID
	logs.Info("register uid:%v", uid)

	// 使用JWT生成token，设置token的有效期为7天
	claims := jwts.CustomClaims{
		Uid: uid, // 根据 uid 生成 token
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 7)),
		},
	}
	token, err := jwts.GenToken(&claims, config.Conf.Jwt.Secret)
	if err != nil {
		logs.Error("Register jwt gen token err:%v", err)
		common.F(ctx, biz.Fail)
		return
	}

	// 准备返回结果，包含token和服务器信息
	result := map[string]any{
		"token": token,
		"serverInfo": map[string]any{
			"host": config.Conf.Services["connector"].ClientHost,
			"port": config.Conf.Services["connector"].ClientPort,
		},
	}
	common.Success(ctx, result)
}
