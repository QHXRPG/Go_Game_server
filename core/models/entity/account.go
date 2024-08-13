package entity

// 账号 用户密码登录， 生成一个账号，在这个账号下有多个角色
// 直接使用账号登录，账号对应一个用户的角色 1对1

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Account struct {
	Id           primitive.ObjectID `bson:"_id,omitempty"`
	Uid          string             `bson:"uid"`
	Account      string             `bson:"account" `
	Password     string             `bson:"password"`
	PhoneAccount string             `bson:"phoneAccount"`
	WxAccount    string             `bson:"wxAccount"`
	CreateTime   time.Time          `bson:"createTime"`
}
