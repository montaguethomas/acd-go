package client

import "github.com/montaguethomas/acd-go/node"

// FetchNodeTree fetches and caches the nodeTree.
func (c *Client) FetchNodeTree() error {
	nt, err := node.NewTree(c, c.cacheFile)
	if err != nil {
		return err
	}
	c.nodeTree = nt
	return nil
}

// GetNodeTree returns the nodeTree.
func (c *Client) GetNodeTree() *node.Tree {
	return c.nodeTree
}
