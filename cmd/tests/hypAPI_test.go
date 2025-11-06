package main

import (
	//"SophonClientv2/internal/logging"
	"SophonClientv2/pkg/hypAPI"
	"SophonClientv2/pkg/installer"
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

	mani, info := operations.GetManifest("hkrpg", "os", "game", "main")
	inst := installer.NewInstaller(".", ".", 100)
	_ = inst.ParseManifest(mani, info.ChunkDownload)
	mani, info = operations.GetManifest("hkrpg", "cn", "game", "main")
	inst = installer.NewInstaller(".", ".", 100)
	_ = inst.ParseManifest(mani, info.ChunkDownload)
	mani, info = operations.GetManifest("hk4e", "os", "game", "main")
	inst = installer.NewInstaller(".", ".", 100)
	_ = inst.ParseManifest(mani, info.ChunkDownload)
	mani, info = operations.GetManifest("hk4e", "cn", "game", "main")
	inst = installer.NewInstaller(".", ".", 100)
	_ = inst.ParseManifest(mani, info.ChunkDownload)
	mani, info = operations.GetManifest("bh3", "os", "game", "main")
	inst = installer.NewInstaller(".", ".", 100)
	_ = inst.ParseManifest(mani, info.ChunkDownload)
	mani, info = operations.GetManifest("bh3", "cn", "game", "main")
	inst = installer.NewInstaller(".", ".", 100)
	_ = inst.ParseManifest(mani, info.ChunkDownload)
	mani, info = operations.GetManifest("nap", "os", "game", "main")
	inst = installer.NewInstaller(".", ".", 100)
	_ = inst.ParseManifest(mani, info.ChunkDownload)
	mani, info = operations.GetManifest("nap", "cn", "game", "main")
	inst = installer.NewInstaller(".", ".", 100)
	_ = inst.ParseManifest(mani, info.ChunkDownload)

	//logging.GlobalLogger.Warn("Testing with real game files")
	//installer = operations.NewInstaller("/Volumes/SSD/Games/Genshin Impact game1", "/Volumes/SSD/Games/Genshin Impact game1/.cache")
	//mani, info = operations.GetManifest("hk4e", "os", "game", "main")
	//_ = installer.ParseManifest(mani, info.ChunkDownload)
	//_ = installer.Prepare()
	//installer.Start()
	//installer.Wait()
	//installer.Stop()

	//mani, info = operations.GetManifest("hk4e", "os", "ko-kr", "main")
	//// same dir (audiopack test)
	//installer = operations.NewInstaller("/Volumes/SSD/Games/Genshin Impact game1", "/Volumes/SSD/Games/Genshin Impact game1/.cache")
	//_ = installer.ParseManifest(mani, info.ChunkDownload)
	//_ = installer.Prepare()
	//installer.Start()
	//installer.Wait()

	mani, info = operations.GetManifest("hk4e", "os", "game", "main")
	inst = installer.NewInstaller("/Volumes/SSD/Games/Genshin Impact game1", "/Volumes/SSD/Games/Genshin Impact game1/.cache", 50)
	_ = inst.ParseManifest(mani, info.ChunkDownload)
	_ = inst.Prepare()
	inst.Start()
	inst.Wait()
	inst.Stop()
}
