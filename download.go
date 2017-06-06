package generator

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Download fetches a file over HTTP, saving it to the current directory.  It
// returns the filename and/or any error encountered during the download.
//
// Attempts are made to reuse any existing file, if it hasn't changed since
// the last download.
func Download(src string) (string, error) {
	client := http.DefaultClient

	// Determine filename
	parsedSrc, err := url.Parse(src)
	if err != nil {
		return "", err
	}
	path := strings.Split(parsedSrc.Path, "/")
	dest := path[len(path)-1]

	// Check if file on web server is newer than any existing file on disk
	stat, err := os.Stat(dest)
	if err == nil {
		request := &http.Request{
			Method: "HEAD",
			URL:    parsedSrc,
			Header: map[string][]string{
				"If-Modified-Since": {
					stat.ModTime().UTC().Format(time.RFC1123),
				},
			},
		}

		response, err := client.Do(request)
		if err != nil {
			return dest, err
		}

		if response.StatusCode == http.StatusNotModified {
			return dest, nil
		}
	}

	// Create temporary file
	wd, _ := os.Getwd()
	download, err := ioutil.TempFile(wd, "download")
	if err != nil {
		return "", err
	}
	defer os.Remove(download.Name())

	// Download file
	response, err := client.Get(src)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(download, response.Body)
	response.Body.Close()
	if err != nil {
		return "", err
	}

	// Replace destination with downloaded file
	err = AtomicFileReplace([]string{download.Name()}, []string{dest})
	if err != nil {
		return "", err
	}

	return dest, nil
}
