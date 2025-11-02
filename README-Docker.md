# Docker 部署指南

本项目提供了完整的Docker化部署方案，包括应用程序本身以及所需的依赖服务（MySQL、Redis、MinIO）。

## 文件说明

- `Dockerfile`: 应用程序的Docker镜像构建文件
- `.env`: 环境变量配置文件（需要根据实际情况修改）
- `.env.example`: 环境变量配置模板
- `docker-compose.yml`: 完整的服务编排文件

## 快速开始

### 1. 准备环境变量

复制环境变量模板并修改配置：

```bash
cp .env.example .env
```

编辑 `.env` 文件，设置实际的MinIO访问密钥：

```env
MINIO_ACCESS_KEY_ID=your_actual_access_key
MINIO_SECRET_ACCESS_KEY=your_actual_secret_key
```

### 2. 使用Docker Compose启动所有服务

```bash
# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看应用日志
docker-compose logs -f online-judge-controller
```

### 3. 仅构建和运行应用程序

如果你已经有外部的MySQL、Redis和MinIO服务，可以只运行应用程序：

```bash
# 构建镜像
docker build -t online-judge-controller .

# 运行容器
docker run -d \
  --name online-judge-controller \
  --env-file .env \
  -p 8080:8080 \
  -v $(pwd)/log:/root/log \
  online-judge-controller
```

## 服务访问

启动成功后，可以通过以下地址访问各个服务：

- **应用程序**: http://localhost:8080
- **MinIO控制台**: http://localhost:9001 (用户名/密码: 见.env文件配置)
- **MySQL**: localhost:3306
- **Redis**: localhost:6379

## 环境变量说明

应用程序会读取以下环境变量：

- `MINIO_ACCESS_KEY_ID`: MinIO访问密钥ID
- `MINIO_SECRET_ACCESS_KEY`: MinIO访问密钥

这些环境变量对应代码中的：
- `pkg/minio/env.go` 中定义的 `EnvMinIOAccessKeyID` 和 `EnvMinIOSecretAccessKey`

## 配置文件

应用程序使用 `cmd/controller/config/config.yaml` 作为主配置文件，该文件在Docker镜像构建时会被复制到容器中。

如果需要修改配置，可以：

1. 修改 `config.yaml` 文件后重新构建镜像
2. 或者通过volume挂载外部配置文件：

```bash
docker run -d \
  --name online-judge-controller \
  --env-file .env \
  -p 8080:8080 \
  -v $(pwd)/config:/root/config \
  -v $(pwd)/log:/root/log \
  online-judge-controller
```

## 停止服务

```bash
# 停止所有服务
docker-compose down

# 停止并删除数据卷（注意：这会删除数据库数据）
docker-compose down -v
```

## 故障排除

### 查看日志

```bash
# 查看所有服务日志
docker-compose logs

# 查看特定服务日志
docker-compose logs online-judge-controller
docker-compose logs mysql
docker-compose logs redis
docker-compose logs minio
```

### 重新构建镜像

如果修改了代码，需要重新构建镜像：

```bash
docker-compose build online-judge-controller
docker-compose up -d online-judge-controller
```