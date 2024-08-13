package database

import (
	"common/config"
	"common/logs"
	"context"
	"github.com/redis/go-redis/v9"
	"time"
)

type RedisManger struct {
	Client        *redis.Client        // 单机
	ClusterClient *redis.ClusterClient //集群
}

func NewRedis() *RedisManger {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var clusterClient *redis.ClusterClient
	var client *redis.Client
	addrs := config.Conf.Database.RedisConf.ClusterAddrs
	if len(addrs) == 0 {
		// 非集群
		client = redis.NewClient(&redis.Options{
			Addr:         config.Conf.Database.RedisConf.Addr,
			PoolSize:     config.Conf.Database.RedisConf.PoolSize,
			MinIdleConns: config.Conf.Database.RedisConf.MinIdleConns,
			Password:     config.Conf.Database.RedisConf.Password,
		})
	} else {
		clusterClient = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        config.Conf.Database.RedisConf.ClusterAddrs,
			PoolSize:     config.Conf.Database.RedisConf.PoolSize,
			MinIdleConns: config.Conf.Database.RedisConf.MinIdleConns,
			Password:     config.Conf.Database.RedisConf.Password,
		})
	}
	if clusterClient != nil {
		err := clusterClient.Ping(ctx).Err()
		if err != nil {
			logs.Error("redis cluster connect err:", err)
			return nil
		}
	}

	if client != nil {
		err := client.Ping(ctx).Err()
		if err != nil {
			logs.Error("redis client connect err:", err)
			return nil
		}
	}
	return &RedisManger{
		Client:        client,
		ClusterClient: clusterClient,
	}
}

func (r *RedisManger) Close() {
	if r.Client != nil {
		err := r.Client.Close()
		if err != nil {
			logs.Error("redis Client close err:", err)
			return
		}
	}

	if r.ClusterClient != nil {
		err := r.ClusterClient.Close()
		if err != nil {
			logs.Error("redis ClusterClient close err:", err)
			return
		}
	}
}

func (r *RedisManger) Set(ctx context.Context, key string, value string, expire int) error {
	if r.ClusterClient != nil {
		err := r.ClusterClient.Set(ctx, key, value, time.Duration(expire)).Err()
		if err != nil {
			return err
		}
	}
	if r.Client != nil {
		err := r.Client.Set(ctx, key, value, time.Duration(expire)).Err()
		if err != nil {
			return err
		}
	}
	return nil
}
