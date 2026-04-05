package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
)

func cmdCluster(args []string) {
	if len(args) < 1 {
		printClusterUsage()
		os.Exit(1)
	}

	subcommand := args[0]
	switch subcommand {
	case "create":
		cmdClusterCreate(args[1:])
	case "join":
		cmdClusterJoin(args[1:])
	case "status":
		cmdClusterStatus(args[1:])
	case "leave":
		cmdClusterLeave(args[1:])
	case "upgrade":
		cmdClusterUpgrade(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown cluster subcommand: %s\n", subcommand)
		printClusterUsage()
		os.Exit(1)
	}
}

func cmdClusterCreate(args []string) {
	fs := flag.NewFlagSet("cluster create", flag.ExitOnError)
	name := fs.String("name", "", "cluster name")
	nodes := fs.Int("nodes", 3, "number of nodes in HA cluster")
	region := fs.String("region", "us-east-1", "cloud region for deployment")
	fs.Parse(args)

	log.SetFlags(0)
	fmt.Printf("cluster create is not yet implemented\n")
	fmt.Printf("  name:   %s\n", *name)
	fmt.Printf("  nodes:  %d\n", *nodes)
	fmt.Printf("  region: %s\n", *region)
	fmt.Printf("\nUse the controller create command to deploy a full infra:\n")
	fmt.Printf("  kontango controller create\n")
}

func cmdClusterJoin(args []string) {
	fs := flag.NewFlagSet("cluster join", flag.ExitOnError)
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "usage: kontango cluster join <controller-url>\n")
		os.Exit(1)
	}

	url := fs.Arg(0)
	log.SetFlags(0)

	log.Printf("joining cluster at %s…\n", url)

	// The join flow is:
	// 1. Enroll this machine against the controller URL
	// 2. Wait for Ziti tunnel to come up
	// 3. Run ziti edge commands to wire this node in as a join controller
	// For now, just enroll
	if err := runEnroll(url, "", "", "", false); err != nil {
		fmt.Fprintf(os.Stderr, "enroll: %v\n", err)
		os.Exit(1)
	}

	log.Printf("machine enrolled successfully")
	log.Printf("next steps:")
	log.Printf("  1. wait for ziti tunnel to come online")
	log.Printf("  2. run: ziti edge login %s:1280\n", url)
}

func cmdClusterStatus(args []string) {
	fs := flag.NewFlagSet("cluster status", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "output as JSON")
	fs.Parse(args)

	log.SetFlags(0)

	// Try to load machine.json to see if we're enrolled
	machineFile := "/opt/kontango/machine.json"
	data, err := os.ReadFile(machineFile)
	if err != nil {
		if *jsonOut {
			fmt.Println(`{"status":"not-enrolled","error":"machine.json not found"}`)
		} else {
			fmt.Printf("cluster status: not enrolled\n")
			fmt.Printf("run: kontango cluster join <url>\n")
		}
		return
	}

	var machine map[string]interface{}
	if err := json.Unmarshal(data, &machine); err != nil {
		if *jsonOut {
			fmt.Printf(`{"status":"error","error":"failed to parse machine.json: %v"}\n`, err)
		} else {
			fmt.Printf("cluster status: error reading machine.json: %v\n", err)
		}
		return
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]interface{}{
			"status":  "enrolled",
			"machine": machine,
		})
	} else {
		id, _ := machine["id"].(string)
		nick, _ := machine["nickname"].(string)
		if nick == "" {
			nick = id[:8]
		}
		fmt.Printf("cluster status: enrolled\n")
		fmt.Printf("  id:       %s\n", id)
		fmt.Printf("  nickname: %s\n", nick)
	}
}

func cmdClusterLeave(args []string) {
	fs := flag.NewFlagSet("cluster leave", flag.ExitOnError)
	purge := fs.Bool("purge-data", false, "delete local machine data")
	fs.Parse(args)

	log.SetFlags(0)
	fmt.Printf("cluster leave is not yet implemented\n")
	if *purge {
		fmt.Printf("  --purge-data flag recognized\n")
	}
}

func cmdClusterUpgrade(args []string) {
	fs := flag.NewFlagSet("cluster upgrade", flag.ExitOnError)
	version := fs.String("version", "", "target version to upgrade to")
	fs.Parse(args)

	log.SetFlags(0)
	fmt.Printf("cluster upgrade is not yet implemented\n")
	if *version != "" {
		fmt.Printf("  target version: %s\n", *version)
	}
}

func printClusterUsage() {
	fmt.Fprintf(os.Stderr, `usage: kontango cluster <subcommand> [flags]

subcommands:
  create         deploy a new HA controller cluster
  join <url>     enroll this machine in an existing cluster
  status         show cluster enrollment status
  leave          leave the cluster
  upgrade        upgrade cluster to a new version

examples:
  kontango cluster create --name prod --nodes 3
  kontango cluster join https://ctrl-1.example.com:1280
  kontango cluster status --json
`)
}
