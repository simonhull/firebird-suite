package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/simonhull/firebird-suite/firebird/internal/helpers"
	"github.com/simonhull/firebird-suite/fledge/output"
	"github.com/spf13/cobra"
)

// RealtimeCmd returns the realtime command group
func RealtimeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "realtime",
		Short: "Manage real-time features",
		Long: `Manage WebSocket and real-time infrastructure in your Firebird project.

Commands:
  init - Initialize real-time support (wire up main.go, routes.go, config)`,
	}

	cmd.AddCommand(realtimeInitCmd())
	return cmd
}

func realtimeInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize real-time features",
		Long: `Initialize WebSocket support in your Firebird project.

This command:
- Detects real-time configuration from internal/schemas/
- Updates cmd/server/main.go with EventBus initialization
- Updates internal/handlers/routes.go with /ws endpoint
- Updates config/firebird.yml with real-time settings

Normally this happens automatically when you generate your first realtime
resource. Use this command to manually initialize or re-initialize with --force.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := initRealtime(force); err != nil {
				output.Error(err.Error())
				os.Exit(1)
			}
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force re-initialization even if already initialized")

	return cmd
}

func initRealtime(force bool) error {
	// Check if in Firebird project
	if _, err := os.Stat("firebird.yml"); os.IsNotExist(err) {
		return fmt.Errorf("not in a Firebird project root (firebird.yml not found)")
	}

	// Check if already initialized (unless --force)
	if helpers.IsRealtimeInitialized() && !force {
		output.Info("✓ Realtime already initialized")
		output.Info("  Use --force to re-initialize")
		output.Info("")

		hasRealtime, backend, _ := helpers.DetectRealtimeFromSchemas()
		if hasRealtime {
			output.Info("Current configuration:")
			output.Step(fmt.Sprintf("Backend: %s", backend))
			output.Step("WebSocket endpoint: /ws")
		}

		return nil
	}

	// Detect realtime from schemas
	hasRealtime, backend, natsURL := helpers.DetectRealtimeFromSchemas()

	if !hasRealtime {
		output.Error("No schemas with realtime enabled found")
		output.Info("")
		output.Info("To enable real-time features:")
		output.Step("1. Add to your schema (.firebird.yml):")
		output.Info("   realtime:")
		output.Info("     enabled: true")
		output.Info("     backend: memory  # or 'nats'")
		output.Step("2. Run: firebird generate resource YourModel")
		output.Info("")
		return nil
	}

	output.Info(fmt.Sprintf("✓ Detected real-time configuration (backend: %s)", backend))

	// Get module path
	modulePath, err := helpers.GetModulePath()
	if err != nil {
		return fmt.Errorf("getting module path: %w", err)
	}

	// Update main.go
	mainPath := filepath.Join("cmd", "server", "main.go")
	if err := helpers.UpdateMainGo(mainPath, modulePath, backend, natsURL); err != nil {
		return fmt.Errorf("updating main.go: %w", err)
	}
	output.Success("✓ Updated cmd/server/main.go")

	// Update routes.go
	routesPath := filepath.Join("internal", "handlers", "routes.go")
	if err := helpers.UpdateRoutesGo(routesPath, modulePath); err != nil {
		return fmt.Errorf("updating routes.go: %w", err)
	}
	output.Success("✓ Updated internal/handlers/routes.go")

	// Update config
	if err := helpers.UpdateConfigWithRealtime(backend, natsURL, true); err != nil {
		return fmt.Errorf("updating config: %w", err)
	}
	output.Success("✓ Updated config/firebird.yml")

	output.Info("")
	output.Success("🎉 Realtime initialized successfully!")
	output.Info("")
	output.Info("Next steps:")
	output.Step("1. Build: go build -o app cmd/server/main.go")
	output.Step("2. Run: ./app")
	output.Step("3. Test: curl -i http://localhost:8080/ws")
	output.Info("")
	output.Info("Expected response: '426 Upgrade Required' (WebSocket ready)")

	return nil
}
