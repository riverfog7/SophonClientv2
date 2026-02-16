package main

import (
	"SophonClientv2/pkg/hypAPI"
	"SophonClientv2/pkg/installer"
	"SophonClientv2/pkg/operations"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"testing"
)

func StructPrettyPrint(data interface{}) {
	fmt.Printf("%+v\n", data)
}

func TestHypAPIConfigs(t *testing.T) {
	cnConfigs, err := hypAPI.GetGameConfigs("cn")
	if err != nil {
		t.Fatalf("failed to fetch CN game configs: %v", err)
	}
	osConfigs, err := hypAPI.GetGameConfigs("os")
	if err != nil {
		t.Fatalf("failed to fetch OS game configs: %v", err)
	}
	cnBranches, err := hypAPI.GetGameBranches("cn")
	if err != nil {
		t.Fatalf("failed to fetch CN game branches: %v", err)
	}
	osBranches, err := hypAPI.GetGameBranches("os")
	if err != nil {
		t.Fatalf("failed to fetch OS game branches: %v", err)
	}

	StructPrettyPrint(cnConfigs)
	StructPrettyPrint(osConfigs)
	StructPrettyPrint(cnBranches)
	StructPrettyPrint(osBranches)
}

func TestFetchCNGameBranches(t *testing.T) {
	branches, err := hypAPI.GetGameBranches("cn")
	if err != nil {
		t.Fatalf("failed to fetch CN game branches: %v", err)
	}

	fmt.Println("Fetching CN Game Branches...")
	for _, gameBranch := range branches.Data.GameBranches {
		mainBranch := gameBranch.Main
		url := hypAPI.BuildSophonGetBuildURL("cn", mainBranch)
		fmt.Println(url)
		sophon, err := hypAPI.GetSophonBuild(url)
		if err != nil {
			t.Logf("Error fetching Sophon build for %s: %v\n", mainBranch.Branch, err)
			continue
		}
		if sophon.Retcode != 0 {
			t.Logf("Error fetching Sophon build for %s: %s\n", mainBranch.Branch, sophon.Message)
		}
	}
}

func TestFetchOSGameBranches(t *testing.T) {
	branches, err := hypAPI.GetGameBranches("os")
	if err != nil {
		t.Fatalf("failed to fetch OS game branches: %v", err)
	}

	fmt.Println("Fetching OS Game Branches...")
	for _, gameBranch := range branches.Data.GameBranches {
		mainBranch := gameBranch.Main
		url := hypAPI.BuildSophonGetBuildURL("os", mainBranch)
		fmt.Println(url)
		sophon, err := hypAPI.GetSophonBuild(url)
		if err != nil {
			t.Logf("Error fetching Sophon build for %s: %v\n", mainBranch.Branch, err)
			continue
		}
		if sophon.Retcode != 0 {
			t.Logf("Error fetching Sophon build for %s: %s\n", mainBranch.Branch, sophon.Message)
		}
	}
}

func TestFetchCNGamePatchBuilds(t *testing.T) {
	branches, err := hypAPI.GetGameBranches("cn")
	if err != nil {
		t.Fatalf("failed to fetch CN game branches: %v", err)
	}

	fmt.Println("Fetching CN Game Patch Builds...")
	for _, gameBranch := range branches.Data.GameBranches {
		mainBranch := gameBranch.Main
		url := hypAPI.PatchSophonGetBuildURL("cn", mainBranch)
		fmt.Println(url)
		sophon, err := hypAPI.GetSophonPatchBuild(url)
		if err != nil {
			t.Logf("Error fetching Sophon patch build for %s: %v\n", mainBranch.Branch, err)
			continue
		}
		if sophon.Retcode != 0 {
			t.Logf("Error fetching Sophon patch build for %s: %s\n", mainBranch.Branch, sophon.Message)
		}
	}
}

func TestFetchOSGamePatchBuilds(t *testing.T) {
	branches, err := hypAPI.GetGameBranches("os")
	if err != nil {
		t.Fatalf("failed to fetch OS game branches: %v", err)
	}

	fmt.Println("Fetching OS Game Patch Builds...")
	for _, gameBranch := range branches.Data.GameBranches {
		mainBranch := gameBranch.Main
		url := hypAPI.PatchSophonGetBuildURL("os", mainBranch)
		fmt.Println(url)
		sophon, err := hypAPI.GetSophonPatchBuild(url)
		if err != nil {
			t.Logf("Error fetching Sophon patch build for %s: %v\n", mainBranch.Branch, err)
			continue
		}
		if sophon.Retcode != 0 {
			t.Logf("Error fetching Sophon patch build for %s: %s\n", mainBranch.Branch, sophon.Message)
		}
	}
}

func TestParseAllManifests(t *testing.T) {
	targets := [][2]string{
		{"hkrpg", "os"},
		{"hkrpg", "cn"},
		{"hk4e", "os"},
		{"hk4e", "cn"},
		{"bh3", "os"},
		{"bh3", "cn"},
		{"nap", "os"},
		{"nap", "cn"},
	}

	for _, target := range targets {
		mani, info, err := operations.GetAndParseManifest(target[0], target[1], "game", "main")
		if err != nil {
			t.Fatalf("failed parsing manifest for %s_%s: %v", target[0], target[1], err)
		}

		inst := installer.NewInstaller(".", ".", 100)
		_ = inst.ParseManifest(mani, info.ChunkDownload)
	}
}

func TestParseAllPatchManifests(t *testing.T) {
	targets := [][2]string{
		{"hkrpg", "os"},
		{"hkrpg", "cn"},
		{"hk4e", "os"},
		{"hk4e", "cn"},
		{"nap", "os"},
		{"nap", "cn"},
	}

	for _, target := range targets {
		_, _, err := operations.GetAndParsePatchManifest(target[0], target[1], "game", "main")
		if err != nil {
			t.Fatalf("failed parsing patch manifest for %s_%s: %v", target[0], target[1], err)
		}
	}
}

func TestFullInstallation(t *testing.T) {
	f_c, err := os.Create("cpu.prof")
	if err != nil {
		t.Fatalf("Could not create CPU profile: %v", err)
	}
	f_m, err := os.Create("mem.prof")
	if err != nil {
		t.Fatalf("Could not create memory profile: %v", err)
	}
	defer f_c.Close()
	defer f_m.Close()

	if err := pprof.StartCPUProfile(f_c); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
	defer pprof.StopCPUProfile()

	mani, info, err := operations.GetAndParseManifest("hk4e", "os", "game", "main")
	if err != nil {
		t.Fatalf("failed getting install manifest: %v", err)
	}
	inst := installer.NewInstaller("/Volumes/SSD/Games/Genshin Impact game1", "/Volumes/SSD/Games/Genshin Impact game1/.cache", 50)
	_ = inst.ParseManifest(mani, info.ChunkDownload)
	_ = inst.Prepare()
	inst.Start()
	if err := inst.WaitWithError(); err != nil {
		t.Fatalf("installation failed: %v", err)
	}
	inst.Stop()

	if err := pprof.Lookup("heap").WriteTo(f_m, 0); err != nil {
		log.Fatal("could not write memory profile: ", err)
	}
}
