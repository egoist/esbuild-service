package util

import (
	"errors"
	"fmt"
	"regexp"
)

var pkgPathnameRe = regexp.MustCompile("^((?:@[^/@]+/)?[^/@]+)(?:@([^/]+))?(/.*)?$")

type ParsedPkgPathname struct {
	Name     string
	Version  string
	Spec     string
	Filename string
}

func ParsePkgName(pkg string) (parsed ParsedPkgPathname, err error) {
	m := pkgPathnameRe.FindStringSubmatch(pkg)

	if m == nil {
		err = errors.New("Invaliad package name")
		return
	}

	name := m[1]
	version := m[2]

	parsed = ParsedPkgPathname{
		Name:     name,
		Version:  version,
		Filename: m[3],
		Spec:     name + "@" + version,
	}

	return
}

func GetRequiredPkg(parsed [3]string) string {
	if parsed[1] == "" {
		return parsed[0]
	}
	return fmt.Sprintf("%s/%s", parsed[0], parsed[1])
}
