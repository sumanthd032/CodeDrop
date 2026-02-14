package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// pullCmd represents the pull command
var pullCmd = &cobra.Command{
	Use:   "pull [url]",
	Short: "Download and decrypt a file from CodeDrop",
	Long: `Pulls a file from a CodeDrop URL. The file is downloaded in chunks, 
verified, and decrypted locally.

Example:
  codedrop pull http://localhost:8080/drop/1234-5678#k=secretkey`,
	Args: cobra.ExactArgs(1), // Requires exactly 1 argument (the URL)
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]

		fmt.Printf("Preparing to pull from: %s\n", url)
		
		// In future, we will add the download and decryption logic 
		fmt.Println("Pull command executed successfully.")
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
}