package operations

import (
	"SophonClientv2/internal/logging"
	"SophonClientv2/internal/models"
	"SophonClientv2/pkg/hypAPI"
	"SophonClientv2/pkg/manifest"
	"strings"
)

func getGameBranch(gameType string, relType string, branch string) models.HYPGameBranch {
	var biz string
	var hypGames []models.HYPGame
	switch strings.ToLower(relType) {
	case "cn":
		biz = strings.ToLower(gameType) + "_cn"
		hypGames = hypAPI.CNGameBranches.Data.GameBranches
	case "os":
		biz = strings.ToLower(gameType) + "_global"
		hypGames = hypAPI.OSGameBranches.Data.GameBranches
	default:
		logging.GlobalLogger.Warn("Unknown release type in function GetAndParseManifest, defaulting to OS")
		biz = strings.ToLower(gameType) + "_global"
	}

	var selectedGame models.HYPGame
	for i, hypGame := range hypGames {
		if strings.ToLower(hypGame.Game.Biz) == biz {
			selectedGame = hypGames[i]
		}
	}

	var targetBranch models.HYPGameBranch
	switch strings.ToLower(branch) {
	case "main":
		targetBranch = selectedGame.Main
	case "predownload":
		if selectedGame.PreDownload != nil {
			targetBranch = *selectedGame.PreDownload
		} else {
			logging.GlobalLogger.Fatal("PreDownload branch not available for game: " + gameType + "_" + relType)
		}
	default:
		logging.GlobalLogger.Warn("Unknown branch type in function GetAndParseManifest, defaulting to Main")
		targetBranch = selectedGame.Main
	}

	return targetBranch
}

func GetAndParseManifest(gameType string, relType string, matchingField string, branch string) (*models.Manifest, *models.SophonManifest) {
	targetBranch := getGameBranch(gameType, relType, branch)
	sophonBuild := hypAPI.GetSophonBuildByBranch(relType, targetBranch)
	if sophonBuild.Retcode != 0 {
		logging.GlobalLogger.Fatal("Failed to fetch Sophon build for branch " + targetBranch.Branch + ": " + sophonBuild.Message)
	}
	for _, manifestInfo := range sophonBuild.Data.Manifests {
		logging.GlobalLogger.Info("Matching field " + manifestInfo.MatchingField + " found for game " + gameType + "_" + relType + " on branch " + branch)
	}
	for _, manifestInfo := range sophonBuild.Data.Manifests {
		if manifestInfo.MatchingField == matchingField {
			mani := manifest.GetManifest(manifestInfo)
			if mani == nil {
				logging.GlobalLogger.Fatal("Failed to fetch manifest for matching field: " + matchingField)
			}
			return mani, &manifestInfo
		}
	}

	logging.GlobalLogger.Fatal("Failed to find matching manifest with field: " + matchingField)
	return nil, nil
}

func GetAndParsePatchManifest(gameType string, relType string, matchingField string, branch string) (*models.DiffManifest, *models.SophonPatchManifest) {
	targetBranch := getGameBranch(gameType, relType, branch)
	sophonPatchBuild := hypAPI.GetSophonPatchBuildByBranch(relType, targetBranch)
	if sophonPatchBuild.Retcode != 0 {
		logging.GlobalLogger.Fatal("Failed to fetch Sophon patch build for branch " + targetBranch.Branch + ": " + sophonPatchBuild.Message)
	}
	for _, manifestInfo := range sophonPatchBuild.Data.Manifests {
		logging.GlobalLogger.Info("Matching field " + manifestInfo.MatchingField + " found for game " + gameType + "_" + relType + " on branch " + branch)
	}
	for _, manifestInfo := range sophonPatchBuild.Data.Manifests {
		if manifestInfo.MatchingField == matchingField {
			mani := manifest.GetLdiffManifest(manifestInfo)
			if mani == nil {
				logging.GlobalLogger.Fatal("Failed to fetch patch manifest for matching field: " + matchingField)
			}
			return mani, &manifestInfo
		}
	}

	logging.GlobalLogger.Fatal("Failed to find matching patch manifest with field: " + matchingField)
	return nil, nil
}
