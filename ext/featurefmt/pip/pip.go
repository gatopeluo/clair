package pip

import (
	"fmt"
	"strings"

	"github.com/gatopeluo/clair/database"
	"github.com/gatopeluo/clair/ext/featurefmt"
	"github.com/gatopeluo/clair/pkg/tarutil"
)

func init() {
	featurefmt.RegisterLister("pip", &lister{})
}

type lister struct{}

func (l lister) ListFeatures(files tarutil.FilesMap) ([]database.FeatureVersion, error) {

	//get names of libs directories
	names := []string{"egg-info", "dist-info"}

	//Start checking for egg-info's
	auxMap := make(map[string]string)
	for i := range files {
		//for libs of python 2.7 and 3.6 on 32 and 64 bits.
		for _, str := range names {
			if strings.Contains(i, str) {
				auxMap[i] = i
			}
		}
	}

	//If empty, it returns an empty list with no errors
	if len(auxMap) == 0 {
		return []database.FeatureVersion{}, nil
	}

	for i := range auxMap {
		if !(strings.Contains(i, "dist-info") || strings.Contains(i, "egg-info")) {
			delete(auxMap, auxMap[i])
		}
	}
	Libs := auxMap

	// Create a map to store packages and ensure their uniqueness
	packagesMap := make(map[string]database.FeatureVersion)

	for fpath := range Libs {
		//get bytes for file on fpath
		f, _ := files[fpath]
		file := strings.Split(string(f), "\n")
		var pkg database.FeatureVersion
		counter := 0
		for _, s := range file {
			aux := strings.Split(s, ":")

			// Here we do a switch-case for each line, this is to check wether it
			// has the info for the version or the name of the package
			switch aux[0] {

			case "Name":
				counter++
				pkg.Feature = database.Feature{Name: aux[1][1:]}

			case "Version":
				// Version starts the line from the second char after de ':' cause
				// there is always a space before the version
				pkg.Version = aux[1][1:]
				counter++

			default:
				fmt.Println(s)
			}
		}
		if pkg.Feature.Name != "" && pkg.Version != "" {
			pkg.VersionFormat = "pip"
			packagesMap[pkg.Feature.Name+"#"+pkg.Version] = pkg
			fmt.Println(pkg.Feature.Name)
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
	pipRoutes := []string{"pip"}
	return pipRoutes
}
