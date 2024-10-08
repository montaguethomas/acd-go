package node

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"maps"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/montaguethomas/acd-go/constants"
	"github.com/montaguethomas/acd-go/log"
)

const (
	// NodePropertyKeysMaxCount is the maximum allowed node property keys
	NodePropertyKeysMaxCount = 10
	// NodePropertyKeyMaxSize is the maximum size of a node property key
	NodePropertyKeyMaxSize = 50
	// NodePropertyKeyCheckRegex is the matching pattern for node property keys
	NodePropertyKeyCheckRegex = "^[a-zA-Z0-9_]*$"
	// NodePropertyValueMaxSize is the maximum size of a node property key's value
	NodePropertyValueMaxSize = 500
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
	Nodes map[string]*Node

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

		//Image
		//Video
	}

	// Node represents a digital asset on the Amazon Cloud Drive, including files
	// and folders, in a parent-child relationship. A node contains only metadata
	// (e.g. folder) or it contains metadata and content (e.g. file).
	// https://developer.amazon.com/docs/amazon-drive/ad-restful-api-nodes.html
	Node struct {
		// Coming from Amazon
		// etag of node
		ETagResponse string `json:"eTagResponse,omitempty"`
		// unique identifier of a file
		Id string `json:"id,omitempty"`
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
		// Extra properties which client wants to add to a node. Properties will be grouped together by the owner application Id
		// which created them. By default, all properties will be restricted to its owner and no one else can read/write/delete
		// them. As of now, only 10 properties can be stored by each owner. This is how properties would look inside a Node:
		// {"owner_app_id1" : {"key":"value", "key2","value2"}, "owner_app_id2" : {"foo":"bar"}, "owner_app_id3": { "key":"value", "key":"value", ...} }
		Properties map[string]*nodeProperty `json:"properties,omitempty"`
		// indicates whether the file is restricted to that app only or accessible to all the applications
		Restricted bool `json:"restricted,omitempty"`
		// indicates whether the folder is a root folder or not
		IsRoot bool `json:"isRoot,omitempty"`
		// set if node is shared
		IsShared bool `json:"isShared,omitempty"`

		// Files Only
		// Pre authenticated link enables viewing the file content for limited times only; has to be specifically requested
		TempLink          string            `json:"tempLink,omitempty"`
		ContentProperties ContentProperties `json:"contentProperties,omitempty"`

		// Internal - exported in order to support gob encode/decode
		Nodes Nodes `json:"nodes,omitempty"`

		// Internal
		mutex sync.RWMutex
	}

	// Request Body for creating new nodes (files, folders)
	newNode struct {
		Name       string              `json:"name"`
		Kind       string              `json:"kind"`
		Labels     []string            `json:"labels,omitempty"`
		Parents    []string            `json:"parents,omitempty"`
		Properties map[string]Property `json:"properties,omitempty"`
	}

	// Request Body for patching nodes (files, folders)
	patchNode struct {
		Name       string              `json:"name,omitempty"`
		Labels     []string            `json:"labels,omitempty"`
		Properties map[string]Property `json:"properties,omitempty"`
	}
)

func New() *Node {
	node := &Node{}
	node.SetOwnerProperties(NewProperty())
	return node
}

func NewProperty() Property {
	p := &nodeProperty{}
	p.props = make(map[string]string, NodePropertyKeysMaxCount)
	return p
}

// Property implements the Node Properties field per the API.
// https://developer.amazon.com/docs/amazon-drive/ad-restful-api-nodes.html#properties-1
type Property interface {
	Clone() Property
	Get(key string) (string, bool)
	GetAll() map[string]string
	Has(key string) bool
	Remove(key string)
	RemoveAll(keys []string)
	Set(key, value string) error
	SetAll(props map[string]string) []error
	Size() int
	GobEncode() ([]byte, error)
	GobDecode(data []byte) error
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(data []byte) error
}

type nodeProperty struct {
	props map[string]string
}

// Clone returns a shallow copy of the property.
func (p *nodeProperty) Clone() Property {
	return &nodeProperty{
		props: maps.Clone(p.props),
	}
}

// Get returns the value of the property key.
func (p *nodeProperty) Get(key string) (string, bool) {
	v, ok := p.props[key]
	return v, ok
}

// GetAll returns all property key/value pairs.
func (p *nodeProperty) GetAll() map[string]string {
	return maps.Clone(p.props)
}

// Has checks if a property key is set.
func (p *nodeProperty) Has(key string) bool {
	_, ok := p.props[key]
	return ok
}

// Remove will remove the property key.
func (p *nodeProperty) Remove(key string) {
	delete(p.props, key)
}

// RemoveAll will remove all the property keys it is provided.
func (p *nodeProperty) RemoveAll(keys []string) {
	for _, key := range keys {
		p.Remove(key)
	}
}

