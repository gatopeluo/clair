// Package singularity implements an imagefmt.Extractor for singularity formatted
// container images.
package singularity

import (
	"io"

	"github.com/tigonza/clair/ext/imagefmt"
	"github.com/tigonza/clair/pkg/tarutil"
)

type format struct{}

func init() {
	imagefmt.RegisterExtractor("singularity", &format{})
}

func (f format) ExtractFiles(layerReader io.ReadCloser, toExtract []string) (tarutil.FilesMap, error) {
	return tarutil.ExtractFiles(layerReader, toExtract)
}
