// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/cheggaaa/pb.v1"
)

// Timeout for an image pull in seconds - could be a large download...
const pullTimeout = 1800
const baseURLlibrary = "https://library.sylabs.io/v1/"

// DownloadImage will retrieve an image from the Container Library,
// saving it into the temp directory
func DownloadImage(filePath string, libraryURL string, Force bool, agentValue string) (string, error) {

	// Check if the reference rings true. This is dumb, why am i doing this again?
	ref, err := parseLibraryRef(libraryURL)
	if err != nil {
		fmt.Println(libraryURL)
		return "", fmt.Errorf("not a sylabs library ref")
	}

	// Define the path for the image to be stored
	if filePath == "" {
		refParts := strings.Split(ref, "/")
		name := strings.Split(refParts[len(refParts)-1], ":")
		filePath = fmt.Sprintf("/home/tomasgonzalez/Documents/%s-%s.sif", name[0], name[1])
	}

	// Check for a manifest to see if the image exists in the registry
	err = getManifest(ref)
	if err != nil {
		return "", fmt.Errorf("manifest not found on the registry")
	}
	if !Force {
		if _, err := os.Stat(filePath); err == nil {
			fmt.Println(filePath)
			return filePath, nil
		}
	}

	client := http.Client{
		Timeout: pullTimeout * time.Second,
	}
	url := baseURLlibrary + "/imagefile/" + ref
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", agentValue)

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("the requested image was not found in the library")
	}

	if res.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		buf.ReadFrom(res.Body)
		s := buf.String()
		return "", fmt.Errorf("Download did not succeed: %d %s\n\t",
			res.StatusCode, s)
	}

	// sylog.Debugf("OK response received, beginning body download\n")

	// Perms are 777 *prior* to umask
	out, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// sylog.Debugf("Created output file: %s\n", filePath)

	bodySize := res.ContentLength
	bar := pb.New(int(bodySize)).SetUnits(pb.U_BYTES)
	// if sylog.GetLevel() < 0 {
	// 	bar.NotPrint = true
	// }
	bar.ShowTimeLeft = true
	bar.ShowSpeed = true
	bar.Start()

	// create proxy reader
	bodyProgress := bar.NewProxyReader(res.Body)

	// Write the body to file
	_, err = io.Copy(out, bodyProgress)
	if err != nil {
		return "", err
	}

	bar.Finish()

	// sylog.Debugf("Download complete\n")
	fmt.Println(filePath)
	return filePath, nil

}

func parseLibraryRef(ref string) (string, error) {
	s := strings.TrimPrefix(ref, "library://")
	if s == ref {
		return "", fmt.Errorf("Not a library reference")
	}
	return s, nil
}

func getManifest(imageUri string) error {
	// Create a new http Hub client
	httpc := http.Client{
		Timeout: 30 * time.Second,
	}

	// Create the request, add headers context
	url, err := url.Parse(baseURLlibrary + "images/" + imageUri)
	if err != nil {
		return err
	}

	url.Scheme = "https"

	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return err
	}

	// Do the request, if status isn't success, return error
	res, err := httpc.Do(req)
	if res == nil {
		return fmt.Errorf("No response received from sylabs library")
	}
	if res.StatusCode == http.StatusNotFound {
		return fmt.Errorf("The requested manifest was not found in sylabs library")
	}
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = errors.New(res.Status)
		return err
	}

	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return nil
}
