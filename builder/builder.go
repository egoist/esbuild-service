package builder

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/egoist/esbuild-service/logger"
	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/sync/singleflight"
)

var log = logger.Logger

var (
	reScoped = regexp.MustCompile("^(@[^/]+/[^/@]+)(?:/([^@]+))?(?:@([^/]+))?")
	reNormal = regexp.MustCompile("^([^/@]+)(?:/([^@]+))?(?:@([^/]+))?")
)

func logError(err error, prefix string) {
	log.Errorf("error %s %s\n", prefix, err.Error())
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

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return !os.IsNotExist(err)
}

type Builder struct {
	g singleflight.Group
}

func (b *Builder) build(pkg, globalName, projectDir, outDir, outFile string) (interface{}, error) {
	log.Printf("trigger build %s, %s", pkg, time.Now())
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

func (b *Builder) Build(pkg, globalName string, isForce bool) (interface{}, error) {
	key := fmt.Sprintf("%s-%s", pkg, globalName)

	projectDir := path.Join(os.TempDir(), key)
	outDir := path.Join(projectDir, "out")

	if !pathExists(outDir) {
		os.MkdirAll(outDir, os.ModePerm)
	}

	outFile := path.Join(outDir, "input.js")

	// cache
	if !isForce && pathExists(outFile) {
		content, err := ioutil.ReadFile(outFile)
		if err != nil {
			logError(err, "open cache file error")
			return nil, err
		}
		log.Printf("return cached file: %s\n", outFile)
		return content, nil
	}

	content, err, _ := b.g.Do(key, func() (interface{}, error) {
		return b.build(pkg, globalName, projectDir, outDir, outFile)
	})

	return content, err
}

func NewBuilder() *Builder {
	return &Builder{}
}
