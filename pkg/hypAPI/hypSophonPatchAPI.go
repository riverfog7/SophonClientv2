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

func PatchSophonGetBuildURL(relType string, branch models.HYPGameBranch) string {
	var baseURL string
	switch strings.ToLower(relType) {
	case "cn":
		baseURL = secrets.CNSophonPatchAPIBaseURL
	case "os":
		baseURL = secrets.OSSophonPatchAPIBaseURL
	default:
		logging.GlobalLogger.Warn("Unknown release type in function PatchSophonGetBuildURL, defaulting to OS")
		baseURL = secrets.OSSophonPatchAPIBaseURL
	}
	return fmt.Sprintf(
		"%s?package_id=%s&branch=%s&password=%s",
		baseURL,
		branch.PackageId,
		branch.Branch,
		branch.Password,
	)
}

func GetSophonPatchBuild(url string) (models.SophonGetPatchBuildAPIResponse, error) {
	resp, err := http.Post(url, "text/plain", nil)
	if err != nil {
		return models.SophonGetPatchBuildAPIResponse{}, fmt.Errorf("fetch sophon patch build: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return models.SophonGetPatchBuildAPIResponse{}, fmt.Errorf("fetch sophon patch build: unexpected http status %s", resp.Status)
	}

	logging.GlobalLogger.Info("Fetched Sophon patch build successfully with status: " + resp.Status)

	var buildResponse models.SophonGetPatchBuildAPIResponse
	err = json.NewDecoder(resp.Body).Decode(&buildResponse)
	if err != nil {
		return models.SophonGetPatchBuildAPIResponse{}, fmt.Errorf("decode sophon patch build response: %w", err)
	}

	if buildResponse.Retcode != 0 {
		return models.SophonGetPatchBuildAPIResponse{}, fmt.Errorf("sophon patch build api returned retcode=%d message=%s", buildResponse.Retcode, buildResponse.Message)
	}

	logging.GlobalLogger.Info("Decoded Sophon patch build response successfully")
	return buildResponse, nil
}

func GetSophonPatchBuildByBranch(relType string, branch models.HYPGameBranch) (models.SophonGetPatchBuildAPIResponse, error) {
	url := PatchSophonGetBuildURL(relType, branch)
	return GetSophonPatchBuild(url)
}
