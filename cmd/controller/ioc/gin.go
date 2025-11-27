package ioc

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/to404hanga/online_judge_controller/config"
	"github.com/to404hanga/online_judge_controller/pkg/gintool"
	"github.com/to404hanga/online_judge_controller/web"
	"github.com/to404hanga/online_judge_controller/web/jwt"
	"github.com/to404hanga/online_judge_controller/web/middleware"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gorm.io/gorm"
)

func InitGinServer(etcdCli *clientv3.Client, l loggerv2.Logger, jwtHandler jwt.Handler, db *gorm.DB, competitionHandler *web.CompetitionHandler, problemHandler *web.ProblemHandler, submissionHandler *web.SubmissionHandler, healthHandler *web.HealthHandler, userHandler *web.UserHandler) *web.GinServer {
	var cfg config.GinConfig
	err := viper.UnmarshalKey(cfg.Key(), &cfg)
	if err != nil {
		log.Panicf("unmarshal gin config failed, err: %v", err)
	}

	// 优先使用环境变量中设置的服务端口
	addr := cfg.Addr
	if addrEnv := os.Getenv("SERVER_ADDR"); addrEnv != "" {
		addr = addrEnv
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Panicf("listen port failed, err: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	var ip string
	// 判断是否为 Docker 环境
	if containerEnv := os.Getenv("SERVER_CONTAINER"); containerEnv != "" {
		ip = cfg.ServiceName
	} else {
		ip, err = getLocalIP()
		if err != nil {
			log.Panicf("get local ip failed, err: %v", err)
		}
	}

	addr = fmt.Sprintf("%s:%d", ip, port)

	key := fmt.Sprintf("/services/%s/%s", cfg.ServiceName, addr)
	valueStruct := EtcdServiceConfig{
		Addr:   addr,
		Weight: cfg.Weight,
	}
	valueBytes, err := json.Marshal(valueStruct)
	if err != nil {
		log.Panicf("marshal etcd service config failed, err: %v", err)
	}
	value := string(valueBytes)

	// 创建 Lease
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := etcdCli.Grant(ctx, 5)
	if err != nil {
		log.Panicf("创建 lease 失败: %v", err)
	}
	leaseID := resp.ID

	// 写入 Etcd (绑定 Lease)
	_, err = etcdCli.Put(ctx, key, value, clientv3.WithLease(leaseID))
	if err != nil {
		log.Panicf("注册 key 失败: %v", err)
	}

	// 设置 KeepAlive
	// KeepAlive 需要一个长生命周期的 context，不能用上面的 timeout context
	keepAliveCtx, _ := context.WithCancel(context.Background())
	keepAliveChan, err := etcdCli.KeepAlive(keepAliveCtx, leaseID)
	if err != nil {
		log.Panicf("设置 keepalive 失败: %v", err)
	}

	// 启动协程处理 KeepAlive 响应（可选，用于监控连接状态）
	go func() {
		for range keepAliveChan {
			// 正常收到心跳响应，这里可以记录 debug 日志
			// log.Printf("收到续租响应: %v", resp.TTL)
		}
	}()

	corsBuilder := middleware.NewCORSMiddlewareBuilder(
		cfg.AllowOrigins,
		cfg.AllowMethods,
		cfg.AllowHeaders,
		cfg.ExposeHeaders,
		cfg.AllowCredentials,
		time.Duration(cfg.MaxAge)*time.Second)
	jwtBuilder := middleware.NewJWTMiddlewareBuilder(jwtHandler, db, l, cfg.CheckCompetitionPath)

	engine := gin.Default()
	engine.Use(
		corsBuilder.Build(),
		jwtBuilder.CheckCompetition(),
		gintool.ContextMiddleware(),
	)

	competitionHandler.Register(engine)
	problemHandler.Register(engine)
	submissionHandler.Register(engine)
	healthHandler.Register(engine)
	userHandler.Register(engine)

	return &web.GinServer{
		Engine:   engine,
		Listener: listener,
	}
}

// getLocalIP 获取本机非回环 IPv4 地址
func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		// 检查是否为 IP 地址且不是 loopback
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("未找到有效 IP")
}

type EtcdServiceConfig struct {
	Addr   string `json:"addr"`
	Weight int    `json:"weight"`
}
