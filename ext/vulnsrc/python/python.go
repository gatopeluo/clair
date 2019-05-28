// Package python downloads a JSON file from the safety-db repo, which is a open source db
// of pip packages vulnerability data.
package python

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/gatopeluo/clair/database"
	"github.com/gatopeluo/clair/ext/vulnsrc"
)

const (
	safetyDbRepo = "https://github.com/pyupio/safety-db"
	updaterFlag  = "python-sec-db"
	nvdURLPrefix = "https://cve.mitre.org/cgi-bin/cvename.cgi?name="
)

var ns = []string{"centos:7", "alpine:3.3", "alpine:3.4", "alpine:3.5", "alpine:3.6", "alpine:3.7", "alpine:3.8", "alpine:3.9", "debian:8",
	"debian:9", "debian:10", "debian:unstable", "ubuntu:12.04", "ubuntu:12.10", "ubuntu:13.04", "ubuntu:14.04", "ubuntu:14.10", "ubuntu:15.04",
	"ubuntu:15.10", "ubuntu:16.04", "ubuntu:16.10", "ubuntu:17.04", "ubuntu:17.10", "ubuntu:18.04"}

type updater struct {
	repositoryLocalPath string
}

// SafetyVuln is the struct used for unmarshalling the vulnerabilities
type SafetyVuln struct {
	Advisory string   `json:"advisory"`
	Cve      string   `json:"cve"`
	ID       string   `json:"id"`
	Specs    []string `json:"specs"`
	Version  string   `json:"v"`
}

func init() {
	vulnsrc.RegisterUpdater("python", &updater{})
}

func (u *updater) Update(db database.Datastore) (resp vulnsrc.UpdateResponse, err error) {

	// generate a commit for the flagValue of response
	commit, err := u.pull()

	// Ask the database for the latest commit we successfully applied.
	var dbCommit string
	dbCommit, err = db.GetKeyValue(updaterFlag)
	if err != nil {
		return
	}

	// Set the updaterFlag to equal the commit processed.
	resp.FlagName = updaterFlag
	resp.FlagValue = commit

	// Short-circuit if there have been no updates.
	if commit == dbCommit {
		return
	}

	// Here's where the issue starts. Since vulnerabilities are linked not to
	// features directly but to namespaces first; just like with versionfmt, it
	// becomes impossible to update the database without either:
	// 		a) Waiting for a layer to be processed first, or
	// 		b) just adding them to each namespace, for each vulnerabilitie.
	if u.repositoryLocalPath == "" {
		return
	}

	// Open our vulnerabilitie data
	full, err := os.Open(u.repositoryLocalPath + "/data/insecure_full.json")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer full.Close()
	byteValue, _ := ioutil.ReadAll(full)

	// We unmarshal our byteArray
	var result map[string][]SafetyVuln
	json.Unmarshal([]byte(byteValue), &result)

	//  start the parsing of the vulnerabilities
	var vulns []database.Vulnerability

	for name, vs := range result {
		for _, v := range vs {
			if v.Cve != "" {
				vulns = u.addingVulns(name, v)
			}
		}
	}

	//resp.Vulnerabilities = vulns
	fmt.Println(vulns[0])
	fmt.Println(vulns[1])
	fmt.Println(vulns[10])

	u.Clean()
	return
}

func (u *updater) Clean() {
	if u.repositoryLocalPath != "" {
		os.RemoveAll(u.repositoryLocalPath)
	}
}

func (u *updater) pull() (commit string, err error) {

	// If the repository doesn't exist, clone it.
	if _, pathExists := os.Stat(u.repositoryLocalPath); u.repositoryLocalPath == "" || os.IsNotExist(pathExists) {
		if u.repositoryLocalPath, err = ioutil.TempDir(os.TempDir(), "safety-db"); err != nil {
			return
		}

		cmd := exec.Command("git", "clone", safetyDbRepo, ".")
		cmd.Dir = u.repositoryLocalPath

		if _, err = cmd.CombinedOutput(); err != nil {
			u.Clean()
			return
		}

	} else {

		// The repository already exists and it needs to be refreshed via a pull.
		cmd := exec.Command("git", "pull")
		cmd.Dir = u.repositoryLocalPath

		if _, err = cmd.CombinedOutput(); err != nil {
			return
		}
	}

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = u.repositoryLocalPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return
	}

	commit = strings.TrimSpace(string(out))

	return
}

func (u *updater) addingVulns(pkgName string, v SafetyVuln) (dbv []database.Vulnerability) {
	for _, nsName := range ns {
		var vuln database.Vulnerability
		vuln.Severity = database.UnknownSeverity
		vuln.Name = v.Cve
		vuln.Link = nvdURLPrefix + v.Cve
		vuln.FixedIn = []database.FeatureVersion{
			{
				Feature: database.Feature{
					Namespace: database.Namespace{
						Name:          nsName,
						VersionFormat: "pip",
					},
					Name: pkgName,
				},
				Version: strings.Trim(v.Version, "<>"),
			},
		}
		dbv = append(dbv, vuln)
	}
	return
}
