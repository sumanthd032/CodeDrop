package cli

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "codedrop",
	Short: "A secure, ephemeral file handoff tool",
	Long: `CodeDrop enables developers to securely and temporarily hand off 
code artifacts via a CLI using client-side encryption and strict lifecycle policies.
`,
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	serverDefault := os.Getenv("CODEDROP_SERVER")
	if serverDefault == "" {
		serverDefault = "http://localhost:8080"
	}
	rootCmd.PersistentFlags().StringP("server", "s", serverDefault, "CodeDrop API server URL")
}