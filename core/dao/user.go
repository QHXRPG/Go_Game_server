package dao

import (
	"context"
	"core/models/entity"
	"core/repo"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Data Access Object (DAO) 数据访问对象
// DAO层用于 ⭐抽象和封装对数据库的访问和操作。
// 它提供了一系列方法用于执行数据库的CRUD操作，使业务逻辑层不需要直接处理数据库操作，从而提高代码的可维护性和可测试性。

// UserDao 结构体用于处理与用户相关的数据库操作
type UserDao struct {
	repo *repo.Manager
}

// FindUserByUid 根据用户的UID查找用户信息
func (d UserDao) FindUserByUid(ctx context.Context, uid string) (*entity.User, error) {
	// 获取user集合
	db := d.repo.Mongo.Db.Collection("user")
	// 查找单个文档
	singleResult := db.FindOne(ctx, bson.D{
		{"uid", uid},
	})
	// 创建一个User实例用于解析结果
	user := new(entity.User)
	// 解析查询结果
	err := singleResult.Decode(user)
	if err != nil {
		// 如果没有找到文档，返回nil
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
	}
	return user, err
}

// Insert 插入新的用户信息到数据库
func (d UserDao) Insert(ctx context.Context, user *entity.User) error {
	// 获取user集合
	db := d.repo.Mongo.Db.Collection("user")
	// 插入一个文档
	_, err := db.InsertOne(ctx, user)
	return err
}

// UpdateUserAddressByUid 通过用户的UID更新用户的地址信息
func (d UserDao) UpdateUserAddressByUid(todo context.Context, user *entity.User) error {
	// 获取user集合
	db := d.repo.Mongo.Db.Collection("user")
	// 更新一个文档
	_, err := db.UpdateOne(todo, bson.M{
		"uid": user.Uid,
	}, bson.M{
		"$set": bson.M{
			"address":  user.Address,
			"Location": user.Location,
		},
	})
	if err != nil {
		return err
	}
	return nil
}

// NewUserDao 创建并返回一个新的 UserDao 实例
func NewUserDao(m *repo.Manager) *UserDao {
	return &UserDao{
		repo: m,
	}
}
