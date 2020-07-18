package handler

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"

	"github.com/evanw/esbuild/pkg/api"
)

// Handler handles the API request
func Handler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	globalNames, ok := query["globalName"]

	if !ok || len(globalNames[0]) < 1 {
		log.Println("Url Param 'globalName' is missing")
		return
	}

	pkgs, ok := query["pkg"]

	if !ok || len(pkgs[0]) < 1 {
		log.Println("Url Param 'pkg' is missing")
		return
	}

	globalName := globalNames[0]
	pkg := pkgs[0]

	projectDir := path.Join(os.TempDir(), pkg)
	outDir := path.Join(projectDir, "out")

	os.MkdirAll(outDir, os.ModePerm)

	outFile := path.Join(outDir, "input.js")

	if _, err := os.Stat(outFile); !os.IsNotExist(err) {
		file, err := ioutil.ReadFile(outFile)
		if err != nil {
			log.Println(err)
			return
		}
		w.Header().Set("content-type", "application/javascript")
		w.Write(file)
		return
	}

	// Install the package
	log.Println("Installing", pkg, "in", outDir)
	c := exec.Command("node", "--version")
	out, err := c.Output()
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(out)
	cmd := exec.Command("yarn", "add", pkg)
	cmd.Dir = projectDir
	_, err = cmd.Output()
	if err != nil {
		log.Println("failed to install pkg", err)
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
		os.Exit(1)
	}

	w.Header().Set("content-type", "application/javascript")
	w.Write(result.OutputFiles[0].Contents)
}
