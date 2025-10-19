package manifest

import (
	"SophonClientv2/internal/logging"
	"SophonClientv2/internal/models"
	"io"
	"net/http"

	"github.com/klauspost/compress/zstd"
	"google.golang.org/protobuf/proto"
)

func GetManifest(sophonBuildAPIManifest models.SophonManifest) *models.Manifest {
	var url string
	urlPrefix := sophonBuildAPIManifest.ManifestDownload.UrlPrefix
	urlSuffix := sophonBuildAPIManifest.ManifestDownload.UrlSuffix
	manifestID := sophonBuildAPIManifest.Manifest.ID

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

	// HTTP Get
	resp, err := http.Get(url)
	if err != nil {
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

	var manifest models.Manifest
	err = proto.Unmarshal(data, &manifest)
	if err != nil {
		logging.GlobalLogger.Fatal("Failed to decode manifest: " + err.Error())
	}

	logging.GlobalLogger.Info("Manifest decoded successfully")
	return &manifest
}
