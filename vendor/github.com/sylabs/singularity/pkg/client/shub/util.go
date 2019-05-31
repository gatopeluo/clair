// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/sylabs/singularity/pkg/image"
	"github.com/sylabs/singularity/pkg/image/unpacker"
)

// JSONError - Struct for standard error returns over REST API
type JSONError struct {
	Code    int    `json:"code,omitempty"`
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

// JSONResponse - Top level container of a REST API response
type JSONResponse struct {
	Data  interface{} `json:"data"`
	Error JSONError   `json:"error,omitempty"`
}

// isShubPullRef returns true if the provided string is a valid Shub
// reference for a pull operation.
func isShubPullRef(shubRef string) bool {
	// define regex for each URI component
	registryRegexp := `([-.a-zA-Z0-9/]{1,64}\/)?`           // target is very open, outside registry
	nameRegexp := `([-a-zA-Z0-9]{1,39}\/)`                  // target valid github usernames
	containerRegexp := `([-_.a-zA-Z0-9]{1,64})`             // target valid github repo names
	tagRegexp := `(:[-_.a-zA-Z0-9]{1,64})?`                 // target is very open, file extensions or branch names
	digestRegexp := `((\@[a-f0-9]{32})|(\@[a-f0-9]{40}))?$` // target file md5 has, git commit hash, git branch

	// expression is anchored
	shubRegex, err := regexp.Compile(`^(shub://)` + registryRegexp + nameRegexp + containerRegexp + tagRegexp + digestRegexp + `$`)
	if err != nil {
		return false
	}

	found := shubRegex.FindString(shubRef)

	// sanity check
	// if found string is not equal to the input, input isn't a valid URI
	return shubRef == found
}

// shubParseReference accepts a valid Shub reference string and parses its content
// It will return an error if the given URI is not valid,
// otherwise it will parse the contents into a ShubURI struct
func shubParseReference(src string) (uri ShubURI, err error) {
	ShubRef := strings.TrimPrefix(src, "shub://")
	refParts := strings.Split(ShubRef, "/")

	if l := len(refParts); l > 2 {
		// more than two pieces indicates a custom registry
		uri.registry = strings.Join(refParts[:l-2], "") + shubAPIRoute
		uri.user = refParts[l-2]
		src = refParts[l-1]
	} else if l == 2 {
		// two pieces means default registry
		uri.registry = defaultRegistry + shubAPIRoute
		uri.user = refParts[l-2]
		src = refParts[l-1]
	} else if l < 2 {
		return ShubURI{}, errors.New("Not a valid Shub reference")
	}

	// look for an @ and split if it exists
	if strings.Contains(src, `@`) {
		refParts = strings.Split(src, `@`)
		uri.digest = `@` + refParts[1]
		src = refParts[0]
	}

	// look for a : and split if it exists
	if strings.Contains(src, `:`) {
		refParts = strings.Split(src, `:`)
		uri.tag = `:` + refParts[1]
		src = refParts[0]
	}

	// container name is left over after other parts are split from it
	uri.container = src

	if uri.tag == "" && uri.digest == "" {
		uri.tag = ":latest"
	}

	return uri, nil
}

//ConvertImage turns a singularity image into a directory
func ConvertImage(filename string, unsquashfsPath string) (string, error) {
	img, err := image.Init(filename, false)
	if err != nil {
		return "", fmt.Errorf("could not open image %s: %s", filename, err)
	}
	defer img.File.Close()

	// squashfs only
	if img.Partitions[0].Type != image.SQUASHFS {
		return "", fmt.Errorf("not a squashfs root filesystem")
	}

	// create a reader for rootfs partition
	reader, err := image.NewPartitionReader(img, "", 0)
	if err != nil {
		return "", fmt.Errorf("could not extract root filesystem: %s", err)
	}
	s := unpacker.NewSquashfs()
	if !s.HasUnsquashfs() && unsquashfsPath != "" {
		s.UnsquashfsPath = unsquashfsPath
	}

	// keep compatibility with v2
	tmpdir := os.Getenv("SINGULARITY_LOCALCACHEDIR")
	if tmpdir == "" {
		tmpdir = os.Getenv("SINGULARITY_CACHEDIR")
	}

	// create temporary sandbox
	dir, err := ioutil.TempDir(tmpdir, "rootfs-")
	if err != nil {
		return "", fmt.Errorf("could not create temporary sandbox: %s", err)
	}

	fmt.Println(dir)
	// extract root filesystem
	if err := s.ExtractAll(reader, dir); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("root filesystem extraction failed: %s", err)
	}

	return dir, err
}

// ParseErrorResponse - Create a JSONResponse out of a raw HTTP response
func ParseErrorResponse(res *http.Response) (jRes JSONResponse) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	s := buf.String()
	jRes.Error.Code = res.StatusCode
	jRes.Error.Status = http.StatusText(res.StatusCode)
	jRes.Error.Message = s
	return jRes
}

// ParseErrorBody - Parse an API format error out of the body
func ParseErrorBody(r io.Reader) (jRes JSONResponse, err error) {
	err = json.NewDecoder(r).Decode(&jRes)
	if err != nil {
		return jRes, fmt.Errorf("The server returned a response that could not be decoded: %v", err)
	}
	return jRes, nil
}
