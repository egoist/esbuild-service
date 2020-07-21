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
	Platform   string
	IsMinify   bool
}

type projectOptions struct {
	InputFile   string
	OutDir      string
	ProjectDir  string
	RequireName string
}

// build without cache
func (b *Builder) buildFresh(options *BuildOptions, project *projectOptions) (interface{}, error) {
	log.Printf("trigger build %s, %s", options.Pkg, time.Now())

	format := api.FormatESModule
	switch options.Format {
	case "cjs":
		format = api.FormatCommonJS
	case "iife":
		format = api.FormatIIFE
	default:
		// nothing
	}

	platform := api.PlatformBrowser
	if options.Platform == "node" {
		platform = api.PlatformNode
	}

	result := api.Build(api.BuildOptions{
		EntryPoints:       []string{project.InputFile},
		Outdir:            project.OutDir,
		Bundle:            true,
		Write:             false,
		GlobalName:        options.GlobalName,
		LogLevel:          api.LogLevelInfo,
		MinifyIdentifiers: options.IsMinify,
		MinifySyntax:      options.IsMinify,
		MinifyWhitespace:  options.IsMinify,
		Format:            format,
		Platform:          platform,
		Externals: []string{
			// exclude modules that don't make sense in browser
			"fs",
			"os",
			"fsevents",
		},
		Defines: map[string]string{
			"process.env.NODE_ENV": "\"production\"",
		},
	})

	if len(result.Errors) > 0 {
		log.Printf("build error: %+v\n", result.Errors)
		e, _ := json.Marshal(result.Errors)
		return nil, errors.New(string(e))
	}

	return result.OutputFiles[0].Contents, nil
}

// Build starts a fresh build and install the package if it doesn't exist
func (b *Builder) Build(options *BuildOptions, isForce bool) (interface{}, error) {
	// We need to send a request to npm registry to find out the version first
	parsedPkg := parsePkgName(options.Pkg)
	requireName := getRequiredPkg(parsedPkg)
	version := parsedPkg[2]
	if version == "" {
		version = "latest"
	}
	pkgVersion, err := getPkgMatchVersion(parsedPkg[0], version)
	if err != nil {
		logError(err, fmt.Sprintf("failed to get pkg match version: %s", options.Pkg))
		return nil, err
	}

	installName := fmt.Sprintf("%s@%s", parsedPkg[0], pkgVersion)
	key := fmt.Sprintf("%s-%s", parsedPkg[0], pkgVersion)
	cacheDir := path.Join(os.TempDir(), "esbuild-service-cache")
	projectDir := path.Join(cacheDir, key)
	outDir := path.Join(projectDir, "out")

	if !pathExists(outDir) {
		os.MkdirAll(outDir, os.ModePerm)
	}

	_, err, _ = b.g.Do("init", func() (i interface{}, err error) {
		if !pathExists(path.Join(cacheDir, "package.json")) {
			log.Println("Installing node-browser-libs")
			cmd := exec.Command("yarn", "add",
				"assert@^1.1.1",
				"buffer",
				"crypto@npm:crypto-browserify",
				"events",
				"path@npm:path-browserify",
				"process",
				"punycode",
				"querystring@npm:querystring-es3",
				"stream@npm:stream-browserify",
				"string_decoder",
				"http@npm:stream-http",
				"https@npm:https-browserify",
				"tty@npm:tty-browserify",
				"url",
				"util",
				"vm@npm:vm-browserify",
				"zlib@npm:browserify-zlib@^0.2.0",
			)
			cmd.Dir = cacheDir
			_, err := cmd.Output()
			if err != nil {
				logError(err, "failed to install browser-node-libs")
				return nil, err
			}
		}
		return nil, nil
	})

	if err != nil {
		return nil, err
	}

	inputFile, err, _ := b.g.Do(key, func() (interface{}, error) {
		// Install the package if not already install
		if isForce || !pathExists(path.Join(projectDir, "node_modules")) {
			// Install the package
			log.Println("Installing", options.Pkg, "in", outDir)

			log.Printf("pkg %s install %s require %s\n", options.Pkg, installName, requireName)

			log.Printf("install in %s", projectDir)

			// Use `yarn init -y` to create a package.json file
			// Otherwise the package will be installed in parent directory
			yarnInit := exec.Command("yarn", "init", "-y")
			yarnInit.Dir = projectDir
			_, err := yarnInit.Output()
			if err != nil {
				logError(err, "failed to run yarn init")
				return nil, err
			}

			yarnAdd := exec.Command(
				"yarn",
				"add",
				installName,
			)
			yarnAdd.Dir = projectDir
			_, err = yarnAdd.Output()
			if err != nil {
				logError(err, "failed to install pkg")
				return nil, err
			}

		}

		inputFile := path.Join(projectDir, "input.js")
		input := fmt.Sprintf("module.exports = require('%s')", requireName)
		ioutil.WriteFile(inputFile, []byte(input), os.ModePerm)
		return inputFile, nil
	})

	if err != nil {
		return nil, err
	}

	return b.buildFresh(options, &projectOptions{
		ProjectDir:  projectDir,
		OutDir:      outDir,
		RequireName: requireName,
		InputFile:   inputFile.(string),
	})
}

func NewBuilder() *Builder {
	return &Builder{}
}
