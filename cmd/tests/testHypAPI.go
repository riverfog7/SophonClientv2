package main

import (
	"SophonClientv2/pkg/hypAPI"
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
}
