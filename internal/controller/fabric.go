package controller

import (
	"fmt"
	"log"

	"github.com/KontangoOSS/TangoKore/internal/controller/clients"
)

// stepFabric creates Ziti services and policies
func stepFabric(cfg *Config) error {
	log.Println("step 11/13: configuring Ziti fabric services and policies...")

	// Skip on edge routers — only controllers create services
	if cfg.JoinMode {
		log.Println("  ⚠ skipping fabric (edge router)")
		return nil
	}

	// Skip in test mode - requires proper Ziti authentication
	if cfg.TestMode {
		log.Println("  ⚠ skipping fabric (test mode)")
		return nil
	}

	// Create Ziti client
	zitiClient, err := clients.NewZitiClient(
		fmt.Sprintf("127.0.0.1:%d", cfg.ZitiCtrlPort),
		cfg.ZitiAdminUser,
		cfg.ZitiAdminPass,
	)
	if err != nil {
		return fmt.Errorf("create ziti client: %w", err)
	}

	// Authenticate with Ziti controller
	if _, err := zitiClient.Authenticate(); err != nil {
		// Ziti may not be fully initialized yet - this is expected during initial bootstrap
		// The default authenticator will be set up later; skip fabric step for now
		log.Printf("  ⚠ Ziti controller not ready for authentication yet (expected during bootstrap): %v\n", err)
		log.Println("  ⚠ skipping fabric services (Ziti will be configured post-bootstrap)")
		return nil
	}

	// 1. Create services
	log.Println("  → creating services...")
	services := []string{
		"enrollment",      // Enrollment API
		"controller-api",  // Controller management API
		"bao-api",         // Bao vault API
		"nats-hub",        // NATS messaging hub
	}

	for _, svcName := range services {
		svcID, err := zitiClient.CreateService(svcName, nil, []string{svcName})
		if err != nil {
			return fmt.Errorf("create service %s: %w", svcName, err)
		}
		log.Printf("    ✓ %s (ID: %s)", svcName, svcID)
	}

	// 2. Create Service Edge Router Policy: all-routers (all services can use all routers)
	log.Println("  → creating service edge router policy...")
	if err := zitiClient.CreateServiceEdgeRouterPolicy("all-routers", nil, nil); err != nil {
		return fmt.Errorf("create serp: %w", err)
	}
	log.Println("    ✓ all-routers (services can use all routers)")

	// 3. Create Service Policies
	log.Println("  → creating service policies...")

	// Schmutz can bind enrollment service
	if err := zitiClient.CreateServicePolicy(
		"schmutz-binds-enrollment",
		"Bind",
		[]string{"#schmutz"},           // schmutz identity role
		[]string{"#enrollment"},        // enrollment service role
	); err != nil {
		return fmt.Errorf("create schmutz-binds-enrollment: %w", err)
	}
	log.Println("    ✓ schmutz-binds-enrollment")

	// Schmutz can bind Bao API
	if err := zitiClient.CreateServicePolicy(
		"schmutz-binds-bao",
		"Bind",
		[]string{"#schmutz"},
		[]string{"#bao-api"},
	); err != nil {
		return fmt.Errorf("create schmutz-binds-bao: %w", err)
	}
	log.Println("    ✓ schmutz-binds-bao")

	// Schmutz can bind controller API
	if err := zitiClient.CreateServicePolicy(
		"schmutz-binds-controller",
		"Bind",
		[]string{"#schmutz"},
		[]string{"#controller-api"},
	); err != nil {
		return fmt.Errorf("create schmutz-binds-controller: %w", err)
	}
	log.Println("    ✓ schmutz-binds-controller")

	// 4. Create Dial Policies: devices can dial enrollment
	log.Println("  → creating dial policies...")
	if err := zitiClient.CreateServicePolicy(
		"devices-dial-enrollment",
		"Dial",
		[]string{"#device-base"},       // device identities
		[]string{"#enrollment"},        // enrollment service
	); err != nil {
		return fmt.Errorf("create devices-dial-enrollment: %w", err)
	}
	log.Println("    ✓ devices-dial-enrollment")

	log.Println("  ✓ fabric configured")
	return nil
}
