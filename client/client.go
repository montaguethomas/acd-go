package client

import (
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/montaguethomas/acd-go/constants"
	"github.com/montaguethomas/acd-go/log"
	"github.com/montaguethomas/acd-go/node"
)

// Client provides a client for Amazon Cloud Drive.
type Client struct {
	// nodeTree is the tree of nodes as stored on the drive.
	nodeTree *node.Tree

	config           *Config
	httpClient       *http.Client
	cacheFile        string
	endpoints        apiEndpointResponse
	purgeTrashDone   chan struct{}
	refreshTokenDone chan struct{}
}

// New returns a new Amazon Cloud Drive "acd" Client
func New(config *Config) (*Client, error) {
	// Validate configs
	if config.CacheFile == "" {
		return nil, constants.ErrCacheFileConfigEmpty
	}
	if config.AppName == "" {
		config.AppName = runtime.Version()
	}
	if config.AppVersion == "" {
		config.AppVersion = runtime.Version()
	}
	if config.Headers == nil {
		config.Headers = map[string]string{}
	}
	if config.SyncChunkSize < 1 {
		config.SyncChunkSize = 25
	}
	if config.SyncInterval == "" {
		config.SyncInterval = "30s"
	}
	if config.Timeout == "" {
		config.Timeout = "0"
	}
	//if config.UserAgent == "" {
	//	config.UserAgent = "CloudDriveMac/10.4.0.3655d303"
	//}

	// Create client
	timeout, err := time.ParseDuration(config.Timeout)
	if err != nil {
		return nil, err
	}
	c := &Client{
		config:    config,
		cacheFile: config.CacheFile,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}

	// If a refresh token is set, try to get a new access token and setup background refresh
	if c.config.RefreshToken != "" {
		if err := c.RefreshToken(); err != nil {
			return nil, err
		}
		ticker := time.NewTicker(time.Minute * 15)
		c.refreshTokenDone = make(chan struct{}, 1)
		go func() {
			for {
				select {
				case <-c.refreshTokenDone:
					ticker.Stop()
					return
				case <-ticker.C:
					log.Debug("Background refresh token starting.")
					if err := c.RefreshToken(); err != nil {
						log.Errorf("Background refresh token error: %s", err)
					}
					log.Debug("Background refresh token completed.")
				}
			}
		}()
	}

	// Load endpoints
	if err := c.setEndpoints(); err != nil {
		return nil, err
	}

	// Setup background trash purging
	c.config.mutex.RLock()
	if c.config.PurgeTrashInterval != "" {
		purgeTrashInterval, err := time.ParseDuration(c.config.PurgeTrashInterval)
		if err != nil {
			return nil, err
		}
		ticker := time.NewTicker(purgeTrashInterval)
		c.purgeTrashDone = make(chan struct{}, 1)
		go func() {
			for {
				select {
				case <-c.purgeTrashDone:
					ticker.Stop()
					return
				case <-ticker.C:
					log.Debug("Background purge trash starting.")
					if err := c.PurgeTrash(); err != nil {
						log.Errorf("Background purge trash error: %s", err)
					}
					log.Debug("Background purge trash completed.")
				}
			}
		}()
	}
	c.config.mutex.RUnlock()

	// Build NodeTree
	syncInterval, err := time.ParseDuration(config.SyncInterval)
	if err != nil {
		return nil, err
	}
	nt, err := node.NewTree(c, c.cacheFile, config.SyncChunkSize, syncInterval)
	if err != nil {
		return nil, err
	}
	c.nodeTree = nt

	return c, nil
}

// Close finalizes the acd.
func (c *Client) Close() error {
	if c.config.PurgeTrashInterval != "" {
		c.purgeTrashDone <- struct{}{}
	}
	if c.config.RefreshToken != "" {
		c.refreshTokenDone <- struct{}{}
	}
	return c.nodeTree.Close()
}

// Do invokes net/http.Client.Do(). Refer to net/http.Client.Do() for documentation.
func (c *Client) Do(r *http.Request) (*http.Response, error) {
	c.config.mutex.RLock()
	for key, value := range c.config.Headers {
		r.Header.Set(key, value)
	}
	if c.config.UserAgent != "" {
		r.Header.Set("User-Agent", c.config.UserAgent)
	}
	c.config.mutex.RUnlock()
	return c.httpClient.Do(r)
}

func LoadConfig(configFile string) (*Config, error) {
	// validate the config file
	if err := validateFile(configFile, false); err != nil {
		return nil, err
	}

	cf, err := os.Open(configFile)
	if err != nil {
		log.Errorf("%s: %s", constants.ErrOpenFile, err)
		return nil, err
	}
	defer cf.Close()
	var config Config
	if err := json.NewDecoder(cf).Decode(&config); err != nil {
		log.Errorf("%s: %s", constants.ErrJSONDecoding, err)
		return nil, err
	}

	return &config, nil
}

func validateFile(file string, checkPerms bool) error {
	stat, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			log.Errorf("%s: %s -- %s", constants.ErrFileNotFound, err, file)
			return constants.ErrFileNotFound
		}
		log.Errorf("%s: %s -- %s", constants.ErrStatFile, err, file)
		return constants.ErrStatFile
	}
	if checkPerms && stat.Mode() != os.FileMode(0600) {
		log.Errorf("%s: want 0600 got %s", constants.ErrWrongPermissions, stat.Mode())
		return constants.ErrWrongPermissions
	}

	return nil
}
