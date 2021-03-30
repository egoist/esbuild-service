package api

import (
	"fmt"
	"net/http"
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
		Pkg := c.Param("pkg")

		if Pkg == "/" {
			c.String(http.StatusOK, `
	USAGE:	https://esbuild.egoist.sh/{pkg}

	REPO:   https://github.com/egoist/esbuild-service
		`)
			return
		}

		GlobalName := strings.TrimSpace(c.Query("globalName"))
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

		parsedPkg, err := util.ParsePkgName(Pkg)
		if err != nil {
			respError(c, 400, err)
			return
		}

		version := parsedPkg.Version
		if version == "" {
			version = "latest"
		}

		pkgVersion, err := builder.GetPkgMatchVersion(parsedPkg.Name, version)
		if err != nil {
			log.Errorf("failed to get pkg match version: %s, error: %s", Pkg, err.Error())
			respError(c, 400, err)
			return
		}

		// redirect to exact matched version
		if pkgVersion != version {
			matchedPkg := fmt.Sprintf("%s@%s%s", parsedPkg.Name, pkgVersion, parsedPkg.Filename)

			url := "/" + matchedPkg
			if c.Request.URL.RawQuery != "" {
				url += "?" + c.Request.URL.RawQuery
			}
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

func Handler(w http.ResponseWriter, r *http.Request) {
	g := gin.Default()

	b := builder.NewBuilder()

	g.GET("/*pkg", CreateBuildHandler(b))

	g.ServeHTTP(w, r)
}
