package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func cmdStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "output as JSON")
	identityPath := fs.String("identity", "/opt/kontango/identity.json", "path to identity")
	fs.Parse(args)

	hostname, _ := os.Hostname()

	status := map[string]interface{}{
		"hostname": hostname,
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
		"version":  version,
	}

	// Load machine.json
	dir := (*identityPath)[:strings.LastIndex(*identityPath, "/")]
	if data, err := os.ReadFile(dir + "/machine.json"); err == nil {
		var rec map[string]interface{}
		if json.Unmarshal(data, &rec) == nil {
			for k, v := range rec {
				status[k] = v
			}
		}
	}

	// Check services (systemctl only available on Linux)
	services := []string{"kontango-tunnel", "kontango-agent", "kontango-portal", "kontango-caddy", "kontango-bao-agent"}
	svcStatus := map[string]string{}
	for _, svc := range services {
		if runtime.GOOS == "linux" {
			out, err := exec.Command("systemctl", "is-active", svc+".service").Output()
			if err != nil {
				svcStatus[svc] = "inactive"
			} else {
				svcStatus[svc] = strings.TrimSpace(string(out))
			}
		} else {
			// On non-Linux platforms, we can't check systemctl, return unknown
			svcStatus[svc] = "unknown"
		}
	}
	status["services"] = svcStatus

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(status)
	} else {
		nick, _ := status["nickname"].(string)
		mode, _ := status["mode"].(string)
		if nick == "" {
			nick = hostname
		}
		if mode == "" {
			mode = "unknown"
		}
		fmt.Printf("kontango %s\n", version)
		fmt.Printf("  node:     %s\n", nick)
		fmt.Printf("  mode:     %s\n", mode)
		fmt.Printf("  host:     %s (%s/%s)\n", hostname, runtime.GOOS, runtime.GOARCH)
		fmt.Println("  services:")
		for _, svc := range services {
			state := svcStatus[svc]
			marker := "  "
			if state == "active" {
				marker = "OK"
			}
			fmt.Printf("    [%s] %s\n", marker, svc)
		}
	}
}
