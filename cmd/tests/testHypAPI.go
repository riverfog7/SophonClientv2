package main

import (
	"SophonClientv2/pkg/hypAPI"
	"SophonClientv2/pkg/operations"
	"fmt"
)

func StructPrettyPrint(data interface{}) {
	fmt.Printf("%+v\n", data)
}

func main() {
	StructPrettyPrint(hypAPI.CNGameConfigs)
	StructPrettyPrint(hypAPI.OSGameConfigs)
	StructPrettyPrint(hypAPI.CNGameBranches)
	StructPrettyPrint(hypAPI.OSGameBranches)

	fmt.Println("Fetching CN Game Branches...")
	for _, gameBranch := range hypAPI.CNGameBranches.Data.GameBranches {
		mainBranch := gameBranch.Main
		url := hypAPI.BuildSophonGetBuildURL("cn", mainBranch)
		fmt.Println(url)
		sophon := hypAPI.GetSophonBuild(url)
		if sophon.Retcode != 0 {
			fmt.Printf("Error fetching Sophon build for %s: %s\n", mainBranch.Branch, sophon.Message)
		}
	}

	fmt.Println("Fetching OS Game Branches...")
	for _, gameBranch := range hypAPI.OSGameBranches.Data.GameBranches {
		mainBranch := gameBranch.Main
		url := hypAPI.BuildSophonGetBuildURL("os", mainBranch)
		fmt.Println(url)
		sophon := hypAPI.GetSophonBuild(url)
		if sophon.Retcode != 0 {
			fmt.Printf("Error fetching Sophon build for %s: %s\n", mainBranch.Branch, sophon.Message)
		}
	}

	_ = operations.GetManifest("hkrpg", "os", "game", "main")
	_ = operations.GetManifest("hkrpg", "cn", "game", "main")
	_ = operations.GetManifest("hk4e", "os", "game", "main")
	_ = operations.GetManifest("hk4e", "cn", "game", "main")
	_ = operations.GetManifest("bh3", "os", "game", "main")
	_ = operations.GetManifest("bh3", "cn", "game", "main")
	_ = operations.GetManifest("nap", "os", "game", "main")
	_ = operations.GetManifest("nap", "cn", "game", "main")
}
