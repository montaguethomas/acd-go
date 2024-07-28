package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/montaguethomas/acd-go/constants"
	"github.com/montaguethomas/acd-go/log"
	"github.com/montaguethomas/acd-go/node"
)

// apiGetTrashResponse is the response body for listing trash
type apiGetTrashResponse struct {
	Count     uint64       `json:"count,omitempty"`
	NextToken string       `json:"nextToken,omitempty"`
	Nodes     []*node.Node `json:"data,omitempty"`
}

type apiBulkPurgeRequest struct {
	Recurse bool     `json:"recurse"`
	NodeIds []string `json:"nodeIds"`
}

type apiBulkPurgeResponse struct {
	ErrorMap map[string]int `json:"errorMap"`
}

// GetTrash will get all the nodes in the trash
func (c *Client) GetTrash() ([]*node.Node, error) {
	log.Debug("client.GetTrash starting.")
	defer log.Debug("client.GetTrash completed.")

	// Get nodes in the trash
	var nextToken string
	var nodes []*node.Node
	for {
		urlStr := c.GetMetadataURL("trash")
		u, err := url.Parse(urlStr)
		if err != nil {
			log.Errorf("%s: %s", constants.ErrParsingURL, urlStr)
			return nil, constants.ErrParsingURL
		}

		v := url.Values{}
		v.Set("limit", "200")
		if nextToken != "" {
			v.Set("startToken", nextToken)
		}
		u.RawQuery = v.Encode()

		// Make Request
		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			log.Errorf("%s: %s", constants.ErrCreatingHTTPRequest, err)
			return nil, constants.ErrCreatingHTTPRequest
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := c.Do(req)
		if err != nil {
			log.Errorf("%s: %s", constants.ErrDoingHTTPRequest, err)
			return nil, constants.ErrDoingHTTPRequest
		}

		// Handle Response
		defer res.Body.Close()
		response := apiGetTrashResponse{}
		if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
			log.Errorf("%s: %s", constants.ErrJSONDecodingResponseBody, err)
			return nil, constants.ErrJSONDecodingResponseBody
		}

		nextToken = response.NextToken
		nodes = append(nodes, response.Nodes...)

		if nextToken == "" {
			break
		}
	}
	return nodes, nil
}

// PurgeNodes will purge the provided nodes
func (c *Client) PurgeNodes(nodes []*node.Node) error {
	log.Debug("client.PurgeNodes starting.")
	defer log.Debug("client.PurgeNodes completed.")

	// Build node ids list
	nodeIds := []string{}
	for _, node := range nodes {
		nodeIds = append(nodeIds, node.Id)
	}

	// Build Request Body
	request := &apiBulkPurgeRequest{
		Recurse: true,
		NodeIds: nodeIds,
	}
	requestJsonBytes, err := json.Marshal(request)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrJSONEncoding, err)
		return constants.ErrJSONEncoding
	}

	// Build Request
	req, err := http.NewRequest("POST", c.GetMetadataURL("bulk/nodes/purge"), bytes.NewBuffer(requestJsonBytes))
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
	response := apiBulkPurgeResponse{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		log.Errorf("%s: %s", constants.ErrJSONDecodingResponseBody, err)
		return constants.ErrJSONDecodingResponseBody
	}

	// Check for any errors
	if len(response.ErrorMap) > 0 {
		return fmt.Errorf("Purge Node Errors: %+v", response.ErrorMap)
	}

	return nil
}

// PurgeTrash will purge all nodes in the trash
func (c *Client) PurgeTrash() error {
	log.Debug("client.PurgeTrash starting.")
	defer log.Debug("client.PurgeTrash completed.")

	nodes, err := c.GetTrash()
	if err != nil {
		return err
	}
	return c.PurgeNodes(nodes)
}
