package api

import (
	"fmt"
	"strings"

	"github.com/egoist/esbuild-service/builder"
	"github.com/egoist/esbuild-service/logger"
	"github.com/egoist/esbuild-service/util"
	"github.com/gin-gonic/gin"
)

var log = logger.Logger

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
		Platform := c.Query("platform")
		Minify := c.Query("minify")
		var enableMinify = true
		if Minify != "" {
			enableMinify = util.StrToBool(Minify)
		}

		parsedPkg := util.ParsePkgName(Pkg)
		version := parsedPkg[2]
		if version == "" {
			version = "latest"
		}
		pkgVersion, err := builder.GetPkgMatchVersion(parsedPkg[0], version)
		if err != nil {
			log.Errorf("failed to get pkg match version: %s, error: %s", Pkg, err.Error())
			respError(c, 400, err)
			return
		}

		// redirect to exact matched version
		if pkgVersion != version {
			matchedPkg := fmt.Sprintf("%s%s@%s", parsedPkg[0], parsedPkg[1], pkgVersion)
			url := strings.Replace(c.Request.URL.String(), Pkg, matchedPkg, 1)
			log.Infof("redirect to %s", url)
			c.Redirect(302, url)
			return
		}

		content, err := b.Build(&builder.BuildOptions{
			Pkg:        Pkg,
			GlobalName: GlobalName,
			Format:     Format,
			Platform:   Platform,
			IsMinify:   enableMinify,
			ParsedPkg:  parsedPkg,
			PkgVersion: pkgVersion,
		}, isForce)

		if err != nil {
			respError(c, 500, err)
			return
		}

		c.Header("content-type", "application/javascript")
		c.Writer.Write(content.([]byte))
	}
}
