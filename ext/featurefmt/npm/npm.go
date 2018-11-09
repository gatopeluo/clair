package npm

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"strings"

	"github.com/gatopeluo/clair/database"
	"github.com/gatopeluo/clair/ext/featurefmt"
	"github.com/gatopeluo/clair/pkg/commonerr"
	"github.com/gatopeluo/clair/pkg/tarutil"
)

func init() {
	featurefmt.RegisterLister("npm", &lister{})
}

type lister struct{}

func (l lister) ListFeatures(files tarutil.FilesMap) ([]database.FeatureVersion, error) {

	//Make a map for the eventual features
	Libs := make(map[string]string)

	// Fill Libs, using name of package as key, and filepath as value
	for i := range files {
		if !strings.HasSuffix(i, "package.json") {
			delete(files, i)
		} else {
			auxSlice := strings.Split(i, "/")
			filepath := auxSlice[len(auxSlice)-2]
			Libs[filepath] = i
		}
	}
	if len(files) == 0 {
		return []database.FeatureVersion{}, nil
	}

	//Load File on temporal directory
	tmpDir, err := ioutil.TempDir(os.TempDir(), "node")
	defer os.RemoveAll(tmpDir)
	if err != nil {
		return []database.FeatureVersion{}, commonerr.ErrFilesystem
	}

	// Create a map to store packages and ensure their uniqueness
	packagesMap := make(map[string]database.FeatureVersion)

	// Check all strings on Libs (filepaths)
	for _, fpath := range Libs {
		//get bytes for file on fpath
		f := files[fpath]

		// initiate NodePackage and database.FeatureVersion structs
		var pkgInfo NodePackage
		var pkg database.FeatureVersion

		// Deploy the features info from the json file into the struct
		json.Unmarshal(f, &pkgInfo)

		//Add the info to the features
		pkg.Feature.Name = pkgInfo.Name
		pkg.Version = pkgInfo.Version
		if pkg.Feature.Name != "" && pkg.Version != "" {
			packagesMap[pkg.Feature.Name+"#"+pkg.Version] = pkg
		}

	}

	//Clear the mapping
	packages := make([]database.FeatureVersion, 0, len(packagesMap))
	for _, pkg := range packagesMap {
		packages = append(packages, pkg)
	}

	return packages, nil
}

func (l lister) RequiredFilenames() []string {
	nodeRoutes := []string{"node"}
	return nodeRoutes
}

// NodePackage is the struct used for importing the strings required directly from the json which has them
type NodePackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
