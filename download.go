package generator

import (
	"errors"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Download fetches a file over HTTP, saving it to the current directory.  It
// returns the filename and/or any error encountered during the download.
//
// The optional args currently consist only of the destination filename.  If
// not specified, it defaults to the filename portion of src.
//
// Attempts are made to reuse any existing download, if it hasn't changed.  If
// no destination filename was specified, and the server sets the filename on
// download to something other than what's contained in the URL, the existing
// file may not be detected.
func Download(src string, args ...string) (string, error) {
	client := http.DefaultClient

	parsedSrc, err := url.Parse(src)
	if err != nil {
		return "", err
	}

	// Earlier versions of this function took a single argument: the URL
	// to retrieve.  The second argument (destination filename) has been
	// added, but is optional so that existing code doesn't need to be
	// updated.  Passing more than one extra argument is an error.
	var dest string
	switch len(args) {
	case 0:
		// Use filename from source URL
		path := strings.Split(parsedSrc.Path, "/")
		dest = path[len(path)-1]
	case 1:
		dest = args[0]
	default:
		return "", errors.New("invalid arguments")
	}

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

		// Don't trust the server to have set HTTP status based on
		// If-Modified-Since; need to check Last-Modified ourselves.
		lm, err := time.Parse(time.RFC1123, response.Header.Get("Last-Modified"))
		if err != nil && !lm.After(stat.ModTime()) {
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

	if len(args) == 0 {
		// Destination filename wasn't specified; let the download's
		// Content-Disposition header (if present) take precedence
		// over the filename parsed from the URL.
		_, params, err := mime.ParseMediaType(response.Header.Get("Content-Disposition"))
		if err != nil {
			if fn, ok := params["filename"]; ok {
				dest = fn
			}
		}
	}

	// Replace destination with downloaded file
	err = AtomicFileReplace([]string{download.Name()}, []string{dest})
	if err != nil {
		return "", err
	}

	return dest, nil
}
