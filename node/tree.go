package node

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/montaguethomas/acd-go/constants"
	"github.com/montaguethomas/acd-go/log"
)

type (
	// Tree represents a node tree.
	Tree struct {
		// Exported in order to support gob encode/decode
		*Node
		LastUpdated time.Time
		Checkpoint  string

		// Internal
		cacheFile string
		chunkSize int
		client    client
		mutex     sync.RWMutex
		nodeIdMap map[string]*Node
		syncDone  chan struct{}
	}

	// Amazon Cloud Drive Client interface
	client interface {
		GetMetadataURL(string) string
		GetContentURL(string) string
		Do(*http.Request) (*http.Response, error)
		CheckResponse(*http.Response) error
		GetNodeTree() *Tree
	}
)

// NewTree returns the root node (the head of the tree).
func NewTree(c client, cacheFile string, chunkSize int, syncInterval time.Duration) (*Tree, error) {
	nt := &Tree{
		cacheFile: cacheFile,
		client:    c,
		chunkSize: chunkSize,
		nodeIdMap: make(map[string]*Node),
	}

	// Load data cache and sync
	if err := nt.loadCache(); err != nil {
		log.Debug(err)
	}
	if err := nt.Sync(); err != nil {
		log.Errorf("initial sync failed %s", err)
		return nil, err
	}

	ticker := time.NewTicker(syncInterval)
	nt.syncDone = make(chan struct{}, 1)
	go func() {
		for {
			select {
			case <-nt.syncDone:
				ticker.Stop()
				return
			case <-ticker.C:
				log.Debug("Background sync starting.")
				if err := nt.Sync(); err != nil {
					log.Errorf("Background sync error: %s", err)
				}
				log.Debug("Background sync completed.")
			}
		}
	}()

	return nt, nil
}

// Close finalizes the NodeTree
func (nt *Tree) Close() error {
	nt.syncDone <- struct{}{}
	return nt.saveCache()
}

func (nt *Tree) Lock() {
	nt.mutex.Lock()
}
func (nt *Tree) Unlock() {
	nt.mutex.Unlock()
}
func (nt *Tree) RLock() {
	nt.mutex.RLock()
}
func (nt *Tree) RUnlock() {
	nt.mutex.RUnlock()
}

// RemoveNode removes this node from the server and from the NodeTree.
func (nt *Tree) RemoveNode(n *Node) error {
	putURL := nt.client.GetMetadataURL(fmt.Sprintf("/trash/%s", n.Id))
	req, err := http.NewRequest("PUT", putURL, nil)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrCreatingHTTPRequest, err)
		return constants.ErrCreatingHTTPRequest
	}
	res, err := nt.client.Do(req)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrDoingHTTPRequest, err)
		return constants.ErrDoingHTTPRequest
	}
	if err := nt.client.CheckResponse(res); err != nil {
		return err
	}

	nt.removeNodeFromTree(n)
	return nil
}

func (nt *Tree) addNodeToNodeIdMap(n *Node) {
	nt.Lock()
	n.RLock()
	nt.nodeIdMap[n.Id] = n
	n.RUnlock()
	nt.Unlock()
}

func (nt *Tree) removeNodeFromTree(n *Node) {
	n.RLock()
	defer n.RUnlock()

	nt.RLock()
	for _, parentId := range n.Parents {
		parent, ok := nt.nodeIdMap[parentId]
		if !ok {
			log.Tracef("node.Tree removeNodeFromTree parent Id %s not found", parentId)
			continue
		}
		parent.removeChild(n)
	}
	nt.RUnlock()
	nt.Lock()
	delete(nt.nodeIdMap, n.Id)
	nt.Unlock()
}

// MkdirAll creates a directory named path, along with any necessary parents,
// and returns the directory node and nil, or else returns an error. If path is
// already a directory, MkDirAll does nothing and returns the directory node
// and nil.
func (nt *Tree) MkDirAll(path string) (*Node, error) {
	var (
		err        error
		folderNode = nt.Node
		logLevel   = log.GetLevel()
		nextNode   *Node
		node       *Node
	)

	// Short-circuit if the node already exists!
	{
		log.SetLevel(log.DisableLogLevel)
		node, err = nt.FindNode(path)
		log.SetLevel(logLevel)
	}
	if err == nil {
		if node.IsDir() {
			return node, err
		}
		log.Errorf("%s: %s", constants.ErrFileExistsAndIsNotFolder, path)
		return nil, constants.ErrFileExistsAndIsNotFolder
	}

	// chop off any leading or trailing slashes.
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		log.Errorf("%s: %s", constants.ErrCannotCreateRootNode, path)
		return nil, constants.ErrCannotCreateRootNode
	}

	for i, part := range parts {
		{
			log.SetLevel(log.DisableLogLevel)
			nextNode, err = nt.FindNode(strings.Join(parts[:i+1], "/"))
			log.SetLevel(logLevel)
		}
		if err != nil && err != constants.ErrNodeNotFound {
			return nil, err
		}
		if err == constants.ErrNodeNotFound {
			nextNode, err = nt.CreateFolder(folderNode, part, []string{}, NewProperty())
			if err != nil {
				return nil, err
			}
		}

		if !nextNode.IsDir() {
			log.Errorf("%s: %s", constants.ErrCannotCreateANodeUnderAFile, strings.Join(parts[:i+1], "/"))
			return nil, constants.ErrCannotCreateANodeUnderAFile
		}

		folderNode = nextNode
	}

	return folderNode, nil
}

func (nt *Tree) buildNodeIdMap(current *Node) {
	if nt.Node == current {
		nt.Lock()
		nt.nodeIdMap = make(map[string]*Node)
		nt.Unlock()
	}
	nt.Lock()
	nt.nodeIdMap[current.Id] = current
	nt.Unlock()
	for _, node := range current.Nodes {
		nt.buildNodeIdMap(node)
	}
}

func (nt *Tree) buildNodeTree() {
	log.Debug("node.Tree buildNodeTree starting.")
	defer log.Debug("node.Tree buildNodeTree completed.")

	for _, node := range nt.nodeIdMap {
		if node.IsRoot {
			nt.Lock()
			nt.Node = node
			nt.Unlock()
		}
		for _, parentId := range node.Parents {
			if parent, ok := nt.nodeIdMap[parentId]; ok {
				parent.addChild(node)
			}
		}
	}
}
