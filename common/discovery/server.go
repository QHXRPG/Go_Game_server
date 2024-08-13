package discovery

//这个文件定义了一个 `Server` 结构体及其相关的方法，
//用于 ⭐在服务发现系统中处理服务实例的信息，包括构建注册键、解析存储值和键等功能。

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Server 表示一个服务实例的信息
type Server struct {
	Name    string `json:"name"`    // 服务名称
	Addr    string `json:"addr"`    // 服务地址
	Version string `json:"version"` // 服务版本
	Weight  int    `json:"weight"`  // 服务权重
	Ttl     int64  `json:"ttl"`     // 服务的生存时间（TTL）
}

// BuildRegisterKey 构建服务注册的键
// 如果服务没有版本信息，则返回格式为 "/{Name}/{Addr}"
// 如果服务有版本信息，则返回格式为 "/{Name}/{Version}/{Addr}"
func (s Server) BuildRegisterKey() string {
	if s.Version == "" {
		return fmt.Sprintf("/%s/%s", s.Name, s.Addr)
	}
	return fmt.Sprintf("/%s/%s/%s", s.Name, s.Version, s.Addr)
}

// ParseValue 解析etcd存储的值，将其转换为 Server 结构体
// val 是一个 JSON 编码的字节数组，表示服务信息
func ParseValue(val []byte) (Server, error) {
	server := Server{}
	if err := json.Unmarshal(val, &server); err != nil {
		return server, err
	}
	return server, nil
}

// ParseKey 解析etcd存储的键，将其转换为 Server 结构体
// key 是一个字符串，表示服务的注册键
// 如果键的格式不合法，则返回错误
func ParseKey(key string) (Server, error) {
	strs := strings.Split(key, "/")
	switch len(strs) {
	case 2:
		// 没有版本信息的键
		return Server{
			Name: strs[0],
			Addr: strs[1],
		}, nil
	case 3:
		// 有版本信息的键
		return Server{
			Name:    strs[0],
			Version: strs[1],
			Addr:    strs[2],
		}, nil
	default:
		// 键格式不合法
		return Server{}, errors.New("invalid key")
	}
}
