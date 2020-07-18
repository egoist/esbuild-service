package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/gin-gonic/gin"
)

var g singleflight.Group

var (
	reScoped = regexp.MustCompile("^(@[^/]+/[^/@]+)(?:/([^@]+))?(?:@([^/]+))?")
	reNormal = regexp.MustCompile("^([^/@]+)(?:/([^@]+))?(?:@([^/]+))?")
)

func respError(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{
		"error": err.Error(),
	})
}

func logError(err error, prefix string) {
	log.Printf("error %s %s\n", prefix, err.Error())
}

func parsePkgName(pkg string) [3]string {
	var matched []string

	if strings.HasPrefix(pkg, "@") {
		matched = reScoped.FindStringSubmatch(pkg)
	} else {
		matched = reNormal.FindStringSubmatch(pkg)
	}

	return [3]string{matched[1], matched[2], matched[3]}
}

func getInstallPkg(parsed [3]string) string {
	if parsed[2] == "" {
		return parsed[0]
	}
	return fmt.Sprintf("%s@%s", parsed[0], parsed[2])
}

func getRequiredPkg(parsed [3]string) string {
	if parsed[1] == "" {
		return parsed[0]
	}
	return fmt.Sprintf("%s/%s", parsed[0], parsed[1])
}

func build(pkg string, outDir string, globalName string, projectDir string, outFile string) (interface{}, error) {
	log.Printf("trigger build %s, %s", pkg, time.Now())
	time.Sleep(time.Second * 10)
	// Install the package
	log.Println("Installing", pkg, "in", outDir)
	cc := exec.Command("node", "--version")
	out, err := cc.Output()
	if err != nil {
		logError(err, "get node version")
		return nil, err
	}
	log.Printf("node version %s\n", out)

	parsedPkg := parsePkgName(pkg)
	installName := getInstallPkg(parsedPkg)
	requireName := getRequiredPkg(parsedPkg)

	log.Printf("pkg %s install %s require %s\n", pkg, installName, requireName)

	cmd := exec.Command("yarn", "add", installName)
	cmd.Dir = projectDir
	_, err = cmd.Output()
	if err != nil {
		logError(err, "failed to install pkg")
		return nil, err
	}

	inputFile := path.Join(projectDir, "input.js")
	input := fmt.Sprintf("module.exports = require('%s')", requireName)
	ioutil.WriteFile(inputFile, []byte(input), os.ModePerm)

	result := api.Build(api.BuildOptions{
		EntryPoints:       []string{inputFile},
		Outdir:            outDir,
		Bundle:            true,
		Write:             false,
		GlobalName:        globalName,
		LogLevel:          api.LogLevelInfo,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		MinifyWhitespace:  true,
	})

	if len(result.Errors) > 0 {
		log.Printf("build error: %+v\n", result.Errors)
		e, _ := json.Marshal(result.Errors)
		return nil, errors.New(string(e))
	}

	// write out files
	go func() {
		err := ioutil.WriteFile(outFile, result.OutputFiles[0].Contents, os.ModePerm)
		if err != nil {
			log.Printf("write out file error: %+v\n", err)
		}
	}()
	return result.OutputFiles[0].Contents, nil
}

func Build(c *gin.Context) {
	globalName := c.Query("globalName")
	pkg := c.Param("pkg")
	// force rebuild
	force := c.Query("force")
	isForce := force != ""

	if globalName == "" {
		respError(c, 400, errors.New("globalName is required"))
		return
	}

	pkg = strings.TrimLeft(pkg, "/")

	projectDir := path.Join(os.TempDir(), pkg)
	outDir := path.Join(projectDir, "out")

	os.MkdirAll(outDir, os.ModePerm)

	outFile := path.Join(outDir, "input.js")

	// cache
	if _, err := os.Stat(outFile); !os.IsNotExist(err) && !isForce {
		file, err := os.Open(outFile)
		if err != nil {
			logError(err, "open cache file error")
			respError(c, 500, err)
			return
		}
		defer file.Close()
		log.Printf("return cached file: %s\n", outFile)
		c.Header("content-type", "application/javascript")
		io.Copy(c.Writer, file)
		return
	}

	// 
	content, err, _ := g.Do(pkg, func() (interface{}, error) {
		return build(pkg, outDir, globalName, projectDir, outFile)
	})

	if err != nil {
		respError(c, 500, err)
		return
	}

	c.Header("content-type", "application/javascript")
	c.Writer.Write(content.([]byte))
}
