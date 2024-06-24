package node

import (
	"regexp"
	"strings"

	"github.com/montaguethomas/acd-go/constants"
	"github.com/montaguethomas/acd-go/log"
)

// FindNode finds a node for a particular path.
// TODO(kalbasit): This does not perform well, this should be cached in a map
// path->node and calculated on load (fresh, cache, refresh).
func (nt *Tree) FindNode(path string) (*Node, error) {
	// replace multiple n*/ with /
	re := regexp.MustCompile("/[/]*")
	path = string(re.ReplaceAll([]byte(path), []byte("/")))
	// chop off any leading or trailing slashes.
	path = strings.Trim(path, "/")
	// did we ask for the root node?
	if path == "" {
		return nt.Node, nil
	}
	// lowercase path
	path = strings.ToLower(path)

	// initialize our search from the root node
	node := nt.Node

	// iterate over the path parts until we find the path (or not).
	parts := strings.Split(path, "/")
	for _, part := range parts {
		var ok bool
		node, ok = node.Nodes[part]
		if !ok {
			log.Errorf("%s: %s", constants.ErrNodeNotFound, path)
			return nil, constants.ErrNodeNotFound
		}
	}

	return node, nil
}

// FindById returns the node identified by the Id.
func (nt *Tree) FindById(id string) (*Node, error) {
	n, ok := nt.nodeIdMap[id]
	if !ok {
		log.Errorf("%s: Id %q", constants.ErrNodeNotFound, id)
		return nil, constants.ErrNodeNotFound
	}
	return n, nil
}
