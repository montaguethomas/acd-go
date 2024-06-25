package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/montaguethomas/acd-go/constants"
	"github.com/montaguethomas/acd-go/log"
)

// CreateFolder creates the named folder under the node
func (nt *Tree) CreateFolder(n *Node, name string, labels []string, properties Property) (*Node, error) {
	cn := &newNode{
		Name:    name,
		Kind:    "FOLDER",
		Labels:  labels,
		Parents: []string{n.Id},
		Properties: map[string]Property{
			constants.CloudDriveWebOwnerName: properties,
		},
	}
	jsonBytes, err := json.Marshal(cn)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrJSONEncoding, err)
		return nil, constants.ErrJSONEncoding
	}

	req, err := http.NewRequest("POST", nt.client.GetMetadataURL("nodes"), bytes.NewBuffer(jsonBytes))
	if err != nil {
		log.Errorf("%s: %s", constants.ErrCreatingHTTPRequest, err)
		return nil, constants.ErrCreatingHTTPRequest
	}

	req.Header.Set("Content-Type", "application/json")
	res, err := nt.client.Do(req)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrDoingHTTPRequest, err)
		return nil, constants.ErrDoingHTTPRequest
	}
	if err := nt.client.CheckResponse(res); err != nil {
		return nil, err
	}

	defer res.Body.Close()
	var node *Node
	if err := json.NewDecoder(res.Body).Decode(&node); err != nil {
		log.Errorf("%s: %s", constants.ErrJSONDecodingResponseBody, err)
		return nil, constants.ErrJSONDecodingResponseBody
	}

	nt.Lock()
	nt.nodeIdMap[node.Id] = node
	nt.Unlock()
	n.addChild(node)
	return node, nil
}

// Upload writes contents of r as name inside the current node.
func (nt *Tree) Upload(parent *Node, name string, labels []string, properties Property, r io.Reader) (*Node, error) {
	metadata := &newNode{
		Name:    name,
		Kind:    "FILE",
		Labels:  labels,
		Parents: []string{parent.Id},
		Properties: map[string]Property{
			constants.CloudDriveWebOwnerName: properties,
		},
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrJSONEncoding, err)
		return nil, constants.ErrJSONEncoding
	}

	postURL := nt.client.GetContentURL("nodes?suppress=deduplication")
	node, err := nt.upload(parent, postURL, "POST", string(metadataJSON), name, r)
	if err != nil {
		return nil, err
	}

	nt.Lock()
	nt.nodeIdMap[node.Id] = node
	nt.Unlock()
	parent.addChild(node)
	return node, nil
}

// Patch updates metadata for the provided node.
func (nt *Tree) Patch(n *Node, labels []string, properties Property) error {
	metadata := &newNode{
		Labels: labels,
		Properties: map[string]Property{
			constants.CloudDriveWebOwnerName: properties,
		},
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrJSONEncoding, err)
		return constants.ErrJSONEncoding
	}

	patchURL := nt.client.GetContentURL(fmt.Sprintf("nodes/%s", n.Id))
	req, err := http.NewRequest("PATCH", patchURL, bytes.NewBuffer(metadataJSON))
	if err != nil {
		log.Errorf("%s: %s", constants.ErrCreatingHTTPRequest, err)
		return constants.ErrCreatingHTTPRequest
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := nt.client.Do(req)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrDoingHTTPRequest, err)
		return constants.ErrDoingHTTPRequest
	}
	if err := nt.client.CheckResponse(res); err != nil {
		return err
	}

	defer res.Body.Close()
	var newNode *Node
	if err := json.NewDecoder(res.Body).Decode(&newNode); err != nil {
		log.Errorf("%s: %s", constants.ErrJSONDecodingResponseBody, err)
		return constants.ErrJSONDecodingResponseBody
	}

	return n.update(newNode)
}

// Overwrite writes contents of r as name inside the current node.
func (nt *Tree) Overwrite(n *Node, labels []string, properties Property, r io.Reader) error {
	putURL := nt.client.GetContentURL(fmt.Sprintf("nodes/%s/content", n.Id))
	node, err := nt.upload(n, putURL, "PUT", "", n.Name, r)
	if err != nil {
		return err
	}

	if err := n.update(node); err != nil {
		return err
	}
	return nt.Patch(n, labels, properties)
}

func (nt *Tree) upload(n *Node, url, method, metadataJSON, name string, r io.Reader) (*Node, error) {
	bodyReader, bodyWriter := io.Pipe()
	errChan := make(chan error)
	bodyChan := make(chan io.ReadCloser)
	contentTypeChan := make(chan string)

	go n.bodyWriter(metadataJSON, name, r, bodyWriter, errChan, contentTypeChan)
	go func() {
		req, err := http.NewRequest(method, url, bodyReader)
		if err != nil {
			log.Errorf("%s: %s", constants.ErrCreatingHTTPRequest, err)
			select {
			case errChan <- constants.ErrCreatingHTTPRequest:
			default:
			}
			return
		}
		req.Header.Add("Content-Type", <-contentTypeChan)
		res, err := nt.client.Do(req) // this should block until the upload is finished.
		if err != nil {
			log.Errorf("%s: %s", constants.ErrDoingHTTPRequest, err)
			select {
			case errChan <- constants.ErrDoingHTTPRequest:
			default:
			}
			return
		}
		if err := nt.client.CheckResponse(res); err != nil {
			select {
			case errChan <- err:
			default:
			}
			return
		}

		select {
		case bodyChan <- res.Body:
		default:
		}
	}()

	for {
		select {
		case err := <-errChan:
			if err != nil {
				return nil, err
			}
		case body := <-bodyChan:
			defer body.Close()
			var node Node
			if err := json.NewDecoder(body).Decode(&node); err != nil {
				log.Errorf("%s: %s", constants.ErrJSONDecodingResponseBody, err)
				return nil, constants.ErrJSONDecodingResponseBody
			}

			return &node, nil
		}
	}
}

func (n *Node) bodyWriter(metadataJSON, name string, r io.Reader, bodyWriter io.WriteCloser, errChan chan error, contentTypeChan chan string) {
	writer := multipart.NewWriter(bodyWriter)
	contentTypeChan <- writer.FormDataContentType()
	if metadataJSON != "" {
		if err := writer.WriteField("metadata", metadataJSON); err != nil {
			log.Errorf("%s: %s", constants.ErrWritingMetadata, err)
			select {
			case errChan <- constants.ErrWritingMetadata:
			default:
			}
			return
		}
	}

	part, err := writer.CreateFormFile("content", name)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrCreatingWriterFromFile, err)
		select {
		case errChan <- err:
		default:
		}
		return
	}
	count, err := io.Copy(part, r)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrWritingFileContents, err)
		select {
		case errChan <- constants.ErrWritingFileContents:
		default:
		}
		return
	}
	if count == 0 {
		select {
		case errChan <- constants.ErrNoContentsToUpload:
		default:
		}
		return
	}

	select {
	case errChan <- writer.Close():
	default:
	}
	select {
	case errChan <- bodyWriter.Close():
	default:
	}
}
