package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sumanthd032/codedrop/internal/client"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "View CodeDrop system health and storage metrics",
	Run: func(cmd *cobra.Command, args []string) {
		serverURL, _ := cmd.Flags().GetString("server")

		fmt.Println("Fetching system metrics from:", serverURL)
		
		api := client.NewAPIClient(serverURL)
		stats, err := api.GetStats()
		if err != nil {
			fmt.Printf("Failed to fetch stats: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n=== CodeDrop Observability ===")
		fmt.Printf("Active Drops   : %d\n", stats.ActiveDrops)
		fmt.Printf("Unique Chunks  : %d\n", stats.TotalChunks)
		fmt.Printf("Storage Used   : %s\n", formatBytes(stats.StorageUsed))
		
		// The flex metric
		if stats.StorageSaved > 0 {
			fmt.Printf("Storage Saved  : %s (De-duplication active!)\n", formatBytes(stats.StorageSaved))
		} else {
			fmt.Printf("Storage Saved  : 0 B\n")
		}
		fmt.Println("=================================")
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}

// formatBytes converts bytes to a human-readable string
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}