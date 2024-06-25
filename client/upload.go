package client

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/montaguethomas/acd-go/constants"
	"github.com/montaguethomas/acd-go/log"
	"github.com/montaguethomas/acd-go/node"
)

// Upload uploads io.Reader to the path defined by the filename. It will create
// any non-existing folders.
func (c *Client) Upload(filename string, overwrite bool, labels []string, properties node.Property, r io.Reader) (*node.Node, error) {
	var (
		err        error
		logLevel   = log.GetLevel()
		fileNode   *node.Node
		parentNode *node.Node
	)

	parentNode, err = c.GetNodeTree().MkDirAll(path.Dir(filename))
	if err != nil {
		return nil, err
	}
	{
		log.SetLevel(log.DisableLogLevel)
		fileNode, err = c.GetNodeTree().FindNode(filename)
		log.SetLevel(logLevel)
	}
	if err == nil {
		if !overwrite {
			log.Errorf("%s: %s", constants.ErrFileExists, filename)
			return nil, constants.ErrFileExists
		}
		if err = c.GetNodeTree().Overwrite(fileNode, labels, properties, r); err != nil {
			return nil, err
		}

		return fileNode, nil
	}

	fileNode, err = c.GetNodeTree().Upload(parentNode, path.Base(filename), labels, properties, r)
	if err != nil {
		return nil, err
	}

	return fileNode, nil
}

// UploadFolder uploads an entire folder.
// If recursive is true, it will recurse through the entire filetree under
// localPath.  If overwrite is false and an existing file with the same md5 was
// found, an error will be returned.
func (c *Client) UploadFolder(localPath, remotePath string, recursive, overwrite bool, labels []string, properties node.Property) error {
	log.Debugf("uploading %q to %q", localPath, remotePath)
	if err := filepath.Walk(localPath, c.uploadFolderFunc(localPath, remotePath, recursive, overwrite, labels, properties)); err != nil {
		return err
	}

	return nil
}

func (c *Client) uploadFolderFunc(localPath, remoteBasePath string, recursive, overwrite bool, labels []string, properties node.Property) filepath.WalkFunc {
	return func(fpath string, info os.FileInfo, err error) error {
		var (
			logLevel   = log.GetLevel()
			fileNode   *node.Node
			remoteNode *node.Node
			f          *os.File
		)

		parts := strings.SplitAfter(fpath, localPath)
		remoteFilename := remoteBasePath + strings.Join(parts[1:], "/")
		remotePath := path.Dir(remoteFilename)
		log.Debugf("localPath %q remotePath %q fpath %q remoteFilename %q recursive %t overwrite %t",
			localPath, remotePath, fpath, remoteFilename, recursive, overwrite)

		// is this a folder?
		if info.IsDir() {
			log.Debugf("%q is a folder, skipping", fpath)
			return nil
		}
		// are we not recursive and trying to upload a file down the tree?
		if !recursive && localPath != path.Dir(fpath) {
			log.Debugf("%q is inside a sub-folder but we are not running recursively, skipping", fpath)
			return nil
		}

		log.Infof("uploading %q to %q", fpath, remoteFilename)
		if remoteNode, err = c.GetNodeTree().MkDirAll(remotePath); err != nil {
			return err
		}

		if f, err = os.Open(fpath); err != nil {
			log.Errorf("%s: %s", constants.ErrOpenFile, fpath)
			return constants.ErrOpenFile
		}
		defer f.Close()

		// does the file already exist?
		{
			log.SetLevel(log.DisableLogLevel)
			fileNode, err = c.GetNodeTree().FindNode(remoteFilename)
			log.SetLevel(logLevel)
		}
		if err == nil {
			if fileNode.IsDir() {
				log.Errorf("%s: remoteFilename %q", constants.ErrFileExistsAndIsFolder, remoteFilename)
				return constants.ErrFileExistsAndIsFolder
			}
			hash := md5.New()
			f.Seek(0, 0)
			io.Copy(hash, f)
			if hex.EncodeToString(hash.Sum(nil)) == fileNode.ContentProperties.MD5 {
				log.Debugf("%q already exists and has the same content, skipping", fpath)
				return nil
			}

			log.Debugf("%q already exists, overwrite is %t", fpath, overwrite)
			if !overwrite {
				log.Errorf("%s: remoteFilename %q", constants.ErrFileExistsWithDifferentContents, remoteFilename)
				return constants.ErrFileExistsWithDifferentContents
			}

			f.Seek(0, 0)
			return c.GetNodeTree().Overwrite(fileNode, labels, properties, f)
		}

		f.Seek(0, 0)
		if _, err := c.GetNodeTree().Upload(remoteNode, path.Base(fpath), labels, properties, f); err != nil && err != constants.ErrNoContentsToUpload {
			return err
		}

		return nil
	}
}
