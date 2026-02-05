package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/tools/workbench"
	"github.com/spf13/cobra"
)

const (
	workbenchProviderFlag   = "provider"
	workbenchConfigFlag     = "config"
	workbenchConfigFileFlag = "config-file"
	workbenchListenFlag     = "listen"
	workbenchPageSizeFlag   = "page-size"
)

func newWorkbench() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workbench",
		Short: "Run the connector workbench for local development",
		Long: `The connector workbench provides a lightweight development environment 
for building and testing payment connectors without requiring Temporal, 
PostgreSQL, or other heavy infrastructure.

Features:
  - In-memory storage (no database required)
  - Multi-connector support - create multiple instances from the UI
  - Step-by-step execution of connector operations
  - HTTP API for triggering operations
  - Web UI for debugging and inspection
  - No Docker required

Examples:
  # Run workbench without any connector (create from UI)
  payments workbench

  # Run workbench with a pre-configured connector
  payments workbench --provider=stripe --config='{"apiKey":"sk_test_..."}'

  # Run workbench with config file
  payments workbench --provider=wise --config-file=./wise-config.json

  # List available providers
  payments workbench --list-providers`,
		RunE: runWorkbench,
	}

	cmd.Flags().StringP(workbenchProviderFlag, "p", "", "Connector provider name (e.g., stripe, wise, adyen)")
	cmd.Flags().StringP(workbenchConfigFlag, "c", "", "Connector configuration as JSON string")
	cmd.Flags().StringP(workbenchConfigFileFlag, "f", "", "Path to connector configuration JSON file")
	cmd.Flags().String(workbenchListenFlag, "127.0.0.1:8080", "HTTP server listen address")
	cmd.Flags().Int(workbenchPageSizeFlag, 25, "Page size for fetch operations")
	cmd.Flags().Bool("list-providers", false, "List available connector providers")

	return cmd
}

func runWorkbench(cmd *cobra.Command, args []string) error {
	// Check if listing providers
	listProviders, _ := cmd.Flags().GetBool("list-providers")
	if listProviders {
		return printProviders()
	}

	// Get optional provider and config for initial connector
	provider, _ := cmd.Flags().GetString(workbenchProviderFlag)
	configJSON, _ := cmd.Flags().GetString(workbenchConfigFlag)
	configFile, _ := cmd.Flags().GetString(workbenchConfigFileFlag)

	var connectorConfig json.RawMessage
	if provider != "" {
		provider = strings.ToLower(provider)

		if configFile != "" {
			data, err := os.ReadFile(configFile)
			if err != nil {
				return fmt.Errorf("failed to read config file: %w", err)
			}
			connectorConfig = data
		} else if configJSON != "" {
			connectorConfig = json.RawMessage(configJSON)
		} else {
			return fmt.Errorf("connector config is required when provider is specified (use --config or --config-file)")
		}

		// Validate JSON
		var validateJSON map[string]interface{}
		if err := json.Unmarshal(connectorConfig, &validateJSON); err != nil {
			return fmt.Errorf("invalid config JSON: %w", err)
		}
	}

	// Get other flags
	listenAddr, _ := cmd.Flags().GetString(workbenchListenFlag)
	pageSize, _ := cmd.Flags().GetInt(workbenchPageSizeFlag)

	// Create logger
	logger := logging.NewDefaultLogger(os.Stdout, true, false, false)

	// Create workbench config
	cfg := workbench.Config{
		ListenAddr:      listenAddr,
		EnableUI:        true,
		DefaultPageSize: pageSize,
	}

	// Create workbench
	wb, err := workbench.New(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create workbench: %w", err)
	}

	// Handle signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		cancel()
		wb.Stop(context.Background())
	}()

	// Start workbench
	if err := wb.Start(ctx); err != nil {
		return fmt.Errorf("failed to start workbench: %w", err)
	}

	// If provider was specified, create and install a connector
	if provider != "" {
		conn, err := wb.CreateConnector(ctx, workbench.CreateConnectorRequest{
			Provider: provider,
			Name:     "default",
			Config:   connectorConfig,
		})
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to create connector: %v", err))
		} else {
			logger.Info(fmt.Sprintf("Created connector instance: %s", conn.ID))

			// Install the connector
			if err := wb.InstallConnector(ctx, conn.ID); err != nil {
				logger.Error(fmt.Sprintf("Failed to install connector: %v", err))
			} else {
				logger.Info("Connector installed successfully")
			}
		}
	}

	// Wait for shutdown
	wb.Wait()

	return nil
}

func printProviders() error {
	configs := registry.GetConfigs(true) // include debug connectors

	fmt.Println("Available connector providers:")
	fmt.Println()

	for provider, config := range configs {
		fmt.Printf("  %s\n", provider)
		fmt.Printf("    Config parameters:\n")
		for paramName, param := range config {
			required := ""
			if param.Required {
				required = " (required)"
			}
			fmt.Printf("      - %s: %s%s\n", paramName, param.DataType, required)
		}
		fmt.Println()
	}

	return nil
}

func init() {
	// This will be added to root in NewRootCommand
}
