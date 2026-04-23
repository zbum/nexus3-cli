package registry

import "time"

// Asset matches Nexus REST v1 AssetXO.
type Asset struct {
	ID           string            `json:"id"`
	Path         string            `json:"path"`
	DownloadUrl  string            `json:"downloadUrl"`
	Repository   string            `json:"repository"`
	Format       string            `json:"format"`
	ContentType  string            `json:"contentType"`
	Checksum     map[string]string `json:"checksum"`
	LastModified time.Time         `json:"lastModified"`
	BlobCreated  time.Time         `json:"blobCreated"`
	FileSize     int64             `json:"fileSize"`
}

// Component matches Nexus REST v1 ComponentXO. For Docker repos:
// Name == image name, Version == tag.
type Component struct {
	ID         string  `json:"id"`
	Repository string  `json:"repository"`
	Format     string  `json:"format"`
	Group      string  `json:"group"`
	Name       string  `json:"name"`
	Version    string  `json:"version"`
	Assets     []Asset `json:"assets"`
}

type pageComponent struct {
	Items             []Component `json:"items"`
	ContinuationToken string      `json:"continuationToken"`
}

// Repository matches Nexus REST v1 RepositoryXO (short form).
type Repository struct {
	Name   string `json:"name"`
	Format string `json:"format"`
	Type   string `json:"type"`
	URL    string `json:"url"`
}
