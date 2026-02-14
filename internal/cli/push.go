package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Variables to hold our flag values
var (
	expire   string
	maxViews int
)

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push [file_path]",
	Short: "Encrypt and push a file to the CodeDrop server",
	Long: `Pushes a file to the CodeDrop server. The file is chunked and 
encrypted client-side before transmission. 

Example:
  codedrop push my_secret.zip --expire 30m --max-views 1`,
	Args: cobra.ExactArgs(1), // We explicitly require exactly 1 argument (the file path)
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		serverURL, _ := cmd.Flags().GetString("server")

		fmt.Printf("Preparing to push: %s\n", filePath)
		fmt.Printf("Server: %s\n", serverURL)
		fmt.Printf("Expiry: %s\n", expire)
		fmt.Printf("Max Views: %d\n", maxViews)
		
		// In future, we will add the actual encryption and upload logic 
		fmt.Println("Push command executed successfully.")
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)

	// Define flags specific to the push command
	pushCmd.Flags().StringVarP(&expire, "expire", "e", "24h", "Time until the drop is permanently deleted (e.g., 30m, 24h)")
	pushCmd.Flags().IntVarP(&maxViews, "max-views", "m", 1, "Maximum number of times this drop can be downloaded")
}