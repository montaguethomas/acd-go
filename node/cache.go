package node

import (
	"encoding/gob"
	"os"

	"github.com/montaguethomas/acd-go/constants"
	"github.com/montaguethomas/acd-go/log"
)

func (nt *Tree) loadCache() error {
	log.Debug("node.Tree loadCache starting.")
	defer log.Debug("node.Tree loadCache completed.")

	f, err := os.Open(nt.cacheFile)
	if err != nil {
		log.Debugf("error opening the cache file %q: %s", nt.cacheFile, constants.ErrLoadingCache)
		return constants.ErrLoadingCache
	}
	nt.Lock()
	// using defer here causes a deadlock with nt.buildNodeIdMap()
	if err := gob.NewDecoder(f).Decode(nt); err != nil {
		nt.Unlock()
		log.Debugf("error decoding the cache file %q: %s", nt.cacheFile, err)
		return constants.ErrLoadingCache
	}
	nt.Unlock()
	log.Debugf("loaded NodeTree from cache file %q.", nt.cacheFile)
	nt.buildNodeIdMap(nt.Node)
	return nil
}

func (nt *Tree) saveCache() error {
	log.Debug("node.Tree saveCache starting.")
	defer log.Debug("node.Tree saveCache completed.")

	f, err := os.Create(nt.cacheFile)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrCreateFile, nt.cacheFile)
		return constants.ErrCreateFile
	}
	nt.Lock()
	defer nt.Unlock()
	if err := gob.NewEncoder(f).Encode(nt); err != nil {
		log.Errorf("%s: %s", constants.ErrGOBEncoding, err)
		return constants.ErrGOBEncoding
	}
	log.Debugf("saved NodeTree to cache file %q.", nt.cacheFile)
	return nil
}