// Set will add/update the property key/value.
func (p *nodeProperty) Set(key, value string) error {
	keyCheck, err := regexp.MatchString(NodePropertyKeyCheckRegex, key)
	if err != nil {
		return err
	}
	if !keyCheck || len(key) > NodePropertyKeyMaxSize {
		return constants.ErrNodePropertyInvalidKey
	}
	if !p.Has(key) && p.Size() == NodePropertyKeysMaxCount {
		return constants.ErrNodePropertyMaxKeys
	}
	if len(value) > NodePropertyValueMaxSize {
		return constants.ErrNodePropertyInvalidValue
	}
	if p.props == nil {
		p.props = map[string]string{}
	}
	p.props[key] = value
	return nil
}

// SetAll will add/update all the property key/value pairs it is provided.
func (p *nodeProperty) SetAll(props map[string]string) []error {
	errors := []error{}
	for key, value := range props {
		err := p.Set(key, value)
		if err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

// Size returns the number of property keys set.
func (p *nodeProperty) Size() int {
	return len(p.props)
}

// GobEncode implements the gob.GobEncoder interface.
func (p *nodeProperty) GobEncode() ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	if err := gob.NewEncoder(buf).Encode(p.props); err != nil {
		return []byte{}, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements the gob.GobDecoder interface.
func (p *nodeProperty) GobDecode(data []byte) error {
	buf := bytes.NewBuffer(data)
	return gob.NewDecoder(buf).Decode(&p.props)
}

// MarshalJSON returns json string of the Property
func (p *nodeProperty) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.props)
}

// UnmarshalJSON tries to populate the Property from the json data
func (p *nodeProperty) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &p.props); err != nil {
		return err
	}
	return nil
}

func (n *Node) Lock() {
	n.mutex.Lock()
}
func (n *Node) Unlock() {
	n.mutex.Unlock()
}
func (n *Node) RLock() {
	n.mutex.RLock()
}
func (n *Node) RUnlock() {
	n.mutex.RUnlock()
}

func (n *Node) Count() uint64 {
	n.RLock()
	defer n.RUnlock()

	if !n.IsDir() {
		return uint64(1)
	}

	// Sum count of all children
	var total uint64
	for _, child := range n.Nodes {
		total += child.Count()
	}
	return total
}

// Size returns the size of the node.
func (n *Node) Size() uint64 {
	n.RLock()
	defer n.RUnlock()

	if !n.IsDir() {
		return n.ContentProperties.Size
	}

	// Sum size of all children
	var total uint64
	for _, child := range n.Nodes {
		total += child.Size()
	}
	return total
}

// ModTime returns the last modified time of the node.
func (n *Node) ModTime() time.Time {
	n.RLock()
	defer n.RUnlock()
	return n.ModifiedDate
}

// IsFile returns whether the node represents a file.
func (n *Node) IsFile() bool {
	n.RLock()
	defer n.RUnlock()
	return n.Kind == KindFile
}

// IsDir returns whether the node represents a folder.
func (n *Node) IsDir() bool {
	n.RLock()
	defer n.RUnlock()
	return n.Kind == KindFolder
}

// IsAsset returns whether the node represents an asset.
func (n *Node) IsAsset() bool {
	n.RLock()
	defer n.RUnlock()
	return n.Kind == KindAsset
}

// IsAvailable returns true if the node is available
func (n *Node) IsAvailable() bool {
	n.RLock()
	defer n.RUnlock()
	return n.Status == StatusAvailable
}

func (n *Node) GetOwnerProperties() (Property, bool) {
	n.RLock()
	defer n.RUnlock()

	props, ok := n.Properties[constants.AMZClientOwnerName]
	return props, ok
}

func (n *Node) GetOwnerProperty(key string) (string, bool) {
	n.RLock()
	defer n.RUnlock()

	props, ok := n.Properties[constants.AMZClientOwnerName]
	if !ok {
		return "", false
	}
	value, ok := props.Get(key)
	if !ok {
		return "", false
	}
	return value, true
}

func (n *Node) SetOwnerProperties(prop Property) {
	n.Lock()
	defer n.Unlock()

	if n.Properties == nil {
		n.Properties = map[string]*nodeProperty{}
	}
	n.Properties[constants.AMZClientOwnerName] = prop.(*nodeProperty)
}

// addChild add a new child for the node
func (n *Node) addChild(child *Node) {
	n.Lock()
	defer n.Unlock()

	log.Tracef("adding child node %s under %s", child.Name, n.Name)
	if n.Nodes == nil {
		n.Nodes = make(Nodes)
	}
	n.Nodes[strings.ToLower(child.Name)] = child
}

// removeChild remove a new child for the node
func (n *Node) removeChild(child *Node) {
	n.Lock()
	defer n.Unlock()

	log.Tracef("removing child node %s from %s", child.Name, n.Name)
	if n.Nodes != nil {
		delete(n.Nodes, strings.ToLower(child.Name))
	}
}

func (n *Node) update(newNode *Node) error {
	// encode the newNode to JSON.
	newNode.RLock()
	v, err := json.Marshal(newNode)
	newNode.RUnlock()
	if err != nil {
		log.Errorf("error encoding the node to JSON: %s", err)
		return constants.ErrJSONEncoding
	}

	// decode it back to n
	n.Lock()
	defer n.Unlock()
	if err := json.Unmarshal(v, n); err != nil {
		log.Errorf("error decoding the node from JSON: %s", err)
		return constants.ErrJSONDecoding
	}

	return nil
}
