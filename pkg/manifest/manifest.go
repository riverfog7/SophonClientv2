package manifest

import (
	"SophonClientv2/internal/config"
	"SophonClientv2/internal/logging"
	"SophonClientv2/internal/models"
	"fmt"
	"strconv"
	"time"

	"google.golang.org/protobuf/proto"
)

func GetManifest(sophonBuildAPIManifest models.SophonManifest) (*models.Manifest, error) {
	urlPrefix := sophonBuildAPIManifest.ManifestDownload.UrlPrefix
	urlSuffix := sophonBuildAPIManifest.ManifestDownload.UrlSuffix
	manifestID := sophonBuildAPIManifest.Manifest.ID
	manifestChecksum := sophonBuildAPIManifest.Manifest.Checksum

	isCompressed := sophonBuildAPIManifest.ManifestDownload.Compression != 0
	isEncrypted := sophonBuildAPIManifest.ManifestDownload.Encryption != 0

	if isEncrypted {
		return nil, fmt.Errorf("encrypted manifests are not supported")
	}

	url := buildManifestURL(urlPrefix, manifestID, urlSuffix)

	maxRetries := config.Config.MaxManifestDownloadRetries
	if maxRetries < 1 {
		maxRetries = 1
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		data, retryable, err := fetchManifestPayload(url, isCompressed, manifestChecksum, "manifest")
		if err != nil {
			if retryable && attempt < maxRetries {
				logging.GlobalLogger.Warn("Failed to fetch manifest, retrying... (attempt " + strconv.Itoa(attempt) + "): " + err.Error())
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			return nil, err
		}

		var manifest models.Manifest
		err = proto.Unmarshal(data, &manifest)
		if err != nil {
			return nil, fmt.Errorf("decode manifest: %w", err)
		}

		logging.GlobalLogger.Info("Manifest decoded successfully")
		return &manifest, nil
	}

	return nil, fmt.Errorf("failed to fetch manifest after retries")
}
