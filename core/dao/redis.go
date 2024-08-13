package dao

// 保证用户id唯一

import (
	"context"
	"core/repo"
	"fmt"
)

const Predix = "QHXRPG"
const AccountIdRedisKey = "AccountId"
const AccountIdBegin = 10000

type RedisDao struct {
	repo *repo.Manager
}

func (d RedisDao) NextAccountId() (string, error) {
	// 自增,给一个前缀
	return d.incr(Predix + ":" + AccountIdRedisKey)
}

func (d RedisDao) incr(key string) (string, error) {
	// 判断此key是否存在，不存在set， 存在就自增
	todo := context.TODO()
	var exist int64
	var err error
	// 0代表不存在
	if d.repo.Redis.Client != nil {
		exist, err = d.repo.Redis.Client.Exists(todo, key).Result()
	} else {
		exist, err = d.repo.Redis.ClusterClient.Exists(todo, key).Result()
	}

	if exist == 0 {
		// 不存在，直接set
		if d.repo.Redis.Client != nil {
			err = d.repo.Redis.Client.Set(todo, key, AccountIdBegin, 0).Err()
		} else {
			err = d.repo.Redis.ClusterClient.Set(todo, key, AccountIdBegin, 0).Err()
		}
		if err != nil {
			return "", err
		}
	}
	var id int64
	if d.repo.Redis.Client != nil {
		id, err = d.repo.Redis.Client.Incr(todo, key).Result()

	} else {
		id, err = d.repo.Redis.ClusterClient.Incr(todo, key).Result()
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", id), nil
}

func NewRedisDao(m *repo.Manager) *RedisDao {
	return &RedisDao{
		repo: m}
}
