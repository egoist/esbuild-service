package api

import (
	"strings"

	"github.com/egoist/esbuild-service/builder"
	"github.com/gin-gonic/gin"
)

func respError(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{
		"error": err.Error(),
	})
}

func CreateBuildHandler(b *builder.Builder) gin.HandlerFunc {
	return func(c *gin.Context) {
		GlobalName := strings.TrimSpace(c.Query("globalName"))
		Pkg := c.Param("pkg")
		Pkg = strings.TrimLeft(Pkg, "/")
		// force rebuild
		force := strings.TrimSpace(c.Query("force"))
		isForce := force != ""
		Format := c.Query("format")

		content, err := b.Build(&builder.BuildOptions{
			Pkg:        Pkg,
			GlobalName: GlobalName,
			Format:     Format,
		}, isForce)

		if err != nil {
			respError(c, 500, err)
			return
		}

		c.Header("content-type", "application/javascript")
		c.Writer.Write(content.([]byte))
	}
}
