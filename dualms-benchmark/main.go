package main

import (
	"fmt"
	"os"
	"strconv"

	"dualmspoc/internal/core"
	"dualmspoc/internal/local"
	"dualmspoc/network"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "net" {
		if err := network.RunNetworkCommand(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "dualms network failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "bench" {
		if len(os.Args) != 4 {
			fmt.Fprintln(os.Stderr, "usage: go run . bench <runs> <signers>")
			os.Exit(1)
		}

		runs, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid benchmark runs: %v\n", err)
			os.Exit(1)
		}
		if runs <= 0 {
			fmt.Fprintln(os.Stderr, "benchmark runs must be positive")
			os.Exit(1)
		}

		numSigners, err := strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid signer count: %v\n", err)
			os.Exit(1)
		}
		if numSigners <= 0 {
			fmt.Fprintln(os.Stderr, "signer count must be positive")
			os.Exit(1)
		}

		cfg := core.DefaultConfig()
		cfg.EnforceNormBounds = true
		if err := cfg.ApplyDualMSParameterSet(numSigners); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		cfg.RefreshDerived()

		if err := local.BenchmarkRun(cfg, runs); err != nil {
			fmt.Fprintf(os.Stderr, "dualms benchmark failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Fprintln(os.Stderr, "usage: go run . bench <runs> <signers> | go run . net <coordinator|signer> ...")
	os.Exit(1)
}
