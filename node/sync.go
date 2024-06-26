package node

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/montaguethomas/acd-go/constants"
	"github.com/montaguethomas/acd-go/log"
)

type (
	// Request Body for fetch changes
	// https://developer.amazon.com/docs/amazon-drive/ad-restful-api-changes.html
	apiChangesRequest struct {
		// A token representing a frontier of updated items.
		Checkpoint string `json:"checkpoint,omitempty"`
		// The number of nodes to be returned within each Changes object in the response stream.
		ChunkSize int `json:"chunkSize,omitempty"`
		// The threshold of number of nodes returned at which the streaming call will be ended.
		// This is not intended to be used for strict pagination as the number of nodes returned
		// may exceed this number.
		MaxNodes int `json:"maxNodes,omitempty"`
		// If true then it will return the purged nodes as well. Default to false.
		IncludePurged string `json:"includePurged,omitempty"`
	}

	// Response Body of changes
	// https://developer.amazon.com/docs/amazon-drive/ad-restful-api-changes.html
	apiChangesResponse struct {
		Checkpoint string  `json:"checkpoint,omitempty"`
		Nodes      []*Node `json:"nodes,omitempty"`
		StatusCode int     `json:"statusCode,omitempty"`
		// If the response couldn't match a checkpoint and has sent all nodes.
		Reset bool `json:"reset,omitempty"`
		// Special ending value - client should check if received ending JSON to decide to resume or to finish.
		End bool `json:"end,omitempty"`
	}
)

// Sync syncs the tree with the server.
func (nt *Tree) Sync() error {
	log.Debug("node.Tree Sync starting.")
	defer log.Debug("node.Tree Sync completed.")

	// Build Request Body
	c := &apiChangesRequest{
		Checkpoint: nt.Checkpoint,
		ChunkSize:  nt.chunkSize,
	}
	jsonBytes, err := json.Marshal(c)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrJSONEncoding, err)
		return constants.ErrJSONEncoding
	}

	// Build Request
	req, err := http.NewRequest("POST", nt.client.GetMetadataURL("changes"), bytes.NewBuffer(jsonBytes))
	if err != nil {
		log.Errorf("%s: %s", constants.ErrCreatingHTTPRequest, err)
		return constants.ErrCreatingHTTPRequest
	}
	req.Header.Set("Content-Type", "application/json")

	// Make Request
	res, err := nt.client.Do(req)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrDoingHTTPRequest, err)
		return constants.ErrDoingHTTPRequest
	}
	if err := nt.client.CheckResponse(res); err != nil {
		return err
	}
	defer res.Body.Close()

	// Get time of response or current time
	lastUpdated, err := http.ParseTime(res.Header.Get("Date"))
	if err != nil {
		lastUpdated = time.Now().UTC()
	}

	// Process response body by line of json
	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		lineBytes := scanner.Bytes()

		var cr apiChangesResponse
		if err := json.Unmarshal(lineBytes, &cr); err != nil {
			log.Errorf("%s: %s", constants.ErrJSONDecodingResponseBody, err)
			return constants.ErrJSONDecodingResponseBody
		}

		// This should be the end of the stream of changes and next Scan() should return false.
		if cr.End {
			// Let's just be safe and break from the loop here.
			break
		}

		log.Debugf("syncing checkpoint %s", cr.Checkpoint)
		if err := nt.updateNodes(cr.Nodes); err != nil {
			return err
		}

		// Update checkpoint tracking
		nt.Lock()
		nt.Checkpoint = cr.Checkpoint
		nt.LastUpdated = lastUpdated
		nt.Unlock()
	}

	// Check for an error from the scanner
	if scanner.Err() != nil {
		log.Errorf("%s: %s", constants.ErrReadingResponseBody, err)
		return constants.ErrReadingResponseBody
	}

	// Rebuild the full node tree
	nt.buildNodeTree()

	// Save the cache after the updates
	if err := nt.saveCache(); err != nil {
		return err
	}

	return nil
}

func (nt *Tree) updateNodes(crNodes []*Node) error {
	for _, crNode := range crNodes {
		log.Debugf("node %s Id %s has changed.", crNode.Name, crNode.Id)

		// Handle root node -- it should never be deleted and won't have parents
		if crNode.IsRoot {
			nt.Lock()
			nt.Node = crNode
			nt.nodeIdMap[crNode.Id] = crNode
			nt.Unlock()
			continue
		}

		// Remove deleted nodes
		if !crNode.IsAvailable() {
			log.Debugf("node Id %s name %s has been deleted", crNode.Id, crNode.Name)
			nt.removeNodeFromTree(crNode)
			continue
		}

		// Get existing node or create it
		node, ok := nt.nodeIdMap[crNode.Id]
		if !ok {
			nt.Lock()
			nt.nodeIdMap[crNode.Id] = crNode
			nt.Unlock()
		}

		// If existing node was found
		if node != nil {
			// Set its Nodes on crNode
			crNode.Nodes = node.Nodes
			nt.Lock()
			nt.nodeIdMap[crNode.Id] = crNode
			nt.Unlock()

			// Remove it from all parents
			for _, parentId := range node.Parents {
				parent, ok := nt.nodeIdMap[parentId]
				if !ok {
					log.Debugf("parent Id %s not found, nothing to remove from", parentId)
					continue
				}
				parent.removeChild(node)
			}
		}

		// Add updated node to all parents
		for _, parentId := range crNode.Parents {
			parent, ok := nt.nodeIdMap[parentId]
			if !ok {
				log.Debugf("parent Id %s not found, creating placeholder", parentId)
				parent = &Node{Id: parentId}
				nt.nodeIdMap[parentId] = parent
			}
			parent.addChild(crNode)
		}
	}

	return nil
}
