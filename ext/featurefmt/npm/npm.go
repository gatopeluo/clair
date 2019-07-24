package npm

import (
	"encoding/json"
	"fmt"

	"strings"

	"github.com/tigonza/clair/database"
	"github.com/tigonza/clair/ext/featurefmt"
	"github.com/tigonza/clair/pkg/tarutil"
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
		if strings.Contains(i, "node_modules") {
			auxSlice := strings.Split(i, "/")
			filename := auxSlice[len(auxSlice)-2]
			Libs[filename] = i
		}
	}
	if len(files) == 0 {
		return []database.FeatureVersion{}, fmt.Errorf("no features")
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
		err := json.Unmarshal(f, &pkgInfo)

		//Add the info to the features
		if err == nil {
			pkg.Feature.Name = pkgInfo.Name
			pkg.Version = pkgInfo.Version
			pkg.VersionFormat = "npm"
			if pkg.Feature.Name != "" && pkg.Version != "" {
				packagesMap[pkg.Feature.Name+"#"+pkg.Version] = pkg
			}
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
