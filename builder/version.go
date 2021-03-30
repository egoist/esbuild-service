package builder

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Masterminds/semver"
	"github.com/tidwall/gjson"
)

var (
	ErrNoMatchVersion = errors.New("no match version")
)

func GetPkgMatchVersion(pkgName, pkgVersion string) (v string, err error) {
	fmt.Println("Getting exact version for", pkgName, pkgVersion)
	resp, err := http.Get("https://registry.npmjs.org/" + pkgName)

	if err != nil {
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return
	}

	// version by dist tag
	distTag := gjson.GetBytes(body, fmt.Sprintf("%s.%s", "dist-tags", pkgVersion))

	if distTag.String() != "" {
		return distTag.String(), nil
	}

	c, err := semver.NewConstraint(pkgVersion)
	if err != nil {
		return
	}
	versions := gjson.GetBytes(body, "versions")

	versions.ForEach(func(key, value gjson.Result) bool {
		vv := key.String()
		version, err := semver.NewVersion(vv)
		if err != nil {
			return true
		}

		if c.Check(version) {
			v = vv
			return false
		}
		return true
	})

	if v == "" {
		err = ErrNoMatchVersion
	}
	return
}
