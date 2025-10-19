package operations

import (
	"SophonClientv2/internal/logging"
	"SophonClientv2/internal/models"
	"SophonClientv2/pkg/hypAPI"
	"SophonClientv2/pkg/manifest"
	"strings"
)

func GetManifest(gameType string, relType string, matchingField string, branch string) *models.Manifest {
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

	sophonBuild := hypAPI.GetSophonBuildByBranch(relType, targetBranch)
	if sophonBuild.Retcode != 0 {
		logging.GlobalLogger.Fatal("Failed to fetch Sophon build for branch " + selectedGame.Main.Branch + ": " + sophonBuild.Message)
	}

	for _, manifestInfo := range sophonBuild.Data.Manifests {
		if manifestInfo.MatchingField == matchingField {
			mani := manifest.GetManifest(manifestInfo)
			if mani == nil {
				logging.GlobalLogger.Fatal("Failed to fetch manifest for matching field: " + matchingField)
			}
			return mani
		}
	}

	logging.GlobalLogger.Fatal("Failed to find matching manifest with field: " + matchingField)
	return nil
}
