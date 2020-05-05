package utils

import (
	"strings"
	"os"
	"net/http"
	"io"
)

// Attachment is a media attachment on a message in the format <content-type>:<url>. Content type may be a full
// media type or may omit the subtype when it is unknown.
//
// Examples:
//  - image/jpeg:http://s3.amazon.com/bucket/test.jpg
//  - image:http://s3.amazon.com/bucket/test.jpg
//
type Attachment string

// ToParts splits an attachment string into content-type and URL
func (a Attachment) ToParts() (string, string) {
	offset := strings.Index(string(a), ":")
	if offset >= 0 {
		return string(a[:offset]), string(a[offset+1:])
	}
	return "", string(a)
}

// ContentType returns the MIME type of this attachment
func (a Attachment) ContentType() string {
	contentType, _ := a.ToParts()
	return contentType
}

// URL returns the full URL of this attachment
func (a Attachment) URL() string {
	_, url := a.ToParts()
	return url
}

// DownloadFile will download a url and store it in local filepath.
// It writes to the destination file as it downloads it, without
// loading the entire file into memory.
func (a Attachment) DownloadFile() (error, string) {
	// Create the file
	urlParts := strings.Split(a.URL(), "/")
	filename := urlParts[len(urlParts) - 1]
	filepath := "/tmp/" + filename
	out, err := os.Create(filepath)
	if err != nil {
		return err, ""
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(a.URL())
	if err != nil {
		return err, ""
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err, ""
	}

	return nil, filepath
}
