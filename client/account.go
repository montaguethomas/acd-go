package client

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/montaguethomas/acd-go/constants"
	"github.com/montaguethomas/acd-go/log"
)

type (
	// AccountInfo represents information about an Amazon Cloud Drive account.
	AccountInfo struct {
		TermsOfUse string `json:"termsOfUse"`
		Status     string `json:"status"`
	}

	// AccountQuota represents information about the account quotas.
	AccountQuota struct {
		Quota          uint64    `json:"quota"`
		LastCalculated time.Time `json:"lastCalculated"`
		Available      uint64    `json:"available"`
	}

	// AccountUsage represents information about the account usage.
	AccountUsage struct {
		LastCalculated time.Time `json:"lastCalculated"`

		Doc struct {
			Billable struct {
				Bytes uint64 `json:"bytes"`
				Count uint64 `json:"count"`
			} `json:"billable"`
			Total struct {
				Bytes uint64 `json:"bytes"`
				Count uint64 `json:"count"`
			} `json:"total"`
		} `json:"doc"`

		Other struct {
			Billable struct {
				Bytes uint64 `json:"bytes"`
				Count uint64 `json:"count"`
			} `json:"billable"`
			Total struct {
				Bytes uint64 `json:"bytes"`
				Count uint64 `json:"count"`
			} `json:"total"`
		} `json:"other"`

		Photo struct {
			Billable struct {
				Bytes uint64 `json:"bytes"`
				Count uint64 `json:"count"`
			} `json:"billable"`
			Total struct {
				Bytes uint64 `json:"bytes"`
				Count uint64 `json:"count"`
			} `json:"total"`
		} `json:"photo"`

		Video struct {
			Billable struct {
				Bytes uint64 `json:"bytes"`
				Count uint64 `json:"count"`
			} `json:"billable"`
			Total struct {
				Bytes uint64 `json:"bytes"`
				Count uint64 `json:"count"`
			} `json:"total"`
		} `json:"video"`
	}
)

func (au *AccountUsage) Billable() (bytes uint64, count uint64) {
	bytes += au.Doc.Billable.Bytes
	bytes += au.Other.Billable.Bytes
	bytes += au.Photo.Billable.Bytes
	bytes += au.Video.Billable.Bytes
	count += au.Other.Billable.Count
	count += au.Doc.Billable.Count
	count += au.Photo.Billable.Count
	count += au.Video.Billable.Count
	return
}

func (au *AccountUsage) Total() (bytes uint64, count uint64) {
	bytes += au.Doc.Total.Bytes
	bytes += au.Other.Total.Bytes
	bytes += au.Photo.Total.Bytes
	bytes += au.Video.Total.Bytes
	count += au.Other.Total.Count
	count += au.Doc.Total.Count
	count += au.Photo.Total.Count
	count += au.Video.Total.Count
	return
}

// GetAccountInfo returns AccountInfo about the current account.
func (c *Client) GetAccountInfo() (*AccountInfo, error) {
	var ai AccountInfo
	req, err := http.NewRequest("GET", c.GetMetadataURL("account/info"), nil)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrCreatingHTTPRequest, err)
		return nil, constants.ErrCreatingHTTPRequest
	}

	res, err := c.Do(req)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrDoingHTTPRequest, err)
		return nil, constants.ErrDoingHTTPRequest
	}

	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&ai); err != nil {
		log.Errorf("%s: %s", constants.ErrJSONDecodingResponseBody, err)
		return nil, constants.ErrJSONDecodingResponseBody
	}

	return &ai, nil
}

// GetAccountQuota returns AccountQuota about the current account.
func (c *Client) GetAccountQuota() (*AccountQuota, error) {
	var aq AccountQuota
	req, err := http.NewRequest("GET", c.GetMetadataURL("account/quota"), nil)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrCreatingHTTPRequest, err)
		return nil, constants.ErrCreatingHTTPRequest
	}

	res, err := c.Do(req)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrDoingHTTPRequest, err)
		return nil, constants.ErrDoingHTTPRequest
	}

	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&aq); err != nil {
		log.Errorf("%s: %s", constants.ErrJSONDecodingResponseBody, err)
		return nil, constants.ErrJSONDecodingResponseBody
	}

	return &aq, nil
}

// GetAccountUsage returns AccountUsage about the current account.
func (c *Client) GetAccountUsage() (*AccountUsage, error) {
	var au AccountUsage
	req, err := http.NewRequest("GET", c.GetMetadataURL("account/usage"), nil)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrCreatingHTTPRequest, err)
		return nil, constants.ErrCreatingHTTPRequest
	}

	res, err := c.Do(req)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrDoingHTTPRequest, err)
		return nil, constants.ErrDoingHTTPRequest
	}

	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&au); err != nil {
		log.Errorf("%s: %s", constants.ErrJSONDecodingResponseBody, err)
		return nil, constants.ErrJSONDecodingResponseBody
	}

	return &au, nil
}
