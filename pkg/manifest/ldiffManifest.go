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

func GetLdiffManifest(sophonPatchAPIManifest models.SophonPatchManifest) (*models.DiffManifest, error) {
	urlPrefix := sophonPatchAPIManifest.ManifestDownload.UrlPrefix
	urlSuffix := sophonPatchAPIManifest.ManifestDownload.UrlSuffix
	manifestID := sophonPatchAPIManifest.Manifest.ID
	manifestChecksum := sophonPatchAPIManifest.Manifest.Checksum

	isCompressed := sophonPatchAPIManifest.ManifestDownload.Compression != 0
	isEncrypted := sophonPatchAPIManifest.ManifestDownload.Encryption != 0

	if isEncrypted {
		return nil, fmt.Errorf("encrypted manifests are not supported")
	}

	url := buildManifestURL(urlPrefix, manifestID, urlSuffix)

	maxRetries := config.Config.MaxManifestDownloadRetries
	if maxRetries < 1 {
		maxRetries = 1
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		data, retryable, err := fetchManifestPayload(url, isCompressed, manifestChecksum, "patch manifest")
		if err != nil {
			if retryable && attempt < maxRetries {
				logging.GlobalLogger.Warn("Failed to fetch patch manifest, retrying... (attempt " + strconv.Itoa(attempt) + "): " + err.Error())
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			return nil, err
		}

		var manifest models.DiffManifest
		err = proto.Unmarshal(data, &manifest)
		if err != nil {
			return nil, fmt.Errorf("decode patch manifest: %w", err)
		}

		logging.GlobalLogger.Info("Patch manifest decoded successfully")
		return &manifest, nil
	}

	return nil, fmt.Errorf("failed to fetch patch manifest after retries")
}
