package main

import (
	"SophonClientv2/pkg/hypAPI"
	"fmt"
)

func StructPrettyPrint(data interface{}) {
	fmt.Printf("%+v\n", data)
}

func main() {
	StructPrettyPrint(hypAPI.CNGameBranches)
	StructPrettyPrint(hypAPI.OSGameBranches)
	StructPrettyPrint(hypAPI.CNGameConfigs)
	StructPrettyPrint(hypAPI.OSGameConfigs)
}
