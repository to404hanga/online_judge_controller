package ioc

import (
	"log"

	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitEtcdClient() *clientv3.Client {
	var cfg clientv3.Config
	err := viper.UnmarshalKey("etcd", &cfg)
	if err != nil {
		log.Panicf("unmarshal etcd config failed: %v", err)
	}
	client, err := clientv3.New(cfg)
	if err != nil {
		log.Panicf("create etcd client failed: %v", err)
	}
	return client
}
