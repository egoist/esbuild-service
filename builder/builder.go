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

// Builder is a struct
type Builder struct {
	g singleflight.Group
}

type BuildOptions struct {
	Pkg        string
	GlobalName string
	Format     string
}

type projectOptions struct {
	OutDir     string
	OutFile    string
	ProjectDir string
}

// build without cache
func (b *Builder) buildFresh(options *BuildOptions, project *projectOptions) (interface{}, error) {
	log.Printf("trigger build %s, %s", options.Pkg, time.Now())
	// Install the package
	log.Println("Installing", options.Pkg, "in", project.OutDir)
	cc := exec.Command("node", "--version")
	out, err := cc.Output()
	if err != nil {
		logError(err, "get node version")
		return nil, err
	}
	log.Printf("node version %s\n", out)

	parsedPkg := parsePkgName(options.Pkg)
	installName := getInstallPkg(parsedPkg)
	requireName := getRequiredPkg(parsedPkg)

	log.Printf("pkg %s install %s require %s\n", options.Pkg, installName, requireName)

	log.Printf("install in %s", project.ProjectDir)
	cmd := exec.Command("yarn", "add", installName)
	cmd.Dir = project.ProjectDir
	_, err = cmd.Output()
	if err != nil {
		logError(err, "failed to install pkg")
		return nil, err
	}

	inputFile := path.Join(project.ProjectDir, "input.js")
	input := fmt.Sprintf("module.exports = require('%s')", requireName)
	ioutil.WriteFile(inputFile, []byte(input), os.ModePerm)

	format := api.FormatCommonJS
	switch options.Format {
	case "esm":
		format = api.FormatESModule
	case "iife":
		format = api.FormatIIFE
	default:
		// nothing
	}

	result := api.Build(api.BuildOptions{
		EntryPoints:       []string{inputFile},
		Outdir:            project.OutDir,
		Bundle:            true,
		Write:             false,
		GlobalName:        options.GlobalName,
		LogLevel:          api.LogLevelInfo,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		MinifyWhitespace:  true,
		Format:            format,
	})

	if len(result.Errors) > 0 {
		log.Printf("build error: %+v\n", result.Errors)
		e, _ := json.Marshal(result.Errors)
		return nil, errors.New(string(e))
	}

	// write out files
	go func() {
		err := ioutil.WriteFile(project.OutFile, result.OutputFiles[0].Contents, os.ModePerm)
		if err != nil {
			log.Printf("write out file error: %+v\n", err)
		}
	}()
	return result.OutputFiles[0].Contents, nil
}

// Build reads file from cache or starts a fresh build
func (b *Builder) Build(options *BuildOptions, isForce bool) (interface{}, error) {
	key := fmt.Sprintf("%s-%s-%s", options.Pkg, options.GlobalName, options.Format)

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
		return b.buildFresh(options, &projectOptions{
			ProjectDir: projectDir,
			OutDir:     outDir,
			OutFile:    outFile,
		})
	})

	return content, err
}

func NewBuilder() *Builder {
	return &Builder{}
}
