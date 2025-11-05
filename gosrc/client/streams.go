package client

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/goccy/go-json"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/client/poststreams"
	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/events"
	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/models"
)

type PostStreamStruct struct {
	SkipIdentify   bool
	ExpectedSha256 string
}

// Writes the content from the reader to the data store and returns its metadata.
func (c *Client) PostStream(source string, label events.DatastreamLabel, reader io.Reader, query *PostStreamStruct) (*events.BinaryEntityDatastream, error) {
	target := fmt.Sprintf("%s/api/v3/stream/%s/%s", c.dataUrl, source, label)
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if query.SkipIdentify {
		q.Set(poststreams.SkipIdentify, fmt.Sprint(query.SkipIdentify))
	}
	if len(query.ExpectedSha256) > 0 {
		q.Set(poststreams.ExpectedSha256, query.ExpectedSha256)
	}
	u.RawQuery = q.Encode()
	req, err := http.NewRequest("POST", u.String(), reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, statusError(resp)
	}
	decoder := json.NewDecoder(resp.Body)
	var d models.DataResponse
	err = decoder.Decode(&d)
	if err != nil {
		return nil, err
	}
	if len(d.Errors) > 0 {
		return nil, errors.New(d.Errors[0].Detail)
	}
	// overwrite label, as dispatcher doesn't take a parameter for label
	d.Data.Label = label
	return &d.Data, nil
}

// Writes the content from the reader to the data store and returns a Binary record.
func (c *Client) PostStreamContent(source string, reader io.Reader) (*events.BinaryEntity, error) {
	data, err := c.PostStream(source, events.DataLabelContent, reader, &PostStreamStruct{})
	if err != nil {
		return nil, err
	}
	return data.ToInputEntity(), nil
}

// DownloadBinaryChunk gets the file with the sha256 provided in the range of start to end bytes from dispatcher.
// bytes are loaded into the buffer provided to allow for the buffer to be reused.
func (c *Client) DownloadBinaryChunk(source string, label events.DatastreamLabel, hash string, start uint64, end uint64) ([]byte, error) {
	target := fmt.Sprintf("%s/api/v3/stream/%s/%s/%s", c.dataUrl, source, label, hash)
	var buf []byte
	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	buf, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 404 {
		// content gone
		return nil, nil
	} else if resp.StatusCode != 200 && resp.StatusCode != 206 {
		return nil, statusError(resp)
	}
	return buf, nil
}

// DownloadBinary downloads the binary from dispatcher.
func (c *Client) DownloadBinary(source string, label events.DatastreamLabel, hash string) (*bufio.Reader, error) {
	target := fmt.Sprintf("%s/api/v3/stream/%s/%s/%s", c.dataUrl, source, label, hash)
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		// error during connection to server
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		// error during connection to server
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		// bad return code
		return nil, statusError(resp)
	}
	// copy response body into memory (for connection reuse)
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)
	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return nil, err
	}
	return bufio.NewReader(&b), nil
}

// DeleteBinary deletes the binary from dispatcher.
func (c *Client) DeleteBinary(source string, label events.DatastreamLabel, hash string, ifOlderThan *time.Time) (bool, error) {
	target := fmt.Sprintf("%s/api/v3/stream/%s/%s/%s", c.dataUrl, source, label, hash)
	u, err := url.Parse(target)
	if err != nil {
		return false, err
	}
	q := u.Query()
	if ifOlderThan != nil {
		q.Set("ifOlderThan", fmt.Sprintf("%v", ifOlderThan.Unix()))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		// error during connection to server
		return false, err
	}
	resp, err := c.do(req)
	if err != nil {
		// error during connection to server
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return false, nil
	}
	if resp.StatusCode != 200 {
		// bad return code
		return false, statusError(resp)
	}
	parsed := make(map[string]bool)
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&parsed)
	if err != nil {
		return false, err
	}
	return parsed["deleted"], nil
}

// Exists checks if the binary exists in dispatcher.
func (c *Client) Exists(source string, label events.DatastreamLabel, hash string) (bool, error) {
	target := fmt.Sprintf("%s/api/v3/stream/%s/%s/%s", c.dataUrl, source, label, hash)
	u, err := url.Parse(target)
	if err != nil {
		return false, err
	}
	q := u.Query()
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("HEAD", u.String(), nil)
	if err != nil {
		// error during connection to server
		return false, err
	}
	resp, err := c.do(req)
	if err != nil {
		// error during connection to server
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return false, nil
	} else if resp.StatusCode == 200 {
		return true, nil
	} else {
		// bad return code
		return false, statusError(resp)
	}
}

// GetMetadata fetches the metadata for the binary.
func (c *Client) GetMetadata(source string, label events.DatastreamLabel, hash string) (events.BinaryEntityDatastream, error) {
	return events.BinaryEntityDatastream{}, errors.New("not implemented")
}
