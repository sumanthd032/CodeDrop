package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sumanthd032/codedrop/internal/client"
	"github.com/sumanthd032/codedrop/internal/crypto"
)

var (
	expire   string
	maxViews int
)

var pushCmd = &cobra.Command{
	Use:   "push [file_path]",
	Short: "Encrypt and push a file to the CodeDrop server",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		serverURL, _ := cmd.Flags().GetString("server")

		// 1. Open the file
		file, err := os.Open(filePath)
		if err != nil {
			fmt.Printf("Error opening file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil {
			fmt.Printf("Error reading file info: %v\n", err)
			os.Exit(1)
		}

		if fileInfo.IsDir() {
			fmt.Println("Error: CodeDrop currently only supports single files, not directories. Zip it first!")
			os.Exit(1)
		}

		// 2. Generate Encryption Key
		fmt.Println("Generating local encryption key...")
		key, encodedKey, err := crypto.GenerateKey()
		if err != nil {
			fmt.Printf("Error generating encryption key: %v\n", err)
			os.Exit(1)
		}

		// 3. Initialize API Client and Create Drop
		fmt.Println("Contacting CodeDrop Server...")
		api := client.NewAPIClient(serverURL)
		
		dropReq := client.CreateDropRequest{
			FileName:       filepath.Base(fileInfo.Name()),
			FileSize:       fileInfo.Size(),
			EncryptionSalt: "v1-aes-gcm", // Future-proofing in case we change algorithms
			ExpiresIn:      expire,
			MaxDownloads:   maxViews,
		}

		dropResp, err := api.CreateDrop(dropReq)
		if err != nil {
			fmt.Printf("Error creating drop: %v\n", err)
			os.Exit(1)
		}

		// 4. Chunk, Encrypt, and Upload
		const chunkSize = 4 * 1024 * 1024 // 4MB chunks
		buffer := make([]byte, chunkSize)
		chunkIndex := 0

		fmt.Printf("Uploading %s (Size: %d bytes)\n", dropReq.FileName, fileInfo.Size())

		for {
			// Read a chunk from the file
			bytesRead, err := file.Read(buffer)
			if err != nil && err != io.EOF {
				fmt.Printf("Error reading file: %v\n", err)
				os.Exit(1)
			}
			if bytesRead == 0 {
				break // End of file
			}

			// Encrypt the chunk
			plaintextChunk := buffer[:bytesRead]
			ciphertext, err := crypto.Encrypt(key, plaintextChunk)
			if err != nil {
				fmt.Printf("Error encrypting chunk %d: %v\n", chunkIndex, err)
				os.Exit(1)
			}

			// Upload the encrypted chunk
			fmt.Printf("   -> Pushing chunk %d...\n", chunkIndex)
			err = api.UploadChunk(dropResp.DropID, chunkIndex, ciphertext)
			if err != nil {
				fmt.Printf("Error uploading chunk %d: %v\n", chunkIndex, err)
				os.Exit(1)
			}

			chunkIndex++
		}

		// 5. Generate Output URL
		// The fragment (#) ensures the browser/CLI doesn't send the key to the server during the GET request.
		finalURL := fmt.Sprintf("%s/drop/%s#k=%s", serverURL, dropResp.DropID, encodedKey)

		fmt.Println("\nUpload Complete!")
		fmt.Println("--------------------------------------------------")
		fmt.Printf("Secure URL : %s\n", finalURL)
		fmt.Printf("Expires At : %s\n", dropResp.ExpiresAt.Local().Format("Jan 02, 2006 15:04:05 MST"))
		fmt.Printf("Max Views  : %d\n", maxViews)
		fmt.Println("--------------------------------------------------")
		fmt.Println("WARNING: Anyone with this URL can decrypt the file. Do not lose it; the key cannot be recovered.")
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().StringVarP(&expire, "expire", "e", "24h", "Time until the drop is permanently deleted (e.g., 30m, 24h)")
	pushCmd.Flags().IntVarP(&maxViews, "max-views", "m", 1, "Maximum number of times this drop can be downloaded")
}