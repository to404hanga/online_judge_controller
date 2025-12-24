package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const defaultConfigPath = "./config/config.yaml"

func main() {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Panicf("load location failed: %v", err)
	}
	time.Local = loc

	cfile := pflag.String("config", defaultConfigPath, "config file path")
	pflag.Parse()

	viper.SetConfigFile(*cfile)
	if err := viper.ReadInConfig(); err != nil {
		log.Panicf("read config file failed: %v", err)
	}

	gin.DisableBindValidation()

	app := BuildDependency()
	log.Println("gin server start")
	if err := app.Start(); err != nil {
		log.Panicf("gin server failed: %v", err)
	}
}
