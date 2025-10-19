package manifest

import (
	"SophonClientv2/internal/config"
	"SophonClientv2/internal/logging"
	"SophonClientv2/internal/models"
	"crypto/md5"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/klauspost/compress/zstd"
	"google.golang.org/protobuf/proto"
)

func GetManifest(sophonBuildAPIManifest models.SophonManifest) *models.Manifest {
	var url string
	urlPrefix := sophonBuildAPIManifest.ManifestDownload.UrlPrefix
	urlSuffix := sophonBuildAPIManifest.ManifestDownload.UrlSuffix
	manifestID := sophonBuildAPIManifest.Manifest.ID
	manifestChecksum := sophonBuildAPIManifest.Manifest.Checksum

	isCompressed := sophonBuildAPIManifest.ManifestDownload.Compression != 0
	isEncrypted := sophonBuildAPIManifest.ManifestDownload.Encryption != 0

	if isEncrypted {
		logging.GlobalLogger.Fatal("Encrypted manifests are not supported")
	}

	if urlSuffix != "" {
		url = urlPrefix + "/" + manifestID + "/" + urlSuffix
	} else {
		url = urlPrefix + "/" + manifestID
	}

	// Retry only on MD5 hash mismatch or network errors
	maxRetries := config.Config.MaxManifestDownloadRetries
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// HTTP GET
		resp, err := http.Get(url)
		if err != nil {
			if attempt < maxRetries {
				logging.GlobalLogger.Warn("Failed to fetch manifest, retrying... (attempt " + strconv.Itoa(attempt) + ")")
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			logging.GlobalLogger.Fatal("Failed to fetch manifest: " + err.Error())
		}
		defer resp.Body.Close()
		logging.GlobalLogger.Info("Fetched manifest successfully with status: " + resp.Status)

		var data []byte
		if isCompressed {
			// Streaming decompression
			dec, err := zstd.NewReader(resp.Body)
			if err != nil {
				logging.GlobalLogger.Fatal("Failed to create zstd streaming reader: " + err.Error())
			}
			defer dec.Close()

			data, err = io.ReadAll(dec)
			if err != nil {
				logging.GlobalLogger.Fatal("Failed to read decompressed manifest: " + err.Error())
			}
		} else {
			data, err = io.ReadAll(resp.Body)
			if err != nil {
				logging.GlobalLogger.Fatal("Failed to read manifest response: " + err.Error())
			}
		}

		// Check MD5 hash
		if manifestChecksum != "" {
			hash := md5.Sum(data)
			computedHash := hex.EncodeToString(hash[:])
			if computedHash != manifestChecksum {
				if attempt < maxRetries {
					logging.GlobalLogger.Warn("Manifest hash mismatch, retrying... (attempt " + strconv.Itoa(attempt) + ")")
					time.Sleep(time.Duration(attempt) * time.Second)
					continue
				}
				logging.GlobalLogger.Fatal("Manifest hash mismatch after retries")
			}
		}

		var manifest models.Manifest
		err = proto.Unmarshal(data, &manifest)
		if err != nil {
			logging.GlobalLogger.Fatal("Failed to decode manifest: " + err.Error())
		}

		logging.GlobalLogger.Info("Manifest decoded successfully")
		return &manifest
	}

	logging.GlobalLogger.Fatal("Failed to fetch manifest after retries")
	return nil
}
