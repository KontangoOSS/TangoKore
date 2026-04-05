package main

import (
	"flag"
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "enroll":
		cmdEnroll(os.Args[2:])
	case "agent":
		cmdAgent(os.Args[2:])
	case "portal":
		cmdPortal(os.Args[2:])
	case "status":
		cmdStatus(os.Args[2:])
	case "reset":
		cmdReset(os.Args[2:])
	case "cluster":
		cmdCluster(os.Args[2:])
	case "controller":
		cmdController(os.Args[2:])
	case "version":
		fmt.Println("kontango", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`kontango — zero-trust mesh SDK

Machine Commands:
  enroll       Enroll this machine onto the mesh
  agent        Start the machine agent (telemetry + config listener)
  portal       Start the local web portal (docs + machine status)
  status       Show machine status
  reset        Remove all kontango services, identity, and binaries

Cluster Commands:
  cluster      Manage cluster enrollment and operations
  controller   Deploy or manage a new controller cluster

Utility:
  version      Print version
  help         Show this help message

Usage:
  kontango enroll <url> [--scan] [--role-id ID --secret-id SECRET]
  kontango agent [--identity /opt/kontango/identity.json]
  kontango portal [--listen :8800] [--identity /opt/kontango/identity.json]
  kontango status [--json]
  kontango reset [--yes]
  kontango cluster create [--name NAME] [--nodes N] [--region REGION]
  kontango cluster join <controller-url>
  kontango cluster status [--json]
  kontango controller create [--name NAME] [--domain DOMAIN]
  kontango controller status [--json]`)
}

// Ensure flag import is used (referenced by subcommands in other files).
var _ = flag.ExitOnError
