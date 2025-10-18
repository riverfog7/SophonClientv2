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

func BuildSophonGetBuildURL(relType string, branch models.HYPGameBranch) string {
	var baseURL string
	switch strings.ToLower(relType) {
	case "cn":
		baseURL = secrets.CNSophonAPIBaseURL
	case "os":
		baseURL = secrets.OSSophonAPIBaseURL
	default:
		logging.GlobalLogger.Warn("Unknown release type in function BuildSophonGetBuildURL, defaulting to OS")
		baseURL = secrets.OSSophonAPIBaseURL
	}
	return fmt.Sprintf(
		"%s?package_id=%s&branch=%s&password=%s",
		baseURL,
		branch.PackageId,
		branch.Branch,
		branch.Password,
	)
}

func GetSophonBuild(url string) models.SophonGetBuildAPIResponse {
	resp, err := http.Get(url)
	if err != nil {
		logging.GlobalLogger.Fatal("Failed to fetch Sophon build: " + err.Error())
	}
	defer resp.Body.Close()
	logging.GlobalLogger.Info("Fetched Sophon build successfully with status: " + resp.Status)

	var buildResponse models.SophonGetBuildAPIResponse
	err = json.NewDecoder(resp.Body).Decode(&buildResponse)
	if err != nil {
		logging.GlobalLogger.Fatal("Failed to decode Sophon build response: " + err.Error())
	}
	logging.GlobalLogger.Info("Decoded Sophon build response successfully")
	return buildResponse
}

func GetSophonBuildByBranch(relType string, branch models.HYPGameBranch) models.SophonGetBuildAPIResponse {
	url := BuildSophonGetBuildURL(relType, branch)
	return GetSophonBuild(url)
}
