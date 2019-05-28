// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	util "github.com/sylabs/singularity/pkg/client/library"
	pb "gopkg.in/cheggaaa/pb.v1"
)

// Timeout for an image pull in seconds (2 hours)
const pullTimeout = 7200

// DownloadImage will retrieve an image from the Container Singularityhub,
// saving it into the specified file
func DownloadImage(filePath string, shubRef string, force, noHTTPS bool, agentValue string) (dir string, err error) {

	// use custom parser to make sure we have a valid shub URI
	if ok := isShubPullRef(shubRef); !ok {
	}

	ShubURI, err := shubParseReference(shubRef)
	if err != nil {
		return "", fmt.Errorf("Failed to parse shub URI: %v", err)
	}

	if filePath == "" {
		filePath = fmt.Sprintf("/tmp/%s_%s.simg", ShubURI.container, ShubURI.tag)
	}

	if !force {
		if _, err := os.Stat(filePath); err == nil {
			return filePath, nil
		}
	}

	// Get the image manifest
	manifest, err := getManifest(ShubURI, noHTTPS, agentValue)
	if err != nil {
		return "", fmt.Errorf("Failed to get manifest from Shub: %v", err)
	}

	fmt.Println("step 4 done, got manifest")

	// Get the image based on the manifest
	httpc := http.Client{
		Timeout: pullTimeout * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, manifest.Image, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", agentValue)

	if noHTTPS {
		req.URL.Scheme = "http"
	}

	// Do the request, if status isn't success, return error
	resp, err := httpc.Do(req)
	if resp == nil {
		return "", fmt.Errorf("No response received from singularity hub")
	}
	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("The requested image was not found in singularity hub")
	}

	if resp.StatusCode != http.StatusOK {
		jRes, err := util.ParseErrorBody(resp.Body)
		if err != nil {
			jRes = util.ParseErrorResponse(resp)
		}
		return "", fmt.Errorf("Download did not succeed: %d %s\n\t%v",
			jRes.Error.Code, jRes.Error.Status, jRes.Error.Message)
	}

	// Perms are 777 *prior* to umask
	out, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
	if err != nil {
		return "", err
	}
	defer out.Close()

	bodySize := resp.ContentLength
	bar := pb.New(int(bodySize)).SetUnits(pb.U_BYTES)

	bar.ShowTimeLeft = true
	bar.ShowSpeed = true
	bar.Start()

	// create proxy reader
	bodyProgress := bar.NewProxyReader(resp.Body)

	// Write the body to file
	bytesWritten, err := io.Copy(out, bodyProgress)
	if err != nil {
		return "", err
	}
	// Simple check to make sure image received is the correct size
	if bytesWritten != resp.ContentLength {
		return "", fmt.Errorf("Image received is not the right size. Supposed to be: %v  Actually: %v", resp.ContentLength, bytesWritten)
	}

	bar.Finish()
	fmt.Println("finished")

	return filePath, err
}
