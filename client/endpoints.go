package client

import (
	"encoding/json"
	"net/http"

	"github.com/montaguethomas/acd-go/constants"
	"github.com/montaguethomas/acd-go/log"
)

type apiEndpointResponse struct {
	ContentURL          string `json:"contentUrl"`
	CountryAtSignup     string `json:"countryAtSignup"`
	CustomerExists      bool   `json:"customerExists"`
	DownloadServiceURL  string `json:"downloadServiceUrl"`
	MetadataURL         string `json:"metadataUrl"`
	Region              string `json:"region"`
	RetailURL           string `json:"retailUrl"`
	ThumbnailServiceURL string `json:"thumbnailServiceUrl"`
}

// GetMetadataURL returns the metadata url.
func (c *Client) GetMetadataURL(path string) string {
	return c.endpoints.MetadataURL + path
}

// GetContentURL returns the content url.
func (c *Client) GetContentURL(path string) string {
	return c.endpoints.ContentURL + path
}

func (c *Client) setEndpoints() error {
	req, err := http.NewRequest("GET", constants.AmazonDriveEndpointURL, nil)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrCreatingHTTPRequest, err)
		return constants.ErrCreatingHTTPRequest
	}

	var response apiEndpointResponse
	res, err := c.Do(req)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrDoingHTTPRequest, err)
		return constants.ErrDoingHTTPRequest
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		log.Errorf("%s: %s", constants.ErrJSONDecodingResponseBody, err)
		return constants.ErrJSONDecodingResponseBody
	}

	log.Debugf("Endpoint Results: %+v", response)
	c.endpoints = response
	return nil
}
