package handler

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/gin-gonic/gin"
)

func respError(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{
		"error": err.Error(),
	})
}

func Build(c *gin.Context) {
	globalName := c.Query("globalName")
	pkg := c.Query("pkg")
	// force rebuild
	force := c.Query("force")
	isForce := force != ""

	if globalName == "" || pkg == "" {
		respError(c, 400, errors.New("globalName and pkg are required"))
		return
	}

	projectDir := path.Join(os.TempDir(), pkg)
	outDir := path.Join(projectDir, "out")

	os.MkdirAll(outDir, os.ModePerm)

	outFile := path.Join(outDir, "input.js")

	// cache
	if _, err := os.Stat(outFile); !os.IsNotExist(err) && !isForce {
		file, err := os.Open(outFile)
		if err != nil {
			log.Println(err)
			respError(c, 500, err)
			return
		}
		defer file.Close()
		log.Printf("return cached file: %s\n", outFile)
		c.Header("content-type", "application/javascript")
		io.Copy(c.Writer, file)
		return
	}

	// Install the package
	log.Println("Installing", pkg, "in", outDir)
	cc := exec.Command("node", "--version")
	out, err := cc.Output()
	if err != nil {
		log.Println(err)
		respError(c, 500, err)
		return
	}
	log.Println(out)
	cmd := exec.Command("yarn", "add", pkg)
	cmd.Dir = projectDir
	_, err = cmd.Output()
	if err != nil {
		log.Println("failed to install pkg", err)
		respError(c, 500, err)
		return
	}

	inputFile := path.Join(projectDir, "input.js")
	input := fmt.Sprintf("module.exports = require('%s')", pkg)
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
		c.JSON(500, result.Errors)
		return
	}

	// write out files
	go func() {
		err := ioutil.WriteFile(outFile, result.OutputFiles[0].Contents, os.ModePerm)
		if err != nil {
			log.Printf("write out file error: %+v\n", err)
		}
	}()

	c.Header("content-type", "application/javascript")
	c.Writer.Write(result.OutputFiles[0].Contents)
}
