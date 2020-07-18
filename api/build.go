package api

import (
	"errors"
	"strings"

	"github.com/egoist/esbuild-service/builder"
	"github.com/gin-gonic/gin"
)

func respError(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{
		"error": err.Error(),
	})
}

func CreateBuildHandler(builder *builder.Builder) gin.HandlerFunc {
	return func(c *gin.Context) {
		globalName := strings.TrimSpace(c.Query("globalName"))
		pkg := c.Param("pkg")
		pkg = strings.TrimLeft(pkg, "/")
		// force rebuild
		force := strings.TrimSpace(c.Query("force"))
		isForce := force != ""

		if globalName == "" {
			respError(c, 400, errors.New("globalName is required"))
			return
		}

		content, err := builder.Build(pkg, globalName, isForce)

		if err != nil {
			respError(c, 500, err)
			return
		}

		c.Header("content-type", "application/javascript")
		c.Writer.Write(content.([]byte))
	}
}
