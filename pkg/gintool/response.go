package gintool

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/to404hanga/online_judge_controller/constants"
)

type Response struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"request_id"`
}

func GinResponse(c *gin.Context, resp *Response) {
	resp.RequestID = c.GetHeader(constants.HeaderRequestIDKey)
	c.JSON(http.StatusOK, resp)
}
