package operations

import (
	"SophonClientv2/internal/logging"
	"SophonClientv2/internal/models"
	"SophonClientv2/pkg/hypAPI"
	"SophonClientv2/pkg/manifest"
	"fmt"
	"strings"
)

func getGameBranch(gameType string, relType string, branch string) (models.HYPGameBranch, error) {
	gameType = strings.ToLower(gameType)
	relType = strings.ToLower(relType)

	var biz string
	switch relType {
	case "cn":
		biz = gameType + "_cn"
	default:
		biz = gameType + "_global"
	}

	branchesResp, err := hypAPI.GetGameBranches(relType)
	if err != nil {
		return models.HYPGameBranch{}, fmt.Errorf("get game branches for %s: %w", relType, err)
	}
	hypGames := branchesResp.Data.GameBranches

	var selectedGame *models.HYPGame
	for i := range hypGames {
		hypGame := hypGames[i]
		if strings.ToLower(hypGame.Game.Biz) == biz {
			selectedGame = &hypGames[i]
			break
		}
	}

	if selectedGame == nil {
		return models.HYPGameBranch{}, fmt.Errorf("game branch not found for biz %s", biz)
	}

	var targetBranch models.HYPGameBranch
	switch strings.ToLower(branch) {
	case "main":
		targetBranch = selectedGame.Main
	case "predownload":
		if selectedGame.PreDownload != nil {
			targetBranch = *selectedGame.PreDownload
		} else {
			return models.HYPGameBranch{}, fmt.Errorf("predownload branch not available for game %s_%s", gameType, relType)
		}
	default:
		logging.GlobalLogger.Warn("Unknown branch type in function GetAndParseManifest, defaulting to Main")
		targetBranch = selectedGame.Main
	}

	return targetBranch, nil
}

func GetAndParseManifest(gameType string, relType string, matchingField string, branch string) (*models.Manifest, *models.SophonManifest, error) {
	targetBranch, err := getGameBranch(gameType, relType, branch)
	if err != nil {
		return nil, nil, err
	}

	sophonBuild, err := hypAPI.GetSophonBuildByBranch(relType, targetBranch)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch sophon build for branch %s: %w", targetBranch.Branch, err)
	}

	for _, manifestInfo := range sophonBuild.Data.Manifests {
		logging.GlobalLogger.Info("Matching field " + manifestInfo.MatchingField + " found for game " + gameType + "_" + relType + " on branch " + branch)
	}

	for _, manifestInfo := range sophonBuild.Data.Manifests {
		if manifestInfo.MatchingField == matchingField {
			mani, err := manifest.GetManifest(manifestInfo)
			if err != nil {
				return nil, nil, fmt.Errorf("fetch manifest for matching field %s: %w", matchingField, err)
			}
			return mani, &manifestInfo, nil
		}
	}

	return nil, nil, fmt.Errorf("matching manifest not found for field %s", matchingField)
}

func GetAndParsePatchManifest(gameType string, relType string, matchingField string, branch string) (*models.DiffManifest, *models.SophonPatchManifest, error) {
	targetBranch, err := getGameBranch(gameType, relType, branch)
	if err != nil {
		return nil, nil, err
	}

	sophonPatchBuild, err := hypAPI.GetSophonPatchBuildByBranch(relType, targetBranch)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch sophon patch build for branch %s: %w", targetBranch.Branch, err)
	}

	for _, manifestInfo := range sophonPatchBuild.Data.Manifests {
		logging.GlobalLogger.Info("Matching field " + manifestInfo.MatchingField + " found for game " + gameType + "_" + relType + " on branch " + branch)
	}

	for _, manifestInfo := range sophonPatchBuild.Data.Manifests {
		if manifestInfo.MatchingField == matchingField {
			mani, err := manifest.GetLdiffManifest(manifestInfo)
			if err != nil {
				return nil, nil, fmt.Errorf("fetch patch manifest for matching field %s: %w", matchingField, err)
			}
			return mani, &manifestInfo, nil
		}
	}

	return nil, nil, fmt.Errorf("matching patch manifest not found for field %s", matchingField)
}
