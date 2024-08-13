package dao

import (
	"context"
	"core/models/entity"
	"core/repo"
)

// 实现了 AccountDao 数据访问对象（DAO），用于与数据库交互，
// 特别是用于保存用户账号信息。通过 NewAccountDao 函数初始化 DAO 实例，
// 并提供了 SaveAccount 方法用于将账号信息保存到 MongoDB 数据库中。

// AccountDao 账号数据访问对象
type AccountDao struct {
	repo *repo.Manager
}

// SaveAccount 保存账号信息到数据库
func (d AccountDao) SaveAccount(ctx context.Context, ac *entity.Account) error {
	// 获取账号表
	table := d.repo.Mongo.Db.Collection("account")
	// 插入账号数据
	_, err := table.InsertOne(ctx, ac)
	if err != nil {
		return err
	}
	return nil
}

// NewAccountDao 创建新的账号DAO实例
func NewAccountDao(m *repo.Manager) *AccountDao {
	return &AccountDao{
		repo: m,
	}
}
