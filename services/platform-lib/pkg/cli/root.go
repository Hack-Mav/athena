package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/spf13/cobra"
)

// NewRootCommand creates the root CLI command
func NewRootCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "athena",
		Short: "ATHENA - Arduino Template Hub & Natural-Language Provisioning",
		Long: `ATHENA is a comprehensive platform that makes Arduino prototyping effortless 
by providing curated templates, unified configuration, natural language processing 
for firmware generation, and one-click provisioning with optional cloud connectivity 
and device management.`,
		Version: "0.1.0",
	}

	// Add subcommands
	rootCmd.AddCommand(newTemplateCommand(cfg, logger))
	rootCmd.AddCommand(newProvisionCommand(cfg, logger))
	rootCmd.AddCommand(newDeviceCommand(cfg, logger))
	rootCmd.AddCommand(newNLPCommand(cfg, logger))
	rootCmd.AddCommand(newProfileCommand(cfg, logger))
	rootCmd.AddCommand(newTelemetryCommand(cfg, logger))
	rootCmd.AddCommand(newOTACommand(cfg, logger))

	return rootCmd
}

func newTemplateCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage Arduino templates",
		Long:  "List, inspect, and select Arduino templates for projects",
	}

	cmd.AddCommand(newTemplateListCommand(cfg, logger))
	cmd.AddCommand(newTemplateInspectCommand(cfg, logger))
	cmd.AddCommand(newTemplateSelectCommand(cfg, logger))

	return cmd
}

func newTemplateListCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewServiceClient(cfg, logger)
			ctx := context.Background()

			templates, err := client.ListTemplates(ctx)
			if err != nil {
				return fmt.Errorf("failed to list templates: %w", err)
			}

			if len(templates) == 0 {
				fmt.Println("No templates found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "ID\tNAME\tCATEGORY\tTAGS\tDESCRIPTION\n")
			for _, tmpl := range templates {
				tags := ""
				if len(tmpl.Tags) > 0 {
					tags = tmpl.Tags[0]
					if len(tmpl.Tags) > 1 {
						tags += "+"
					}
				}
				description := tmpl.Description
				if len(description) > 50 {
					description = description[:47] + "..."
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					tmpl.ID, tmpl.Name, tmpl.Category, tags, description)
			}
			w.Flush()

			return nil
		},
	}
}

func newTemplateInspectCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	var version string
	cmd := &cobra.Command{
		Use:   "inspect [id]",
		Short: "Inspect a template by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewServiceClient(cfg, logger)
			ctx := context.Background()

			templateID := args[0]
			template, err := client.GetTemplate(ctx, templateID)
			if err != nil {
				return fmt.Errorf("failed to get template: %w", err)
			}

			fmt.Printf("ID: %s\n", template.ID)
			fmt.Printf("Name: %s\n", template.Name)
			fmt.Printf("Description: %s\n", template.Description)
			fmt.Printf("Category: %s\n", template.Category)
			fmt.Printf("Created: %s\n", template.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Updated: %s\n", template.UpdatedAt.Format("2006-01-02 15:04:05"))

			if len(template.Tags) > 0 {
				fmt.Printf("Tags: %s\n", template.Tags)
			}

			if len(template.Metadata) > 0 {
				fmt.Println("\nMetadata:")
				for k, v := range template.Metadata {
					fmt.Printf("  %s: %s\n", k, v)
				}
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&version, "version", "latest", "Template version")
	return cmd
}

func newTemplateSelectCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	var version string
	cmd := &cobra.Command{
		Use:   "select [id]",
		Short: "Select a template for the current profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewServiceClient(cfg, logger)
			pm, err := NewProfileManager()
			if err != nil {
				return fmt.Errorf("failed to initialize profile manager: %w", err)
			}

			templateID := args[0]
			ctx := context.Background()

			// Verify template exists
			template, err := client.GetTemplate(ctx, templateID)
			if err != nil {
				return fmt.Errorf("failed to get template: %w", err)
			}

			// Update current profile
			updates := map[string]interface{}{
				"template_id":      templateID,
				"template_version": version,
			}
			if err := pm.UpdateCurrentProfile(updates); err != nil {
				return fmt.Errorf("failed to update profile: %w", err)
			}

			fmt.Printf("Selected template '%s' (%s) for current profile\n",
				template.Name, template.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&version, "version", "latest", "Template version")
	return cmd
}

func newProvisionCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provision",
		Short: "Provision Arduino devices",
		Long:  "Compile and flash Arduino devices using selected templates",
	}

	cmd.AddCommand(newProvisionCompileCommand(cfg, logger))
	cmd.AddCommand(newProvisionFlashCommand(cfg, logger))

	return cmd
}

func newProvisionCompileCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	var board string
	var params map[string]string
	cmd := &cobra.Command{
		Use:   "compile",
		Short: "Compile the selected template",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewServiceClient(cfg, logger)
			pm, err := NewProfileManager()
			if err != nil {
				return fmt.Errorf("failed to initialize profile manager: %w", err)
			}

			profile, err := pm.GetCurrentProfile()
			if err != nil {
				return fmt.Errorf("failed to get current profile: %w", err)
			}

			if profile.TemplateID == "" {
				return fmt.Errorf("no template selected in current profile. Use 'athena template select' first")
			}

			// Use board from flag or profile
			targetBoard := board
			if targetBoard == "" {
				targetBoard = profile.Board
			}
			if targetBoard == "" {
				return fmt.Errorf("no board specified. Use --board flag or set it in profile")
			}

			ctx := context.Background()
			req := &CompileRequest{
				TemplateID: profile.TemplateID,
				Board:      targetBoard,
				Parameters: params,
			}

			resp, err := client.Compile(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to compile: %w", err)
			}

			fmt.Printf("Compilation completed. Artifact ID: %s\n", resp.ArtifactID)
			fmt.Printf("Status: %s\n", resp.Status)
			if resp.Message != "" {
				fmt.Printf("Message: %s\n", resp.Message)
			}

			// Store artifact ID in profile for flashing
			updates := map[string]interface{}{
				"parameters": map[string]string{"last_artifact_id": resp.ArtifactID},
			}
			if err := pm.UpdateCurrentProfile(updates); err != nil {
				logger.Warnf("Failed to save artifact ID to profile: %v", err)
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&board, "board", "", "Arduino board (e.g., arduino:avr:uno)")
	cmd.Flags().StringToStringVar(&params, "param", nil, "Template parameters (key=value)")
	return cmd
}

func newProvisionFlashCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	var port string
	var board string
	var artifactID string
	cmd := &cobra.Command{
		Use:   "flash",
		Short: "Flash firmware to a device",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewServiceClient(cfg, logger)
			pm, err := NewProfileManager()
			if err != nil {
				return fmt.Errorf("failed to initialize profile manager: %w", err)
			}

			profile, err := pm.GetCurrentProfile()
			if err != nil {
				return fmt.Errorf("failed to get current profile: %w", err)
			}

			// Use artifact ID from flag, profile, or error
			targetArtifactID := artifactID
			if targetArtifactID == "" {
				if profile.Parameters != nil && profile.Parameters["last_artifact_id"] != "" {
					targetArtifactID = profile.Parameters["last_artifact_id"]
				}
			}
			if targetArtifactID == "" {
				return fmt.Errorf("no artifact ID specified. Use --artifact-id flag or run 'athena provision compile' first")
			}

			// Use board from flag or profile
			targetBoard := board
			if targetBoard == "" {
				targetBoard = profile.Board
			}
			if targetBoard == "" {
				return fmt.Errorf("no board specified. Use --board flag or set it in profile")
			}

			// Use port from flag or profile
			targetPort := port
			if targetPort == "" {
				targetPort = profile.Port
			}
			if targetPort == "" {
				return fmt.Errorf("no port specified. Use --port flag or set it in profile")
			}

			ctx := context.Background()
			req := &FlashRequest{
				Port:       targetPort,
				Board:      targetBoard,
				ArtifactID: targetArtifactID,
			}

			resp, err := client.Flash(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to flash: %w", err)
			}

			if resp.Success {
				fmt.Printf("Successfully flashed firmware to %s\n", targetPort)
			} else {
				fmt.Printf("Flash failed: %s\n", resp.Message)
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&port, "port", "", "Serial port (e.g., COM3, /dev/ttyUSB0)")
	cmd.Flags().StringVar(&board, "board", "", "Arduino board (e.g., arduino:avr:uno)")
	cmd.Flags().StringVar(&artifactID, "artifact-id", "", "Artifact ID to flash")
	return cmd
}

func newDeviceCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "device",
		Short: "Manage Arduino devices",
		Long:  "List, inspect, and manage registered Arduino devices",
	}

	cmd.AddCommand(newDeviceListCommand(cfg, logger))
	cmd.AddCommand(newDeviceGetCommand(cfg, logger))

	return cmd
}

func newDeviceListCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all registered devices",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewServiceClient(cfg, logger)
			ctx := context.Background()

			devices, err := client.ListDevices(ctx)
			if err != nil {
				return fmt.Errorf("failed to list devices: %w", err)
			}

			if len(devices) == 0 {
				fmt.Println("No devices found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "ID\tNAME\tBOARD\tSTATUS\tUPDATED\n")
			for _, dev := range devices {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					dev.ID, dev.Name, dev.Board, dev.Status,
					dev.UpdatedAt.Format("2006-01-02 15:04"))
			}
			w.Flush()

			return nil
		},
	}
}

func newDeviceGetCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "get [id]",
		Short: "Get device details by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewServiceClient(cfg, logger)
			ctx := context.Background()

			deviceID := args[0]
			device, err := client.GetDevice(ctx, deviceID)
			if err != nil {
				return fmt.Errorf("failed to get device: %w", err)
			}

			fmt.Printf("ID: %s\n", device.ID)
			fmt.Printf("Name: %s\n", device.Name)
			fmt.Printf("Board: %s\n", device.Board)
			fmt.Printf("Status: %s\n", device.Status)
			fmt.Printf("Created: %s\n", device.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Updated: %s\n", device.UpdatedAt.Format("2006-01-02 15:04:05"))

			if len(device.Metadata) > 0 {
				fmt.Println("\nMetadata:")
				for k, v := range device.Metadata {
					fmt.Printf("  %s: %s\n", k, v)
				}
			}

			return nil
		},
	}
}

func newNLPCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Generate implementation plan from natural language",
		Long:  "Use natural language to describe your project and get an implementation plan",
	}

	cmd.AddCommand(newNLPGenerateCommand(cfg, logger))

	return cmd
}

func newNLPGenerateCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "generate [description]",
		Short: "Generate a plan from natural language description",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			description := args[0]
			for i := 1; i < len(args); i++ {
				description += " " + args[i]
			}

			// For now, we'll create a simple plan without calling the NLP service
			// In a full implementation, this would call the NLP service
			fmt.Printf("Generating plan for: %s\n\n", description)

			fmt.Println("This is a placeholder implementation.")
			fmt.Println("In a full implementation, this would:")
			fmt.Println("1. Parse the natural language description")
			fmt.Println("2. Extract requirements (sensors, actuators, communication)")
			fmt.Println("3. Select appropriate templates")
			fmt.Println("4. Generate wiring diagram")
			fmt.Println("5. Create bill of materials")
			fmt.Println("6. Provide step-by-step instructions")

			return nil
		},
	}
}

func newTelemetryCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "telemetry",
		Short: "Access device telemetry data",
		Long:  "Stream and monitor telemetry data from devices",
	}

	cmd.AddCommand(newTelemetryMetricsCommand(cfg, logger))

	return cmd
}

func newTelemetryMetricsCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Get telemetry metrics for a device",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewServiceClient(cfg, logger)
			ctx := context.Background()

			deviceID, err := cmd.Flags().GetString("device")
			if err != nil {
				return fmt.Errorf("failed to get device flag: %w", err)
			}
			if deviceID == "" {
				return fmt.Errorf("device ID is required. Use --device flag")
			}

			metrics, err := client.GetTelemetryMetrics(ctx, deviceID)
			if err != nil {
				return fmt.Errorf("failed to get telemetry metrics: %w", err)
			}

			fmt.Printf("Device ID: %s\n", metrics.DeviceID)
			fmt.Printf("Timestamp: %s\n", metrics.Timestamp.Format("2006-01-02 15:04:05"))

			if len(metrics.Metrics) > 0 {
				fmt.Println("\nMetrics:")
				for k, v := range metrics.Metrics {
					fmt.Printf("  %s: %v\n", k, v)
				}
			} else {
				fmt.Println("No metrics available.")
			}

			return nil
		},
	}
	cmd.Flags().String("device", "", "Device ID")
	return cmd
}

func newOTACommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ota",
		Short: "Manage OTA updates",
		Long:  "Over-the-air update management for devices",
	}

	cmd.AddCommand(newOTAReleasesCommand(cfg, logger))

	return cmd
}

func newOTAReleasesCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "releases",
		Short: "List available OTA releases",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := NewServiceClient(cfg, logger)
			ctx := context.Background()

			releases, err := client.ListReleases(ctx)
			if err != nil {
				return fmt.Errorf("failed to list releases: %w", err)
			}

			if len(releases) == 0 {
				fmt.Println("No releases found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "ID\tVERSION\tDESCRIPTION\tCREATED\n")
			for _, rel := range releases {
				description := rel.Description
				if len(description) > 40 {
					description = description[:37] + "..."
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					rel.ID, rel.Version, description,
					rel.CreatedAt.Format("2006-01-02 15:04"))
			}
			w.Flush()

			return nil
		},
	}
}

func newProfileCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage CLI profiles",
		Long:  "Manage CLI profiles for template selection and device configuration",
	}

	cmd.AddCommand(newProfileShowCommand(cfg, logger))
	cmd.AddCommand(newProfileListCommand(cfg, logger))
	cmd.AddCommand(newProfileUseCommand(cfg, logger))
	cmd.AddCommand(newProfileCreateCommand(cfg, logger))
	cmd.AddCommand(newProfileDeleteCommand(cfg, logger))

	return cmd
}

func newProfileShowCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			pm, err := NewProfileManager()
			if err != nil {
				return fmt.Errorf("failed to initialize profile manager: %w", err)
			}

			profile, err := pm.GetCurrentProfile()
			if err != nil {
				return fmt.Errorf("failed to get current profile: %w", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "Name:\t%s\n", profile.Name)
			fmt.Fprintf(w, "Template ID:\t%s\n", profile.TemplateID)
			fmt.Fprintf(w, "Template Version:\t%s\n", profile.TemplateVersion)
			fmt.Fprintf(w, "Board:\t%s\n", profile.Board)
			fmt.Fprintf(w, "Port:\t%s\n", profile.Port)
			w.Flush()

			if len(profile.Parameters) > 0 {
				fmt.Println("\nParameters:")
				for k, v := range profile.Parameters {
					fmt.Printf("  %s: %s\n", k, v)
				}
			}

			return nil
		},
	}
}

func newProfileListCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			pm, err := NewProfileManager()
			if err != nil {
				return fmt.Errorf("failed to initialize profile manager: %w", err)
			}

			profiles := pm.ListProfiles()
			current, _ := pm.GetCurrentProfile()

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "NAME\tTEMPLATE\tBOARD\tPORT\tCURRENT\n")
			for name, profile := range profiles {
				currentMarker := ""
				if current != nil && current.Name == name {
					currentMarker = "*"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					name, profile.TemplateID, profile.Board, profile.Port, currentMarker)
			}
			w.Flush()

			return nil
		},
	}
}

func newProfileUseCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "use [name]",
		Short: "Switch to a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pm, err := NewProfileManager()
			if err != nil {
				return fmt.Errorf("failed to initialize profile manager: %w", err)
			}

			name := args[0]
			if err := pm.SetCurrentProfile(name); err != nil {
				return fmt.Errorf("failed to switch profile: %w", err)
			}

			fmt.Printf("Switched to profile: %s\n", name)
			return nil
		},
	}
}

func newProfileCreateCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pm, err := NewProfileManager()
			if err != nil {
				return fmt.Errorf("failed to initialize profile manager: %w", err)
			}

			name := args[0]
			profile := Profile{
				Name: name,
			}

			if err := pm.CreateProfile(name, profile); err != nil {
				return fmt.Errorf("failed to create profile: %w", err)
			}

			fmt.Printf("Created profile: %s\n", name)
			return nil
		},
	}
}

func newProfileDeleteCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pm, err := NewProfileManager()
			if err != nil {
				return fmt.Errorf("failed to initialize profile manager: %w", err)
			}

			name := args[0]
			if err := pm.DeleteProfile(name); err != nil {
				return fmt.Errorf("failed to delete profile: %w", err)
			}

			fmt.Printf("Deleted profile: %s\n", name)
			return nil
		},
	}
}
