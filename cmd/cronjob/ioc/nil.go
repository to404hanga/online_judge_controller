package ioc

import (
	"github.com/redis/go-redis/v9"
	"github.com/to404hanga/online_judge_controller/event"
)

func InitNilRedis() redis.Cmdable {
	return nil
}

func InitNilKafka() event.Producer {
	return nil
}
