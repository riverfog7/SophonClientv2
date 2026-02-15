package hypAPI

import (
	"SophonClientv2/internal/logging"
	"SophonClientv2/internal/models"
	"SophonClientv2/internal/secrets"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func GetGameBranchesE(relType string) (models.HYPGetGameBranchesResponse, error) {
	var url string
	switch strings.ToLower(relType) {
	case "cn":
		url = secrets.GetGameBranchCNUrl
	case "os":
		url = secrets.GetGameBranchOSUrl
	default:
		logging.GlobalLogger.Warn("Unknown release type in function GetGameBranches, defaulting to OS")
		url = secrets.GetGameBranchOSUrl
	}

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

func GetGameBranches(relType string) models.HYPGetGameBranchesResponse {
	branches, err := GetGameBranchesE(relType)
	if err != nil {
		logging.GlobalLogger.Error("Failed to get game branches: " + err.Error())
		return models.HYPGetGameBranchesResponse{}
	}
	return branches
}

func GetGameConfigsE(relType string) (models.HYPGetGameConfigsResponse, error) {
	var url string
	switch strings.ToLower(relType) {
	case "cn":
		url = secrets.GetGameConfigsCNUrl
	case "os":
		url = secrets.GetGameConfigsOSUrl
	default:
		logging.GlobalLogger.Warn("Unknown release type in function GetGameConfigs, defaulting to OS")
		url = secrets.GetGameConfigsOSUrl
	}

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

func GetGameConfigs(relType string) models.HYPGetGameConfigsResponse {
	configs, err := GetGameConfigsE(relType)
	if err != nil {
		logging.GlobalLogger.Error("Failed to get game configs: " + err.Error())
		return models.HYPGetGameConfigsResponse{}
	}
	return configs
}

var OSGameBranches = GetGameBranches("os")
var CNGameBranches = GetGameBranches("cn")
var OSGameConfigs = GetGameConfigs("os")
var CNGameConfigs = GetGameConfigs("cn")
