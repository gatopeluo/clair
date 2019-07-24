// Copyright 2017 clair authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package notification fetches notifications from the database and informs the
// specified remote handler about their existences, inviting the third party to
// actively query the API about it.

// Package imagefmt exposes functions to dynamically register methods to
// detect different types of container image formats.
package imagefmt

import (
	"crypto/tls"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

	slib "github.com/sylabs/singularity/pkg/client/net"
	shub "github.com/sylabs/singularity/pkg/client/shub"

	"github.com/tigonza/clair/pkg/commonerr"
	"github.com/tigonza/clair/pkg/httputil"
	"github.com/tigonza/clair/pkg/tarutil"
)

var (
	// ErrCouldNotFindLayer is returned when we could not download or open the layer file.
	ErrCouldNotFindLayer = commonerr.NewBadRequestError("could not find layer")

	// insecureTLS controls whether TLS server's certificate chain and hostname are verified
	// when pulling layers, verified in default.
	insecureTLS = false

	extractorsM sync.RWMutex
	extractors  = make(map[string]Extractor)
)

// Extractor represents an ability to extract files from a particular container
// image format.
type Extractor interface {
	// ExtractFiles produces a tarutil.FilesMap from a image layer.
	ExtractFiles(layer io.ReadCloser, filenames []string) (tarutil.FilesMap, error)
}

// RegisterExtractor makes an extractor available by the provided name.
//
// If called twice with the same name, the name is blank, or if the provided
// Extractor is nil, this function panics.
func RegisterExtractor(name string, d Extractor) {
	extractorsM.Lock()
	defer extractorsM.Unlock()

	if name == "" {
		panic("imagefmt: could not register an Extractor with an empty name")
	}

	if d == nil {
		panic("imagefmt: could not register a nil Extractor")
	}

	// Enforce lowercase names, so that they can be reliably be found in a map.
	name = strings.ToLower(name)

	if _, dup := extractors[name]; dup {
		panic("imagefmt: RegisterExtractor called twice for " + name)
	}

	extractors[name] = d
}

// Extractors returns the list of the registered extractors.
func Extractors() map[string]Extractor {
	extractorsM.RLock()
	defer extractorsM.RUnlock()

	ret := make(map[string]Extractor)
	for k, v := range extractors {
		ret[k] = v
	}

	return ret
}

// UnregisterExtractor removes a Extractor with a particular name from the list.
func UnregisterExtractor(name string) {
	extractorsM.Lock()
	defer extractorsM.Unlock()
	delete(extractors, name)
}

// Extract streams an image layer from disk or over HTTP, determines the
// image format, then extracts the files specified.
func Extract(format, path string, headers map[string]string, toExtract []string) (tarutil.FilesMap, error) {
	var layerReader io.ReadCloser
	layerLoaded := false
	if format == "Singularity" {
		// Generate valid user-agent value
		agentValue := httputil.GetWithSingularityUseAgent()

		var image string
		var err error

		if strings.HasPrefix(path, "shub://") {
			//Download the image through the shub API
			image, err = shub.DownloadImage("", path, false, false, agentValue)
			if err != nil {
				return nil, err
			}
			if image == "" {
				return nil, commonerr.NewBadRequestError("no name for image")
			}
		} else {
			//Download the image through the singularity official library
			image, err = slib.DownloadImage("", path, false, agentValue)
			if err != nil {
				return nil, err
			}
			if image == "" {
				return nil, commonerr.NewBadRequestError("no name for image")
			}
		}

		//If squashfs is detected, all files will be sent to a temporary directory
		dir, err := shub.ConvertImage(image, "")
		if err != nil {
			return nil, err
		}
		fmt.Println("starting tarball")

		err = CompressImg(dir)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}

		fmt.Println("tarball done")
		layerReader, err = os.Open("/home/tomasgonzalez/Documents/layer.tar")
		if err != nil {
			return nil, ErrCouldNotFindLayer
		}
		layerLoaded = true
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		// Create a new HTTP request object.
		request, err := http.NewRequest("GET", path, nil)
		if err != nil {
			return nil, ErrCouldNotFindLayer
		}

		// Set any provided HTTP Headers.
		if headers != nil {
			for k, v := range headers {
				request.Header.Set(k, v)
			}
		}

		// Send the request and handle the response.
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureTLS},
			Proxy:           http.ProxyFromEnvironment,
		}
		client := &http.Client{Transport: tr}
		r, err := client.Do(request)
		if err != nil {
			log.WithError(err).Warning("could not download layer")
			return nil, ErrCouldNotFindLayer
		}

		// Fail if we don't receive a 2xx HTTP status code.
		if math.Floor(float64(r.StatusCode/100)) != 2 {
			log.WithField("status code", r.StatusCode).Warning("could not download layer: expected 2XX")
			return nil, ErrCouldNotFindLayer
		}

		layerReader = r.Body
	} else {
		var err error
		if !layerLoaded {
			layerReader, err = os.Open(path)
			if err != nil {
				return nil, ErrCouldNotFindLayer
			}
			layerLoaded = true
		}
	}
	defer layerReader.Close()

	if extractor, exists := Extractors()[strings.ToLower(format)]; exists {
		files, err := extractor.ExtractFiles(layerReader, toExtract)
		if err != nil {
			return nil, err
		}
		return files, nil
	}

	return nil, commonerr.NewBadRequestError(fmt.Sprintf("unsupported image format '%s'", format))
}

// SetInsecureTLS sets the insecureTLS to control whether TLS server's certificate chain
// and hostname are verified when pulling layers.
func SetInsecureTLS(insecure bool) {
	insecureTLS = insecure
}

//CompressImg compresses the mounted FS into a tarball
func CompressImg(dir string) (err error) {
	cmd1 := "chmod -R 777 " + dir
	cmd2 := "cd " + dir
	cmd3 := "tar -cf /home/tomasgonzalez/Documents/layer.tar *"
	cmd := exec.Command("/bin/sh", "-c", cmd1+";"+cmd2+";"+cmd3)
	return cmd.Run()
}
