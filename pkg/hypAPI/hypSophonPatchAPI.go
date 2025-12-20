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

func GetSophonPatchBuild(url string) models.SophonGetPatchBuildAPIResponse {
	resp, err := http.Post(url, "text/plain", nil)
	if err != nil {
		logging.GlobalLogger.Fatal("Failed to fetch Sophon patch build: " + err.Error())
	}
	defer resp.Body.Close()
	logging.GlobalLogger.Info("Fetched Sophon patch build successfully with status: " + resp.Status)

	var buildResponse models.SophonGetPatchBuildAPIResponse
	err = json.NewDecoder(resp.Body).Decode(&buildResponse)
	if err != nil {
		logging.GlobalLogger.Fatal("Failed to decode Sophon patch build response: " + err.Error())
	}
	logging.GlobalLogger.Info("Decoded Sophon patch build response successfully")
	return buildResponse
}

func GetSophonPatchBuildByBranch(relType string, branch models.HYPGameBranch) models.SophonGetPatchBuildAPIResponse {
	url := PatchSophonGetBuildURL(relType, branch)
	return GetSophonPatchBuild(url)
}
