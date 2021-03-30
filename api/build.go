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
			c.Header("content-type", "text/html")
			c.String(http.StatusOK, `

	<meta charset="utf-8">
	<style>
	body {
		font-family:system-ui,-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif,Apple Color Emoji,Segoe UI Emoji;
		padding: 20px;
	}

	.link {
		color: #000;
		text-decoration: none;
	}

	.link:hover {
		color: blue;
	}

	.label {
		color: #555;
		font-weight: bold;
		min-width: 140px;
		display: inline-block;
	}

	p {
		margin: 20px 0;
	}
	</style>

	<title>esbuild service</title>

	<h1>esbuild</h1>

	<p style="margin-bottom:30px">Bundle npm packages in ESM / CJS format on the fly.</p>

	<p>
	<span class="label">USAGE:</span>https://esbuild.vercel.app/{pkg}
	</p>

	<p>

	<p>
	<span class="label">EXAMPLE:</span><a class="link" href="/lodash@4.17.21/debounce?minify=false">lodash/debounce (no minification)</a>
	&nbsp;&nbsp;<a class="link" href="/chalk">chalk (minified)</a>
	</p>

	<p>
	<span class="label">MORE USAGES:</span><a class="link" href="https://github.com/egoist/esbuild-service">https://github.com/egoist/esbuild-service</a>
	</p>

	<span class="label">SUPPORT:</span><a class="link" href="https://github.com/sponsors/egoist">💖 Become a sponsor on GitHub</a>
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
