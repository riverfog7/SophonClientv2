package models

type SophonManifestInfo struct {
	ID               string `json:"id"`
	Checksum         string `json:"checksum"`
	CompressedSize   int64  `json:"compressed_size,string"`
	UncompressedSize int64  `json:"uncompressed_size,string"`
}

type SophonChunkDownloadInfo struct {
	Encryption  int    `json:"encryption"`
	Password    string `json:"password"`
	Compression int    `json:"compression"`
	UrlPrefix   string `json:"url_prefix"`
	UrlSuffix   string `json:"url_suffix"`
}

type SophonManifestDownloadInfo struct {
	Encryption  int    `json:"encryption"`
	Password    string `json:"password"`
	Compression int    `json:"compression"`
	UrlPrefix   string `json:"url_prefix"`
	UrlSuffix   string `json:"url_suffix"`
}

type SophonManifestStats struct {
	CompressedSize   int64 `json:"compressed_size,string"`
	UncompressedSize int64 `json:"uncompressed_size,string"`
	FileCount        int   `json:"file_count,string"`
	ChunkCount       int   `json:"chunk_count,string"`
}

type SophonManifestDeduplicatedStats struct {
	CompressedSize   int64 `json:"compressed_size,string"`
	UncompressedSize int64 `json:"uncompressed_size,string"`
	FileCount        int   `json:"file_count,string"`
	ChunkCount       int   `json:"chunk_count,string"`
}

type SophonManifest struct {
	CategoryID        string                          `json:"category_id"`
	CategoryName      string                          `json:"category_name"`
	Manifest          SophonManifestInfo              `json:"manifest"`
	ChunkDownload     SophonChunkDownloadInfo         `json:"chunk_download"`
	ManifestDownload  SophonManifestDownloadInfo      `json:"manifest_download"`
	MatchingField     string                          `json:"matching_field"`
	Stats             SophonManifestStats             `json:"stats"`
	DeduplicatedStats SophonManifestDeduplicatedStats `json:"deduplicated_stats"`
}

type SophonGetBuildAPIData struct {
	BuildID   string           `json:"build_id"`
	Tag       string           `json:"tag"`
	Manifests []SophonManifest `json:"manifests"`
}

type SophonGetBuildAPIResponse struct {
	Retcode int                   `json:"retcode"`
	Message string                `json:"message"`
	Data    SophonGetBuildAPIData `json:"data"`
}
