package client

import (
	"github.com/montaguethomas/acd-go/constants"
	"github.com/montaguethomas/acd-go/log"
	"github.com/montaguethomas/acd-go/node"
)

// List returns nodes.Nodes for all of the nodes underneath the path. It's up
// to the caller to differentiate between a file, a folder or an asset by using
// (*node.Node).IsFile(), (*node.Node).IsDir() and/or (*node.Node).IsAsset().
// A dir has sub-nodes accessible via (*node.Node).Nodes, you do not need to
// call this this function for every sub-node.
func (c *Client) List(path string) (node.Nodes, error) {
	rootNode, err := c.GetNodeTree().FindNode(path)
	if err != nil {
		return nil, err
	}
	if !rootNode.IsDir() {
		log.Errorf("%s: %s", constants.ErrPathIsNotFolder, path)
		return nil, constants.ErrPathIsNotFolder
	}

	return rootNode.Nodes, nil
}
