package main

import (
	"SophonClientv2/pkg/hypAPI"
	"SophonClientv2/pkg/installer"
	"SophonClientv2/pkg/operations"
	"fmt"
	"testing"
)

func StructPrettyPrint(data interface{}) {
	fmt.Printf("%+v\n", data)
}

func TestHypAPIConfigs(t *testing.T) {
	StructPrettyPrint(hypAPI.CNGameConfigs)
	StructPrettyPrint(hypAPI.OSGameConfigs)
	StructPrettyPrint(hypAPI.CNGameBranches)
	StructPrettyPrint(hypAPI.OSGameBranches)
}

func TestFetchCNGameBranches(t *testing.T) {
	fmt.Println("Fetching CN Game Branches...")
	for _, gameBranch := range hypAPI.CNGameBranches.Data.GameBranches {
		mainBranch := gameBranch.Main
		url := hypAPI.BuildSophonGetBuildURL("cn", mainBranch)
		fmt.Println(url)
		sophon := hypAPI.GetSophonBuild(url)
		if sophon.Retcode != 0 {
			t.Logf("Error fetching Sophon build for %s: %s\n", mainBranch.Branch, sophon.Message)
		}
	}
}

func TestFetchOSGameBranches(t *testing.T) {
	fmt.Println("Fetching OS Game Branches...")
	for _, gameBranch := range hypAPI.OSGameBranches.Data.GameBranches {
		mainBranch := gameBranch.Main
		url := hypAPI.BuildSophonGetBuildURL("os", mainBranch)
		fmt.Println(url)
		sophon := hypAPI.GetSophonBuild(url)
		if sophon.Retcode != 0 {
			t.Logf("Error fetching Sophon build for %s: %s\n", mainBranch.Branch, sophon.Message)
		}
	}
}

func TestParseAllManifests(t *testing.T) {
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
}

func TestFullInstallation(t *testing.T) {
	mani, info := operations.GetManifest("hk4e", "os", "game", "main")
	inst := installer.NewInstaller("/Volumes/SSD/Games/Genshin Impact game1", "/Volumes/SSD/Games/Genshin Impact game1/.cache", 50)
	_ = inst.ParseManifest(mani, info.ChunkDownload)
	_ = inst.Prepare()
	inst.Start()
	inst.Wait()
	inst.Stop()
}
