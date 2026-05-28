// Package main provides the entry point for scrapedoctl.
package main

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/mr-pmillz/scrapedoctl/internal/version"
)

type CLIMetadata struct {
	Name     string            `json:"name"`
	Version  string            `json:"version"`
	Commands []CommandMetadata `json:"commands"`
	Config   ConfigMetadata    `json:"config"`
}

type CommandMetadata struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Usage       string         `json:"usage"`
	Flags       []FlagMetadata `json:"flags,omitempty"`
}

type FlagMetadata struct {
	Name        string `json:"name"`
	Shorthand   string `json:"shorthand,omitempty"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
}

type ConfigMetadata struct {
	ActiveProfile string `json:"active_profile"`
	ConfigPath    string `json:"config_path"`
}

func newMetadataCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "metadata",
		Short:  "Output CLI metadata in JSON format",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root := cmd.Root()
			metadata := CLIMetadata{
				Name:    root.Name(),
				Version: version.Version,
				Config: ConfigMetadata{
					ActiveProfile: cfg.ActiveProfile,
					ConfigPath:    configPath,
				},
			}

			for _, c := range root.Commands() {
				if c.Hidden {
					continue
				}
				cmdMeta := CommandMetadata{
					Name:        c.Name(),
					Description: c.Short,
					Usage:       c.UseLine(),
				}

				c.Flags().VisitAll(func(f *pflag.Flag) {
					cmdMeta.Flags = append(cmdMeta.Flags, FlagMetadata{
						Name:        f.Name,
						Shorthand:   f.Shorthand,
						Type:        f.Value.Type(),
						Default:     f.DefValue,
						Description: f.Usage,
					})
				})

				metadata.Commands = append(metadata.Commands, cmdMeta)
			}

			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(metadata)
		},
	}
}
