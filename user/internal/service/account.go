package service

import (
	"common/biz"
	"common/logs"
	"context"
	"core/dao"
	"core/models/entity"
	"core/models/requests"
	"core/repo"
	"framework/msError"
	"time"
	"user/pb"
)

// 实现了用户账户服务的业务逻辑，包括用户注册功能。它使用了 gRPC 作为通信协议，
// 定义了 AccountService 结构体，并通过 NewAccountService 函数初始化服务。
// Register 方法实现了具体的注册逻辑，包括微信注册的处理

// AccountService 账号服务结构体
type AccountService struct {
	accountDao                        *dao.AccountDao
	redisDao                          *dao.RedisDao
	pb.UnimplementedUserServiceServer // 作为 gRPC 服务接口的默认实现
}

// NewAccountService 创建新的账号服务实例
func NewAccountService(manger *repo.Manager) *AccountService {
	return &AccountService{
		accountDao: dao.NewAccountDao(manger),
		redisDao:   dao.NewRedisDao(manger),
	}
}

// Register 用户注册方法
func (a *AccountService) Register(ctx context.Context, req *pb.RegisterParams) (*pb.RegisterResponse, error) {
	// 处理微信注册逻辑
	if req.LoginPlatform == requests.WeiXin {
		ac, err := a.wxRegister(req)
		if err != nil {
			return &pb.RegisterResponse{}, msError.GrpcError(err)
		}
		return &pb.RegisterResponse{
			Uid: ac.Uid,
		}, nil
	}

	logs.Info("register service called ...")
	return &pb.RegisterResponse{
		Uid: "10000",
	}, nil
}

// 微信账号注册逻辑
func (a *AccountService) wxRegister(req *pb.RegisterParams) (*entity.Account, *msError.Error) {
	// 1. 封装一个account结构，将其存入数据库
	ac := &entity.Account{
		WxAccount:  req.Account,
		CreateTime: time.Now(),
	}
	// 2. 生成用户唯一ID，使用Redis自增
	uid, err := a.redisDao.NextAccountId()
	if err != nil {
		return ac, biz.SqlError
	}
	ac.Uid = uid
	err = a.accountDao.SaveAccount(context.TODO(), ac)
	if err != nil {
		return ac, biz.SqlError
	}
	return ac, nil
}
