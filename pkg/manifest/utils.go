package manifest

import (
	"SophonClientv2/internal/logging"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"

	"github.com/klauspost/compress/zstd"
)

func buildManifestURL(urlPrefix, manifestID, urlSuffix string) string {
	if urlSuffix != "" {
		return urlPrefix + "/" + manifestID + "/" + urlSuffix
	}
	return urlPrefix + "/" + manifestID
}

func isRetryableManifestStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func fetchManifestPayload(url string, isCompressed bool, manifestChecksum string, label string) ([]byte, bool, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, true, fmt.Errorf("fetch %s: %w", label, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, isRetryableManifestStatus(resp.StatusCode), fmt.Errorf("fetch %s: unexpected http status %s", label, resp.Status)
	}

	logging.GlobalLogger.Info("Fetched " + label + " successfully with status: " + resp.Status)

	reader := io.Reader(resp.Body)
	var dec *zstd.Decoder
	if isCompressed {
		dec, err = zstd.NewReader(resp.Body)
		if err != nil {
			return nil, false, fmt.Errorf("create zstd streaming reader for %s: %w", label, err)
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
		return nil, true, fmt.Errorf("read %s: %w", label, err)
	}

	if manifestChecksum != "" {
		computedHash := hex.EncodeToString(hashWriter.Sum(nil))
		if computedHash != manifestChecksum {
			return nil, true, fmt.Errorf("%s hash mismatch", label)
		}
	}

	return data, false, nil
}
