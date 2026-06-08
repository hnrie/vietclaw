package main

import (
	"fmt"
	"runtime"

	"vietclaw/internal/version"
)

func runVersion() error {
	info := version.Current()
	fmt.Printf("VietClaw %s\n", info.Version)
	fmt.Printf("commit %s\n", info.Commit)
	fmt.Printf("go %s\n", runtime.Version())
	return nil
}
