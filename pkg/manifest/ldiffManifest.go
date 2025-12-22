package manifest

import (
	"SophonClientv2/internal/config"
	"SophonClientv2/internal/logging"
	"SophonClientv2/internal/models"
	"crypto/md5"
	"encoding/hex"
	"hash"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/klauspost/compress/zstd"
	"google.golang.org/protobuf/proto"
)

func GetLdiffManifest(sophonPatchAPIManifest models.SophonPatchManifest) *models.DiffManifest {
	var url string
	urlPrefix := sophonPatchAPIManifest.ManifestDownload.UrlPrefix
	urlSuffix := sophonPatchAPIManifest.ManifestDownload.UrlSuffix
	manifestID := sophonPatchAPIManifest.Manifest.ID
	manifestChecksum := sophonPatchAPIManifest.Manifest.Checksum

	isCompressed := sophonPatchAPIManifest.ManifestDownload.Compression != 0
	isEncrypted := sophonPatchAPIManifest.ManifestDownload.Encryption != 0

	if isEncrypted {
		logging.GlobalLogger.Fatal("Encrypted manifests are not supported")
	}

	if urlSuffix != "" {
		url = urlPrefix + "/" + manifestID + "/" + urlSuffix
	} else {
		url = urlPrefix + "/" + manifestID
	}

	maxRetries := config.Config.MaxManifestDownloadRetries
	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := http.Get(url)
		if err != nil {
			if attempt < maxRetries {
				logging.GlobalLogger.Warn("Failed to fetch manifest, retrying... (attempt " + strconv.Itoa(attempt) + ")")
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			logging.GlobalLogger.Fatal("Failed to fetch patch manifest: " + err.Error())
		}
		defer resp.Body.Close()
		logging.GlobalLogger.Info("Fetched patch manifest successfully with status: " + resp.Status)

		var reader io.Reader = resp.Body
		if isCompressed {
			dec, err := zstd.NewReader(resp.Body)
			if err != nil {
				logging.GlobalLogger.Fatal("Failed to create zstd streaming reader: " + err.Error())
			}
			defer dec.Close()
			reader = dec
		}

		var hashWriter hash.Hash
		if manifestChecksum != "" {
			hashWriter = md5.New()
			reader = io.TeeReader(reader, hashWriter)
		}

		data, err := io.ReadAll(reader)
		if err != nil {
			logging.GlobalLogger.Fatal("Failed to read patch manifest: " + err.Error())
		}

		if manifestChecksum != "" {
			computedHash := hex.EncodeToString(hashWriter.Sum(nil))
			if computedHash != manifestChecksum {
				if attempt < maxRetries {
					logging.GlobalLogger.Warn("Patch manifest hash mismatch, retrying... (attempt " + strconv.Itoa(attempt) + ")")
					time.Sleep(time.Duration(attempt) * time.Second)
					continue
				}
				logging.GlobalLogger.Fatal("Patch manifest hash mismatch after retries")
			}
		}

		var manifest models.DiffManifest
		err = proto.Unmarshal(data, &manifest)
		if err != nil {
			logging.GlobalLogger.Fatal("Failed to decode patch manifest: " + err.Error())
		}

		logging.GlobalLogger.Info("Patch manifest decoded successfully")
		return &manifest
	}

	logging.GlobalLogger.Fatal("Failed to fetch patch manifest after retries")
	return nil
}
