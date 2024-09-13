# Go_Game_server

基于Go语言实现的游戏服务器

![Example Image](https://github.com/QHXRPG/Go_Game_server/blob/main/go.png)

## 项目概述

搭建一个分布式的游戏服务器，并基于该服务器实现了简单的“拼三张”多人在线小游戏。

## 相关技术栈

- GoLang
- gRPC
- HTTP
- websocket
- NATS
- etcd
- 容器化部署 (Docker, Harbor)
- Kubernetes (K8s)
- Redis
- MongoDB

## 工作内容

1. 使用 `statsviz` 第三方库进行网络监测。
2. 搭建网关服务，并使用 gRPC 远程调用本地相关服务。
3. 使用 `Gin` HTTP 框架处理 GET、POST 等请求，并设置相关的 HTTP 请求与函数的映射。
4. 搭建基于 `websocket` 框架的连接服务，双向连通客户端与服务端相关服务(大厅服务、游戏服务)。
5. 构建基于 `NATS` 的连接服务与服务端通信通道，实现互相订阅与转发各个服务的消息，实现松耦合。
6. 使用信号控制各个服务的启动与停止，并通过 `etcd` 实现服务发现与负载均衡。
7. 使用 `Redis` 生成自增的用户 ID，并使用 `MongoDB` 存储玩家信息。
8. 使用 `Harbor` 搭建私有 Docker 镜像仓库，管理和分发容器镜像。
9. 部署 `StorageClass`，实现持久化存储，如用户数据、游戏状态、配置数据等数据的存储。
10. 使用 `Kubernetes` (K8s) 实现服务器集群化部署。
11. 在 K8s 集群中对 `NATS`、gRPC 和 etcd 等中间件进行部署和管理。

## 安装与使用

### 环境要求

- Go 1.16+
- Docker
- Kubernetes
- Redis
- MongoDB
- NATS
- etcd

### 安装步骤

1. 克隆项目代码：

    ```bash
    git clone https://github.com/QHXRPG/Go_Game_server.git
    cd Go_Game_server
    ```

2. 安装依赖：

    ```bash
    go mod tidy
    ```

3. 配置环境变量：

    根据项目需要配置相关的环境变量，如数据库连接字符串、NATS地址等。

4. 运行服务：

    ```bash
    go run main.go
    ```

### Docker 容器化部署

1. 构建 Docker 镜像：

    ```bash
    docker build -t go_game_server:latest .
    ```

2. 推送镜像到 Harbor 仓库：

    ```bash
    docker tag go_game_server:latest harbor.example.com/library/go_game_server:latest
    docker push harbor.example.com/library/go_game_server:latest
    ```

### Kubernetes 集群部署

1. 创建 Kubernetes 配置文件（YAML）：

    根据项目需求创建 Deployment、Service 等配置文件。

2. 部署到 Kubernetes 集群：

    ```bash
    kubectl apply -f k8s-deployment.yaml
    ```

## 功能模块

### 网络监测

使用 `statsviz` 库进行网络监测，实时查看服务器的性能指标。

### 网关服务

搭建网关服务，通过 gRPC 进行远程调用，处理客户端请求。

### HTTP 服务

使用 `Gin` 框架处理 HTTP 请求，实现 RESTful API。

### WebSocket 服务

基于 `websocket` 框架，实现客户端与服务端的实时通信。

### 消息订阅与转发

使用 `NATS` 实现服务间的消息订阅与转发，解耦各个服务。

### 服务发现与负载均衡

使用 `etcd` 实现服务发现与负载均衡，保证服务的高可用性。

### 数据存储

- **用户ID生成**:
  使用 `Redis` 生成自增的用户ID，确保用户ID的唯一性和高效生成。

- **玩家信息存储**:
  使用 `MongoDB` 存储玩家信息，包括用户数据、游戏状态和配置数据等。



