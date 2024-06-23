package client

import (
	"encoding/json"
	"net/http"

	"github.com/montaguethomas/acd-go/constants"
	"github.com/montaguethomas/acd-go/log"
)

// GetMetadataURL returns the metadata url.
func (c *Client) GetMetadataURL(path string) string {
	return c.endpoints.MetadataURL + path
}

// GetContentURL returns the content url.
func (c *Client) GetContentURL(path string) string {
	return c.endpoints.ContentURL + path
}

func (c *Client) setEndpoints() error {
	req, err := http.NewRequest("GET", EndpointURL, nil)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrCreatingHTTPRequest, err)
		return constants.ErrCreatingHTTPRequest
	}

	var er EndpointResponse
	res, err := c.Do(req)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrDoingHTTPRequest, err)
		return constants.ErrDoingHTTPRequest
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&er); err != nil {
		log.Errorf("%s: %s", constants.ErrJSONDecodingResponseBody, err)
		return constants.ErrJSONDecodingResponseBody
	}

	log.Debugf("Endpoint Results: %+v", er)
	c.endpoints = er
	return nil
}
