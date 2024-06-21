package node

import (
	"encoding/gob"
	"os"

	"github.com/montaguethomas/acd-go/internal/constants"
	"github.com/montaguethomas/acd-go/internal/log"
)

func (nt *Tree) loadCache() error {
	f, err := os.Open(nt.cacheFile)
	if err != nil {
		log.Debugf("error opening the cache file %q: %s", nt.cacheFile, constants.ErrLoadingCache)
		return constants.ErrLoadingCache
	}
	if err := gob.NewDecoder(f).Decode(nt); err != nil {
		log.Debugf("error decoding the cache file %q: %s", nt.cacheFile, err)
		return constants.ErrLoadingCache
	}
	log.Debugf("loaded NodeTree from cache file %q.", nt.cacheFile)
	nt.setClient(nt.Node)
	nt.buildNodeMap(nt.Node)

	return nil
}

func (nt *Tree) saveCache() error {
	f, err := os.Create(nt.cacheFile)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrCreateFile, nt.cacheFile)
		return constants.ErrCreateFile
	}
	if err := gob.NewEncoder(f).Encode(nt); err != nil {
		log.Errorf("%s: %s", constants.ErrGOBEncoding, err)
		return constants.ErrGOBEncoding
	}
	log.Debugf("saved NodeTree to cache file %q.", nt.cacheFile)
	return nil
}
