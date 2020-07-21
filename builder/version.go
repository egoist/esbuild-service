package builder

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/Masterminds/semver"
	"github.com/tidwall/gjson"
)

var (
	ErrNoMatchVersion = errors.New("no match version")
)

func getPkgMatchVersion(pkgName, pkgVersion string) (v string, err error) {
	cmd := exec.Command("yarn", "info", pkgName, "--json")
	out, err := cmd.Output()
	if err != nil {
		return
	}

	// version by dist tag
	distTag := gjson.GetBytes(out, fmt.Sprintf("data.%s.%s", "dist-tags", pkgVersion))
	if distTag.String() != "" {
		return distTag.String(), nil
	}

	c, err := semver.NewConstraint(pkgVersion)
	if err != nil {
		return
	}
	versions := gjson.GetBytes(out, "data.versions").Array()
	for _, value := range versions {
		vv := value.String()
		version, err := semver.NewVersion(vv)
		if err != nil {
			continue
		}
		if c.Check(version) {
			v = vv
		}
	}
	if v == "" {
		err = ErrNoMatchVersion
	}
	return
}
