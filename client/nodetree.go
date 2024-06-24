package client

import "github.com/montaguethomas/acd-go/node"

// GetNodeTree returns the nodeTree.
func (c *Client) GetNodeTree() *node.Tree {
	return c.nodeTree
}
