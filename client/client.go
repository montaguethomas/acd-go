package client

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/montaguethomas/acd-go/constants"
	"github.com/montaguethomas/acd-go/log"
	"github.com/montaguethomas/acd-go/node"
)

// Config represents the clients configuration.
type Config struct {
	// Cookies contains all cookies to pass on all requests made.
	// These will be used for authentication to the API endpoints.
	Cookies map[string]string `json:"cookies"`

	// CacheFile represents the file used by the client to cache the NodeTree.
	// This file is not assumed to be present and will be created on the first
	// run. It is gob-encoded node.Node.
	CacheFile string `json:"cacheFile"`

	// PurgeTrashInterval is how often to purge trash
	PurgeTrashInterval string `json:"purgeTrashInterval"`

	// SyncChunkSize is the number of nodes to be returned within each Changes
	// object in the response stream.
	SyncChunkSize int `json:"syncChunkSize"`

	// SyncInterval is how often to sync the Node Tree cache
	SyncInterval string `json:"syncInterval"`

	// Timeout configures the HTTP Client with a timeout after which the client
	// will cancel the request and return. A timeout of 0 means no timeout.
	// See http://godoc.org/net/http#Client for more information.
	Timeout string `json:"timeout"`

	// UserAgent is the value to use for the user agent header on all http requests
	UserAgent string `json:"userAgent"`
}

// Client provides a client for Amazon Cloud Drive.
type Client struct {
	// nodeTree is the tree of nodes as stored on the drive.
	nodeTree *node.Tree

	config         *Config
	httpClient     *http.Client
	cacheFile      string
	endpoints      EndpointResponse
	purgeTrashDone chan struct{}
}

type EndpointResponse struct {
	ContentURL          string `json:"contentUrl"`
	CountryAtSignup     string `json:"countryAtSignup"`
	CustomerExists      bool   `json:"customerExists"`
	DownloadServiceURL  string `json:"downloadServiceUrl"`
	MetadataURL         string `json:"metadataUrl"`
	Region              string `json:"region"`
	RetailURL           string `json:"retailUrl"`
	ThumbnailServiceURL string `json:"thumbnailServiceUrl"`
}

// New returns a new Amazon Cloud Drive "acd" Client
func New(config *Config) (*Client, error) {
	// Validate configs
	if config.CacheFile == "" {
		return nil, constants.ErrCacheFileConfigEmpty
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

	// Initialize client
	if err := c.setEndpoints(); err != nil {
		return nil, err
	}

	// Setup background trash purging
	if config.PurgeTrashInterval != "" {
		purgeTrashInterval, err := time.ParseDuration(config.PurgeTrashInterval)
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
	return c.nodeTree.Close()
}

// Do invokes net/http.Client.Do(). Refer to net/http.Client.Do() for documentation.
func (c *Client) Do(r *http.Request) (*http.Response, error) {
	for name, value := range c.config.Cookies {
		r.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	if value, ok := c.config.Cookies["session-id"]; ok {
		r.Header.Add("x-amzn-sessionid", value)
	}
	if c.config.UserAgent != "" {
		r.Header.Add("user-agent", c.config.UserAgent)
	}
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
