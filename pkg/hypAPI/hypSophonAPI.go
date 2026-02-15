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

func GetSophonBuildE(url string) (models.SophonGetBuildAPIResponse, error) {
	resp, err := http.Get(url)
	if err != nil {
		return models.SophonGetBuildAPIResponse{}, fmt.Errorf("fetch sophon build: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return models.SophonGetBuildAPIResponse{}, fmt.Errorf("fetch sophon build: unexpected http status %s", resp.Status)
	}

	logging.GlobalLogger.Info("Fetched Sophon build successfully with status: " + resp.Status)

	var buildResponse models.SophonGetBuildAPIResponse
	err = json.NewDecoder(resp.Body).Decode(&buildResponse)
	if err != nil {
		return models.SophonGetBuildAPIResponse{}, fmt.Errorf("decode sophon build response: %w", err)
	}

	if buildResponse.Retcode != 0 {
		return models.SophonGetBuildAPIResponse{}, fmt.Errorf("sophon build api returned retcode=%d message=%s", buildResponse.Retcode, buildResponse.Message)
	}

	logging.GlobalLogger.Info("Decoded Sophon build response successfully")
	return buildResponse, nil
}

func GetSophonBuild(url string) models.SophonGetBuildAPIResponse {
	build, err := GetSophonBuildE(url)
	if err != nil {
		logging.GlobalLogger.Error("Failed to get Sophon build: " + err.Error())
		return models.SophonGetBuildAPIResponse{}
	}
	return build
}

func GetSophonBuildByBranchE(relType string, branch models.HYPGameBranch) (models.SophonGetBuildAPIResponse, error) {
	url := BuildSophonGetBuildURL(relType, branch)
	return GetSophonBuildE(url)
}

func GetSophonBuildByBranch(relType string, branch models.HYPGameBranch) models.SophonGetBuildAPIResponse {
	build, err := GetSophonBuildByBranchE(relType, branch)
	if err != nil {
		logging.GlobalLogger.Error("Failed to get Sophon build by branch: " + err.Error())
		return models.SophonGetBuildAPIResponse{}
	}
	return build
}
