package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/montaguethomas/acd-go/constants"
	"github.com/montaguethomas/acd-go/log"
)

type apiAuthTokenRequest struct {
	AppName            string `json:"app_name"`
	AppVersion         string `json:"app_version"`
	RequestedTokenType string `json:"requested_token_type"`
	SourceToken        string `json:"source_token"`
	SourceTokenType    string `json:"source_token_type"`
}

type apiAuthTokenResponse struct {
	AccessToken      string `json:"access_token"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ExpiresIn        int    `json:"expires_in"`
	RequestId        string `json:"request_id"`
	TokenType        string `json:"token_type"`
}

func (c *Client) RefreshToken() error {
	log.Debug("client.RefreshToken starting.")
	defer log.Debug("client.RefreshToken completed.")

	// Build Request Body
	c.config.mutex.RLock()
	request := &apiAuthTokenRequest{
		AppName:            c.config.AppName,
		AppVersion:         c.config.AppVersion,
		RequestedTokenType: "access_token",
		SourceToken:        c.config.RefreshToken,
		SourceTokenType:    "refresh_token",
	}
	c.config.mutex.RUnlock()
	requestJsonBytes, err := json.Marshal(request)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrJSONEncoding, err)
		return constants.ErrJSONEncoding
	}

	// Build Request
	req, err := http.NewRequest("POST", constants.AmazonAPITokenURL, bytes.NewBuffer(requestJsonBytes))
	if err != nil {
		log.Errorf("%s: %s", constants.ErrCreatingHTTPRequest, err)
		return constants.ErrCreatingHTTPRequest
	}
	req.Header.Set("Content-Type", "application/json")

	// Make Request
	res, err := c.Do(req)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrDoingHTTPRequest, err)
		return constants.ErrDoingHTTPRequest
	}
	if err := c.CheckResponse(res); err != nil {
		return err
	}

	// Handle Response
	defer res.Body.Close()
	response := apiAuthTokenResponse{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		log.Errorf("%s: %s", constants.ErrJSONDecodingResponseBody, err)
		return constants.ErrJSONDecodingResponseBody
	}

	log.Debugf("Refresh token response: %+v\n", response)

	if response.Error != "" || response.ErrorDescription != "" {
		return fmt.Errorf("Failed to refresh access token. Error: %s - %s", response.Error, response.ErrorDescription)
	}

	c.config.mutex.Lock()
	c.config.Headers["x-amz-access-token"] = response.AccessToken
	c.config.mutex.Unlock()
	return nil
}
