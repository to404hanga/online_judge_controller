package web

import "github.com/gin-gonic/gin"

type GinServer struct {
	Engine *gin.Engine
	Addr   string
}

func (s *GinServer) Start() error {
	return s.Engine.Run(s.Addr)
}
