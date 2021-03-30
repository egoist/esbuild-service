package builder

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/egoist/esbuild-service/logger"
	"github.com/egoist/esbuild-service/util"
	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/sync/singleflight"
)

var log = logger.Logger

func logError(err error, prefix string) {
	log.Errorf("error %s %s\n", prefix, err.Error())
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
	ParsedPkg  util.ParsedPkgPathname
	PkgVersion string
}

var httpPlugin = api.Plugin{
	Name: "http",
	Setup: func(build api.PluginBuild) {
		build.OnResolve(api.OnResolveOptions{Filter: ".*"}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
			// Conver package name to skypack cdn address
			matched, err := regexp.Match(`^[a-z@_]`, []byte(args.Path))
			if err != nil {
				return api.OnResolveResult{}, err
			}
			if matched {
				return api.OnResolveResult{
					Path:      "https://cdn.skypack.dev/" + args.Path,
					Namespace: "http-url",
				}, nil
			}
			return api.OnResolveResult{}, nil
		})

		// Intercept import paths starting with "http:" and "https:" so
		// esbuild doesn't attempt to map them to a file system location.
		// Tag them with the "http-url" namespace to associate them with
		// this plugin.
		build.OnResolve(api.OnResolveOptions{Filter: `^https?://`},
			func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				return api.OnResolveResult{
					Path:      args.Path,
					Namespace: "http-url",
				}, nil
			})

		// We also want to intercept all import paths inside downloaded
		// files and resolve them against the original URL. All of these
		// files will be in the "http-url" namespace. Make sure to keep
		// the newly resolved URL in the "http-url" namespace so imports
		// inside it will also be resolved as URLs recursively.
		build.OnResolve(api.OnResolveOptions{Filter: ".*", Namespace: "http-url"},
			func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				base, err := url.Parse(args.Importer)
				if err != nil {
					return api.OnResolveResult{}, err
				}
				relative, err := url.Parse(args.Path)
				if err != nil {
					return api.OnResolveResult{}, err
				}
				return api.OnResolveResult{
					Path:      base.ResolveReference(relative).String(),
					Namespace: "http-url",
				}, nil
			})

		// When a URL is loaded, we want to actually download the content
		// from the internet. This has just enough logic to be able to
		// handle the example import from unpkg.com but in reality this
		// would probably need to be more complex.
		build.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: "http-url"},
			func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				println("Load", args.Path)
				res, err := http.Get(args.Path)
				if err != nil {
					return api.OnLoadResult{}, err
				}
				defer res.Body.Close()
				bytes, err := ioutil.ReadAll(res.Body)
				if err != nil {
					return api.OnLoadResult{}, err
				}
				contents := string(bytes)
				if res.StatusCode < 200 || res.StatusCode >= 300 {
					return api.OnLoadResult{}, errors.New(contents)
				}
				return api.OnLoadResult{Contents: &contents}, nil
			})
	},
}

// build without cache
func (b *Builder) buildFresh(options *BuildOptions) (interface{}, error) {
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
		EntryPoints:       []string{options.Pkg},
		Outdir:            "/dist",
		Bundle:            true,
		Write:             false,
		GlobalName:        options.GlobalName,
		LogLevel:          api.LogLevelInfo,
		MinifyIdentifiers: options.IsMinify,
		MinifySyntax:      options.IsMinify,
		MinifyWhitespace:  options.IsMinify,
		Format:            format,
		Platform:          platform,
		External: []string{
			// exclude modules that don't make sense in browser
			"fs",
			"os",
			"fsevents",
		},
		Define: map[string]string{
			"process.env.NODE_ENV": "\"production\"",
		},
		Plugins: []api.Plugin{httpPlugin},
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
	return b.buildFresh(options)
}

func NewBuilder() *Builder {
	return &Builder{}
}
