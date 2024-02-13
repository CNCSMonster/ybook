//go:build release

package log

import (
	"io"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard
}

func Printf(format string, v ...interface{}) {
	//do nothing
}
func Println(v ...interface{}) {
	//do nothing
}
func Fatal(v ...interface{}) {
	//do nothing
}
