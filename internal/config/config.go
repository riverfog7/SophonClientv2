package config

type SophonClientConfig struct {
	MaxManifestDownloadRetries int
	MaxChunkDownloadRetries    int
	CocurrentDownloads         int
	CocurrentDecompressions    int
	CocurrentHashchecks        int
}

var Config SophonClientConfig = SophonClientConfig{
	MaxManifestDownloadRetries: 5,
	MaxChunkDownloadRetries:    5,
	CocurrentDownloads:         16,
	CocurrentDecompressions:    4,
	CocurrentHashchecks:        8,
}
