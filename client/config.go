package client

import "sync"

// Config represents the clients configuration.
type Config struct {
	// AppName is the name of the application to send to Amazon
	AppName string `json:"appName"`

	// AppVersion is the version of the application to send to Amazon
	AppVersion string `json:"appVersion"`

	// CacheFile represents the file used by the client to cache the NodeTree.
	// This file is not assumed to be present and will be created on the first
	// run. It is gob-encoded node.Node.
	CacheFile string `json:"cacheFile"`

	// Headers contains all the additional headers to pass on all requests made.
	Headers map[string]string `json:"headers"`

	// PurgeTrashInterval is how often to purge trash
	PurgeTrashInterval string `json:"purgeTrashInterval"`

	// RefreshToken is an Amazon API Refresh Token
	// https://developer.amazon.com/docs/login-with-amazon/refresh-token.html
	RefreshToken string `json:"refreshToken"`

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

	// Internal
	mutex sync.RWMutex
}
