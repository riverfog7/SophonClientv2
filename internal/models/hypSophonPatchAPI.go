package models

type SophonDiffManifestInfo struct {
	ID               string `json:"id"`
	Checksum         string `json:"checksum"`
	CompressedSize   int64  `json:"compressed_size,string"`
	UncompressedSize int64  `json:"uncompressed_size,string"`
}

type SophonDiffDownloadInfo struct {
	Encryption  int    `json:"encryption"`
	Password    string `json:"password"`
	Compression int    `json:"compression"`
	UrlPrefix   string `json:"url_prefix"`
	UrlSuffix   string `json:"url_suffix"`
}

type SophonDiffManifestDownloadInfo struct {
	Encryption  int    `json:"encryption"`
	Password    string `json:"password"`
	Compression int    `json:"compression"`
	UrlPrefix   string `json:"url_prefix"`
	UrlSuffix   string `json:"url_suffix"`
}

type SophonDiffManifestStat struct {
	CompressedSize   int64 `json:"compressed_size,string"`
	UncompressedSize int64 `json:"uncompressed_size,string"`
	FileCount        int   `json:"file_count,string"`
	ChunkCount       int   `json:"chunk_count,string"`
}

type SophonDiffManifestStats map[string]SophonDiffManifestStat

type SophonPatchManifest struct {
	CategoryID       string                         `json:"category_id"`
	CategoryName     string                         `json:"category_name"`
	Manifest         SophonDiffManifestInfo         `json:"manifest"`
	DiffDownload     SophonDiffDownloadInfo         `json:"chunk_download"`
	ManifestDownload SophonDiffManifestDownloadInfo `json:"manifest_download"`
	MatchingField    string                         `json:"matching_field"`
	Stats            SophonDiffManifestStats        `json:"stats"`
}

type SophonGetPatchBuildAPIData struct {
	BuildID   string           `json:"build_id"`
	PatchID   string           `json:"patch_id"`
	Tag       string           `json:"tag"`
	Manifests []SophonManifest `json:"manifests"`
}

type SophonGetPatchBuildAPIResponse struct {
	Retcode int                   `json:"retcode"`
	Message string                `json:"message"`
	Data    SophonGetBuildAPIData `json:"data"`
}
