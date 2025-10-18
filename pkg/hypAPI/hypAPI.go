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

func GetGameBranches(relType string) models.HYPGetGameBranchesResponse {
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
		fmt.Printf("Error fetching game branches: %v\n", err)
		logging.GlobalLogger.Fatal("Failed to fetch game branches: " + err.Error())
	}
	defer resp.Body.Close()
	logging.GlobalLogger.Info("Fetched game branches successfully with status: " + resp.Status)

	var branches models.HYPGetGameBranchesResponse
	err = json.NewDecoder(resp.Body).Decode(&branches)
	if err != nil {
		logging.GlobalLogger.Fatal("Failed to decode game branches response: " + err.Error())
	}
	logging.GlobalLogger.Info("Decoded game branches response successfully")
	logging.GlobalLogger.Info(fmt.Sprintf("Number of game branches fetched: %d", len(branches.Data.GameBranches)))

	return branches
}

func GetGameConfigs(relType string) models.HYPGetGameConfigsResponse {
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
		logging.GlobalLogger.Fatal("Failed to fetch game configs: " + err.Error())
	}
	defer resp.Body.Close()
	logging.GlobalLogger.Info("Fetched game configs successfully with status: " + resp.Status)

	var configs models.HYPGetGameConfigsResponse
	err = json.NewDecoder(resp.Body).Decode(&configs)
	if err != nil {
		logging.GlobalLogger.Fatal("Failed to decode game configs response: " + err.Error())
	}
	logging.GlobalLogger.Info("Decoded game configs response successfully")
	logging.GlobalLogger.Info(fmt.Sprintf("Number of game configs fetched: %d", len(configs.Data.LaunchConfigs)))

	return configs
}

var OSGameBranches = GetGameBranches("os")
var CNGameBranches = GetGameBranches("cn")
var OSGameConfigs = GetGameConfigs("os")
var CNGameConfigs = GetGameConfigs("cn")
