package node

import (
	"fmt"
	"io"
	"net/http"

	"github.com/montaguethomas/acd-go/constants"
	"github.com/montaguethomas/acd-go/log"
)

// Download downloads the node and returns the body as io.ReadCloser or an
// error. The caller is responsible for closing the reader.
func (nt *Tree) Download(n *Node) (io.ReadCloser, error) {
	if n.IsDir() {
		log.Errorf("%s: cannot download a folder", constants.ErrPathIsFolder)
		return nil, constants.ErrPathIsFolder
	}
	url := nt.client.GetContentURL(fmt.Sprintf("nodes/%s/content", n.Id))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrCreatingHTTPRequest, err)
		return nil, constants.ErrCreatingHTTPRequest
	}
	res, err := nt.client.Do(req)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrDoingHTTPRequest, err)
		return nil, constants.ErrDoingHTTPRequest
	}
	if err := nt.client.CheckResponse(res); err != nil {
		return nil, err
	}

	return res.Body, nil
}
