package cli

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sumanthd032/codedrop/internal/client"
	"github.com/sumanthd032/codedrop/internal/crypto"
)

var pullCmd = &cobra.Command{
	Use:   "pull [url]",
	Short: "Download and decrypt a file from CodeDrop",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputURL := args[0]

		// 1. Parse the URL
		parsedURL, err := url.Parse(inputURL)
		if err != nil {
			fmt.Printf("Invalid URL format: %v\n", err)
			os.Exit(1)
		}

		// Extract Base URL (e.g., http://localhost:8080)
		baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

		// Extract Drop ID from Path (e.g., /drop/1234-5678)
		pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
		if len(pathParts) != 2 || pathParts[0] != "drop" {
			fmt.Println("Invalid URL path. Expected format: http://host/drop/<id>#k=<key>")
			os.Exit(1)
		}
		dropID := pathParts[1]

		// Extract Key from Fragment (e.g., k=base64key)
		fragment := parsedURL.Fragment
		if !strings.HasPrefix(fragment, "k=") {
			fmt.Println("Missing decryption key in URL fragment (#k=...).")
			os.Exit(1)
		}
		encodedKey := strings.TrimPrefix(fragment, "k=")

		// 2. Decode the Key
		fmt.Println("Decoding decryption key...")
		key, err := crypto.DecodeKey(encodedKey)
		if err != nil {
			fmt.Printf("Invalid key: %v\n", err)
			os.Exit(1)
		}

		// 3. Fetch Metadata
		fmt.Println("🌐 Contacting server for metadata...")
		api := client.NewAPIClient(baseURL)
		
		meta, err := api.GetDropMetadata(dropID)
		if err != nil {
			fmt.Printf("Failed to fetch metadata: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Found file: %s (Size: %d bytes, Chunks: %d)\n", meta.FileName, meta.FileSize, meta.ChunkCount)

		// 4. Create Output File
		// We add "downloaded_" to the filename so we don't accidentally overwrite the original if testing locally
		outputFileName := "downloaded_" + meta.FileName
		outFile, err := os.Create(outputFileName)
		if err != nil {
			fmt.Printf("Failed to create output file: %v\n", err)
			os.Exit(1)
		}
		defer outFile.Close()

		// 5. Download and Decrypt Chunks
		fmt.Println("Downloading and decrypting chunks...")
		for i := 0; i < meta.ChunkCount; i++ {
			fmt.Printf("   -> Pulling chunk %d/%d...\n", i+1, meta.ChunkCount)
			
			// Download
			encryptedChunk, err := api.DownloadChunk(dropID, i)
			if err != nil {
				fmt.Printf("\nFailed to download chunk %d: %v\n", i, err)
				os.Remove(outputFileName) // Clean up partial file
				os.Exit(1)
			}

			// Decrypt
			plaintextChunk, err := crypto.Decrypt(key, encryptedChunk)
			if err != nil {
				fmt.Printf("\nDecryption failed on chunk %d! The data may be corrupted or the key is wrong: %v\n", i, err)
				os.Remove(outputFileName) // Clean up partial file
				os.Exit(1)
			}

			// Write to disk
			if _, err := outFile.Write(plaintextChunk); err != nil {
				fmt.Printf("\nFailed to write to file: %v\n", err)
				os.Remove(outputFileName)
				os.Exit(1)
			}
		}

		fmt.Println("\nDownload Complete!")
		fmt.Printf("Saved as: %s\n", outputFileName)
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
}