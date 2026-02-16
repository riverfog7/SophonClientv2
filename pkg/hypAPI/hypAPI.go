package hypAPI

import (
	"SophonClientv2/internal/logging"
	"SophonClientv2/internal/models"
	"SophonClientv2/internal/secrets"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

var (
	cacheMu       sync.RWMutex
	branchesCache = make(map[string]models.HYPGetGameBranchesResponse)
	configsCache  = make(map[string]models.HYPGetGameConfigsResponse)
)

func normalizeRelType(relType string) string {
	switch strings.ToLower(relType) {
	case "cn":
		return "cn"
	case "os":
		return "os"
	default:
		logging.GlobalLogger.Warn("Unknown release type, defaulting to OS")
		return "os"
	}
}

func gameBranchesURL(relType string) string {
	if relType == "cn" {
		return secrets.GetGameBranchCNUrl
	}
	return secrets.GetGameBranchOSUrl
}

func gameConfigsURL(relType string) string {
	if relType == "cn" {
		return secrets.GetGameConfigsCNUrl
	}
	return secrets.GetGameConfigsOSUrl
}

func fetchGameBranches(relType string) (models.HYPGetGameBranchesResponse, error) {
	url := gameBranchesURL(relType)

	resp, err := http.Get(url)
	if err != nil {
		return models.HYPGetGameBranchesResponse{}, fmt.Errorf("fetch game branches: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return models.HYPGetGameBranchesResponse{}, fmt.Errorf("fetch game branches: unexpected http status %s", resp.Status)
	}

	logging.GlobalLogger.Info("Fetched game branches successfully with status: " + resp.Status)

	var branches models.HYPGetGameBranchesResponse
	err = json.NewDecoder(resp.Body).Decode(&branches)
	if err != nil {
		return models.HYPGetGameBranchesResponse{}, fmt.Errorf("decode game branches response: %w", err)
	}

	if branches.Retcode != 0 {
		return models.HYPGetGameBranchesResponse{}, fmt.Errorf("game branches api returned retcode=%d message=%s", branches.Retcode, branches.Message)
	}

	logging.GlobalLogger.Info("Decoded game branches response successfully")
	logging.GlobalLogger.Info(fmt.Sprintf("Number of game branches fetched: %d", len(branches.Data.GameBranches)))

	return branches, nil
}

func GetGameBranches(relType string) (models.HYPGetGameBranchesResponse, error) {
	relType = normalizeRelType(relType)

	cacheMu.RLock()
	if branches, ok := branchesCache[relType]; ok {
		cacheMu.RUnlock()
		return branches, nil
	}
	cacheMu.RUnlock()

	branches, err := fetchGameBranches(relType)
	if err != nil {
		return models.HYPGetGameBranchesResponse{}, err
	}

	cacheMu.Lock()
	branchesCache[relType] = branches
	cacheMu.Unlock()

	return branches, nil
}

func fetchGameConfigs(relType string) (models.HYPGetGameConfigsResponse, error) {
	url := gameConfigsURL(relType)

	resp, err := http.Get(url)
	if err != nil {
		return models.HYPGetGameConfigsResponse{}, fmt.Errorf("fetch game configs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return models.HYPGetGameConfigsResponse{}, fmt.Errorf("fetch game configs: unexpected http status %s", resp.Status)
	}

	logging.GlobalLogger.Info("Fetched game configs successfully with status: " + resp.Status)

	var configs models.HYPGetGameConfigsResponse
	err = json.NewDecoder(resp.Body).Decode(&configs)
	if err != nil {
		return models.HYPGetGameConfigsResponse{}, fmt.Errorf("decode game configs response: %w", err)
	}

	if configs.Retcode != 0 {
		return models.HYPGetGameConfigsResponse{}, fmt.Errorf("game configs api returned retcode=%d message=%s", configs.Retcode, configs.Message)
	}

	logging.GlobalLogger.Info("Decoded game configs response successfully")
	logging.GlobalLogger.Info(fmt.Sprintf("Number of game configs fetched: %d", len(configs.Data.LaunchConfigs)))

	return configs, nil
}

func GetGameConfigs(relType string) (models.HYPGetGameConfigsResponse, error) {
	relType = normalizeRelType(relType)

	cacheMu.RLock()
	if configs, ok := configsCache[relType]; ok {
		cacheMu.RUnlock()
		return configs, nil
	}
	cacheMu.RUnlock()

	configs, err := fetchGameConfigs(relType)
	if err != nil {
		return models.HYPGetGameConfigsResponse{}, err
	}

	cacheMu.Lock()
	configsCache[relType] = configs
	cacheMu.Unlock()

	return configs, nil
}
