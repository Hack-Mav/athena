package cli

import (
	"github.com/athena/platform/pkg/config"
	"github.com/athena/platform/pkg/logger"
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

	return rootCmd
}

func newTemplateCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "template",
		Short: "Manage Arduino templates",
		Run: func(cmd *cobra.Command, args []string) {
			logger.Info("Template command placeholder")
		},
	}
}

func newProvisionCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "provision",
		Short: "Provision Arduino devices",
		Run: func(cmd *cobra.Command, args []string) {
			logger.Info("Provision command placeholder")
		},
	}
}

func newDeviceCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "device",
		Short: "Manage Arduino devices",
		Run: func(cmd *cobra.Command, args []string) {
			logger.Info("Device command placeholder")
		},
	}
}

func newNLPCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "plan",
		Short: "Generate implementation plan from natural language",
		Run: func(cmd *cobra.Command, args []string) {
			logger.Info("NLP planning command placeholder")
		},
	}
}