package models

import "time"

// JDownloaderConfig represents the JDownloader configuration
type JDownloaderConfig struct {
	ID                 int64      `json:"id"`
	ConnectionType     string     `json:"connectionType"` // 'local' or 'cloud'
	LocalHost          string     `json:"localHost"`
	LocalPort          int        `json:"localPort"`
	LocalPassword      string     `json:"localPassword,omitempty"`
	LocalUseAuth       bool       `json:"localUseAuth"`
	CloudEmail         string     `json:"cloudEmail,omitempty"`
	CloudPassword      string     `json:"cloudPassword,omitempty"`
	CloudDeviceID      string     `json:"cloudDeviceId,omitempty"`
	CloudDeviceName    string     `json:"cloudDeviceName,omitempty"`
	DefaultPackageName string     `json:"defaultPackageName"`
	AutoStartDownloads bool       `json:"autoStartDownloads"`
	Enabled            bool       `json:"enabled"`
	LastConnectedAt    *time.Time `json:"lastConnectedAt,omitempty"`
	ConnectionStatus   string     `json:"connectionStatus"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// JDownloaderHistory represents a record of links sent to JDownloader
type JDownloaderHistory struct {
	ID           int64      `json:"id"`
	VideoID      *int64     `json:"videoId,omitempty"`
	LinkURL      string     `json:"linkUrl"`
	PackageName  string     `json:"packageName,omitempty"`
	Status       string     `json:"status"` // 'queued', 'sent', 'failed'
	ErrorMessage string     `json:"errorMessage,omitempty"`
	SentAt       time.Time  `json:"sentAt"`
	VideoTitle   string     `json:"videoTitle,omitempty"` // Joined from videos table
}

// JDownloaderLinkRequest represents a request to add links to JDownloader
type JDownloaderLinkRequest struct {
	URL          string `json:"url"`
	PackageName  string `json:"packageName,omitempty"`
	DownloadPath string `json:"downloadPath,omitempty"`
	VideoID      *int64 `json:"videoId,omitempty"`
	AutoStart    bool   `json:"autoStart,omitempty"`
}

// JDownloaderSendLinksRequest represents a bulk request to send links
type JDownloaderSendLinksRequest struct {
	Links []JDownloaderLinkRequest `json:"links"`
}

// JDownloaderConnectionStatus represents the current connection status
type JDownloaderConnectionStatus struct {
	Connected      bool       `json:"connected"`
	Type           string     `json:"type"` // 'local' or 'cloud'
	DeviceName     string     `json:"deviceName,omitempty"`
	APIVersion     string     `json:"apiVersion,omitempty"`
	JDVersion      string     `json:"jdVersion,omitempty"`
	LastCheck      *time.Time `json:"lastCheck,omitempty"`
	ErrorMessage   string     `json:"errorMessage,omitempty"`
	AvailableSpace string     `json:"availableSpace,omitempty"`
}

// JDownloaderTestResult represents the result of a connection test
type JDownloaderTestResult struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	APIAvailable bool   `json:"apiAvailable"`
	Version      string `json:"version,omitempty"`
	ErrorDetails string `json:"errorDetails,omitempty"`
}

// JDownloaderDevice represents a device from MyJDownloader cloud
type JDownloaderDevice struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// JDownloaderAddLinkResponse represents the response from adding a link
type JDownloaderAddLinkResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	LinkID  string `json:"linkId,omitempty"`
}

// JDownloaderPackage represents a package in JDownloader
type JDownloaderPackage struct {
	UUID          string   `json:"uuid"`
	Name          string   `json:"name"`
	DownloadLinks int      `json:"downloadLinks"`
	FinishedDate  int64    `json:"finishedDate,omitempty"`
	Enabled       bool     `json:"enabled"`
	Status        string   `json:"status,omitempty"`
	BytesTotal    int64    `json:"bytesTotal,omitempty"`
	BytesLoaded   int64    `json:"bytesLoaded,omitempty"`
	Speed         int64    `json:"speed,omitempty"`
	Links         []string `json:"links,omitempty"`
}

// JDownloaderAPIRequest is the base structure for JDownloader local API requests
type JDownloaderAPIRequest struct {
	URL              string `json:"url,omitempty"`
	PackageName      string `json:"packageName,omitempty"`
	ForcePackageName bool   `json:"forcePackageName,omitempty"`
}

// JDownloaderAPIResponse is the response structure from JDownloader local API
type JDownloaderAPIResponse struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// LinkGrabberQueryLinksParams represents parameters for querying linkgrabber
type LinkGrabberQueryLinksParams struct {
	BytesTotal         bool `json:"bytesTotal,omitempty"`
	Comment            bool `json:"comment,omitempty"`
	Status             bool `json:"status,omitempty"`
	Enabled            bool `json:"enabled,omitempty"`
	MaxResults         int  `json:"maxResults,omitempty"`
	StartAt            int  `json:"startAt,omitempty"`
	PackageUUIDs       []string `json:"packageUUIDs,omitempty"`
	Host               bool `json:"host,omitempty"`
	Availability       bool `json:"availability,omitempty"`
	Url                bool `json:"url,omitempty"`
	Priority           bool `json:"priority,omitempty"`
	VariantIcon        bool `json:"variantIcon,omitempty"`
	VariantID          bool `json:"variantID,omitempty"`
	VariantName        bool `json:"variantName,omitempty"`
	Variants           bool `json:"variants,omitempty"`
	AddedDate          bool `json:"addedDate,omitempty"`
	FinishedDate       bool `json:"finishedDate,omitempty"`
	ExtractionStatus   bool `json:"extractionStatus,omitempty"`
	Running            bool `json:"running,omitempty"`
	Skipped            bool `json:"skipped,omitempty"`
	Speed              bool `json:"speed,omitempty"`
	Eta                bool `json:"eta,omitempty"`
}

// JDownloaderLinkInfo represents link information from linkgrabber
type JDownloaderLinkInfo struct {
	UUID         string `json:"uuid"`
	Name         string `json:"name"`
	PackageUUID  string `json:"packageUUID"`
	URL          string `json:"url,omitempty"`
	Host         string `json:"host,omitempty"`
	Status       string `json:"status,omitempty"`
	Availability string `json:"availability,omitempty"`
	BytesTotal   int64  `json:"bytesTotal,omitempty"`
	Enabled      bool   `json:"enabled"`
}
