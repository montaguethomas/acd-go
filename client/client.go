package client

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/montaguethomas/acd-go/internal/constants"
	"github.com/montaguethomas/acd-go/internal/log"
	"github.com/montaguethomas/acd-go/node"
)

const EndpointURL = "https://drive.amazonaws.com/drive/v1/account/endpoint"

// Config represents the clients configuration.
type Config struct {
	// Cookies contains all cookies to pass on all requests made.
	// These will be used for authentication to the API endpoints.
	Cookies map[string]string `json:"cookies"`

	// CacheFile represents the file used by the client to cache the NodeTree.
	// This file is not assumed to be present and will be created on the first
	// run. It is gob-encoded node.Node.
	CacheFile string `json:"cacheFile"`

	// Timeout configures the HTTP Client with a timeout after which the client
	// will cancel the request and return. A timeout of 0 (the default) means
	// no timeout. See http://godoc.org/net/http#Client for more information.
	Timeout time.Duration `json:"timeout"`
}

// Client provides a client for Amazon Cloud Drive.
type Client struct {
	// NodeTree is the tree of nodes as stored on the drive. This tree should
	// be fetched using (*Client).FetchNodeTree() as soon the client is
	// created.
	NodeTree *node.Tree

	config      *Config
	httpClient  *http.Client
	cacheFile   string
	metadataURL string
	contentURL  string
	endpoints   EndpointResponse
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

type SharedCookieJar struct {
	cookies []*http.Cookie
}

func (j *SharedCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.cookies = append(j.cookies, cookies...)
}

func (j *SharedCookieJar) Cookies(u *url.URL) (cookies []*http.Cookie) {
	return j.cookies
}

// New returns a new Amazon Cloud Drive "acd" Client. configFile must exist and must be a valid JSON decodable into Config.
func New(configFile string) (*Client, error) {
	config, err := loadConfig(configFile)
	if err != nil {
		return nil, err
	}

	jar := &SharedCookieJar{}
	cookies := []*http.Cookie{}
	for name, value := range config.Cookies {
		cookies = append(cookies, &http.Cookie{Name: name, Value: value})
	}
	jar.SetCookies(nil, cookies)

	c := &Client{
		config:    config,
		cacheFile: config.CacheFile,
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: config.Timeout,
		},
	}
	if err := setEndpoints(c); err != nil {
		return nil, err
	}

	return c, nil
}

// Close finalizes the acd.
func (c *Client) Close() error {
	return c.NodeTree.Close()
}

// Do invokes net/http.Client.Do(). Refer to net/http.Client.Do() for documentation.
func (c *Client) Do(r *http.Request) (*http.Response, error) {
	return c.httpClient.Do(r)
}

func loadConfig(configFile string) (*Config, error) {
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
