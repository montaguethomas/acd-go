package node

import (
	"encoding/json"
	"net/http"
	"slices"
	"time"

	"github.com/montaguethomas/acd-go/constants"
	"github.com/montaguethomas/acd-go/log"
)

type NodeKind string

const (
	KindFile   NodeKind = "FILE"
	KindFolder NodeKind = "FOLDER"
	KindAsset  NodeKind = "ASSET"
)

type NodeStatus string

const (
	StatusAvailable NodeStatus = "AVAILABLE"
	StatusTrash     NodeStatus = "TRASH"
	StatusPurged    NodeStatus = "PURGED"
)

type (
	// Nodes is a slice of nodes
	Nodes []*Node

	// ContentProperties hold the properties of the node.
	ContentProperties struct {
		// content version of the file (number)
		Version uint64 `json:"version,omitempty"`
		// md5 of a file content in HEX representation. (string)
		Extension string `json:"extension,omitempty"`
		// byte size (number, positive integer)
		Size uint64 `json:"size,omitempty"`
		// Media Type defined as per RFC 2046 (string)
		MD5 string `json:"md5,omitempty"`
		// file extension (not including the '.') (string)
		ContentType string `json:"contentType,omitempty"`
		// date extracted from media types (images and videos) (ISO8601 date with timezone offset)
		ContentDate time.Time `json:"contentDate,omitempty"`
	}

	Property map[string]string

	// Node represents a digital asset on the Amazon Cloud Drive, including files
	// and folders, in a parent-child relationship. A node contains only metadata
	// (e.g. folder) or it contains metadata and content (e.g. file).
	// https://developer.amazon.com/docs/amazon-drive/ad-restful-api-nodes.html
	Node struct {
		// Coming from Amazon
		// unique identifier of a file
		ID string `json:"id,omitempty"`
		// user friendly name of a file
		Name string `json:"name,omitempty"`
		// literal string "FILE", "FOLDER", "ASSET"
		Kind NodeKind `json:"kind,omitempty"`
		// metadata version of the file
		Version uint64 `json:"version,omitempty"`
		// Last modified date (ISO8601 date with timezone offset)
		ModifiedDate time.Time `json:"modifiedDate,omitempty"`
		// First uploaded date (ISO8601 date with timezone offset)
		CreatedDate time.Time `json:"createdDate,omitempty"`
		// List of Strings that are labeled to the file. Each label Max 256 characters. Max 10 labels.
		Labels []string `json:"labels,omitempty"`
		// short description of the file. Max 500 characters.
		Description string `json:"description,omitempty"`
		// Friendly name of Application Id which created the file
		CreatedBy string `json:"createdBy,omitempty"`
		// List of parent folder Ids
		Parents []string `json:"Parents,omitempty"`
		// either "AVAILABLE", "TRASH", "PURGED"
		Status NodeStatus `json:"status,omitempty"`
		// map of application properties {"owner_app_id1" : {"key":"value", "key2","value2"}, "owner_app_id2" : {"foo":"bar"} }
		Properties map[string]Property `json:"properties,omitempty"`

		// Files
		// Pre authenticated link enables viewing the file content for limited times only; has to be specifically requested
		TempLink          string            `json:"tempLink,omitempty"`
		ContentProperties ContentProperties `json:"contentProperties,omitempty"`

		// Folders
		// indicates whether the file is restricted to that app only or accessible to all the applications
		Restricted bool `json:"restricted,omitempty"`
		// indicates whether the folder is a root folder or not
		IsRoot bool `json:"isRoot,omitempty"`
		// set if node is shared
		IsShared bool `json:"isShared,omitempty"`

		// Internal
		Nodes  Nodes `json:"nodes,omitempty"`
		Root   bool  `json:"root,omitempty"`
		client client
	}

	newNode struct {
		Name       string              `json:"name,omitempty"`
		Kind       string              `json:"kind,omitempty"`
		Labels     []string            `json:"labels,omitempty"`
		Properties map[string]Property `json:"properties"`
		Parents    []string            `json:"parents"`
	}

	client interface {
		GetMetadataURL(string) string
		GetContentURL(string) string
		Do(*http.Request) (*http.Response, error)
		CheckResponse(*http.Response) error
		GetNodeTree() *Tree
	}
)

// Size returns the size of the node.
func (n *Node) Size() int64 {
	return int64(n.ContentProperties.Size)
}

// ModTime returns the last modified time of the node.
func (n *Node) ModTime() time.Time {
	return n.ModifiedDate
}

// IsFile returns whether the node represents a file.
func (n *Node) IsFile() bool {
	return n.Kind == KindFile
}

// IsDir returns whether the node represents a folder.
func (n *Node) IsDir() bool {
	return n.Kind == KindFolder
}

// IsAsset returns whether the node represents an asset.
func (n *Node) IsAsset() bool {
	return n.Kind == KindAsset
}

// Available returns true if the node is available
func (n *Node) Available() bool {
	return n.Status == StatusAvailable
}

// AddChild add a new child for the node
func (n *Node) AddChild(child *Node) {
	log.Debugf("adding %s under %s", child.Name, n.Name)
	n.Nodes = append(n.Nodes, child)
	child.client = n.client
}

// RemoveChild remove a new child for the node
func (n *Node) RemoveChild(child *Node) {
	found := false
	i := slices.Index(n.Nodes, child)
	if i >= 0 {
		found = true
		n.Nodes = slices.Delete(n.Nodes, i, i+1)
	}
	log.Debugf("removed %s from %s: %t", child.Name, n.Name, found)
}

func (n *Node) update(newNode *Node) error {
	// encode the newNode to JSON.
	v, err := json.Marshal(newNode)
	if err != nil {
		log.Errorf("error encoding the node to JSON: %s", err)
		return constants.ErrJSONEncoding
	}

	// decode it back to n
	if err := json.Unmarshal(v, n); err != nil {
		log.Errorf("error decoding the node from JSON: %s", err)
		return constants.ErrJSONDecoding
	}

	return nil
}
