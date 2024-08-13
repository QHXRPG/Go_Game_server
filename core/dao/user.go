package dao

import (
	"context"
	"core/models/entity"
	"core/repo"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserDao struct {
	repo *repo.Manager
}

func (d UserDao) FindUserByUid(ctx context.Context, uid string) (*entity.User, error) {
	db := d.repo.Mongo.Db.Collection("user")
	singleResult := db.FindOne(ctx, bson.D{
		{"uid", uid},
	})
	user := new(entity.User)
	err := singleResult.Decode(user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
	}
	return user, err
}

func (d UserDao) Insert(ctx context.Context, user *entity.User) error {
	db := d.repo.Mongo.Db.Collection("user")
	_, err := db.InsertOne(ctx, user)
	return err
}

// UpdateUserAddressByUid 通过玩家的Uid更新玩家的地址
func (d UserDao) UpdateUserAddressByUid(todo context.Context, user *entity.User) error {
	db := d.repo.Mongo.Db.Collection("user")
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

func NewUserDao(m *repo.Manager) *UserDao {
	return &UserDao{
		repo: m,
	}
}
