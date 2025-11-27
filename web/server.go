package web

import (
	"net"

	"github.com/gin-gonic/gin"
)

type GinServer struct {
	Engine   *gin.Engine
	Listener net.Listener
}

func (s *GinServer) Start() error {
	return s.Engine.RunListener(s.Listener)
}
