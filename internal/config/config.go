package config

type SophonClientConfig struct {
	MaxManifestDownloadRetries int
	MaxChunkDownloadRetries    int
	CocurrentDownloads         int
}

var Config SophonClientConfig = SophonClientConfig{
	MaxManifestDownloadRetries: 5,
	MaxChunkDownloadRetries:    5,
	CocurrentDownloads:         16,
}
