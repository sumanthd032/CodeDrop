package test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// Configuration
const (
	cliPath    = "../codedrop_test_bin" // Temporary binary name
	serverURL  = "http://localhost:8080"
	sourceMain = "../cmd/cli/main.go"
)

// TestMain handles setup (building the CLI) and teardown
func TestMain(m *testing.M) {
	fmt.Println("Building CLI binary for testing...")

	// Build the CLI tool from source
	cmd := exec.Command("go", "build", "-o", cliPath, sourceMain)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to build CLI: %v\n", err)
		os.Exit(1)
	}

	// Run the tests
	code := m.Run()

	// Cleanup binary
	os.Remove(cliPath)
	os.Exit(code)
}

func TestEndToEndFlows(t *testing.T) {
	// Ensure server is reachable
	if !isServerUp() {
		t.Fatalf("Server is not running at %s. Please start it via 'go run cmd/server/main.go'", serverURL)
	}

	t.Run("Lifecycle: Push, Pull, and Integrity Check", func(t *testing.T) {
		// 1. Create dummy file
		filename := "test_payload.txt"
		content := []byte("This is a rigorous integration test payload.")
		createFile(t, filename, content)
		defer os.Remove(filename)
		defer os.Remove("downloaded_" + filename)

		// 2. Push
		output := runCLI(t, "push", filename, "--expire", "10m", "--max-views", "2")
		url := extractURL(t, output)

		t.Logf("   -> Pushed. URL: %s", url)

		// 3. Pull
		runCLI(t, "pull", url)

		// 4. Verify Integrity
		originalHash := hashFile(t, filename)
		downloadedHash := hashFile(t, "downloaded_"+filename)

		if originalHash != downloadedHash {
			t.Fatalf("Hash Mismatch!\nOriginal: %s\nDownload: %s", originalHash, downloadedHash)
		}
	})

	t.Run("Security: Atomic Download Limits", func(t *testing.T) {
		filename := "test_limit.txt"
		createFile(t, filename, []byte("Limit Test"))
		defer os.Remove(filename)
		defer os.Remove("downloaded_" + filename)

		// Push with strict limit of 1
		output := runCLI(t, "push", filename, "--max-views", "1")
		url := extractURL(t, output)

		// Attempt 1 (Success)
		runCLI(t, "pull", url)

		// Attempt 2 (Should Fail)
		cmd := exec.Command(cliPath, "pull", url)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		err := cmd.Run()

		// We expect an error here!
		if err == nil {
			t.Fatalf("Security Failure: CLI allowed a 2nd download on a 1-view drop!")
		}

		if !strings.Contains(out.String(), "reached its download limit") {
			t.Fatalf("Unexpected error message: %s", out.String())
		}
	})

	t.Run("Optimization: CAS Deduplication", func(t *testing.T) {
		filename := "test_cas.bin"
		// Create 5MB random file
		cmd := exec.Command("dd", "if=/dev/urandom", "of="+filename, "bs=1M", "count=5")
		cmd.Run()
		defer os.Remove(filename)

		// Push Twice
		runCLI(t, "push", filename)
		runCLI(t, "push", filename) // Should trigger CAS

		// Check Stats
		stats := runCLI(t, "stats")

		// We expect Storage Saved to be > 0
		if strings.Contains(stats, "Storage Saved  : 0 B") {
			t.Errorf("Optimization Failure: CAS did not save any storage.")
		} else if !strings.Contains(stats, "Storage Saved") {
			t.Errorf("Failed to parse stats output: %s", stats)
		}
	})

	t.Run("Security: Zero-Knowledge Key Tampering", func(t *testing.T) {
		filename := "test_tamper.txt"
		createFile(t, filename, []byte("Secret Data"))
		defer os.Remove(filename)

		output := runCLI(t, "push", filename)
		url := extractURL(t, output)

		// Tamper with the key (last char)
		badURL := url[:len(url)-1] + "X"

		// Try to pull
		cmd := exec.Command(cliPath, "pull", badURL)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		err := cmd.Run()

		if err == nil {
			t.Fatalf("Cryptography Failure: Decryption succeeded with a wrong key!")
		}

		outputStr := out.String()
		// We accept two types of failures here, both mean the user was blocked:
		// 1. "Invalid key" -> The tamper made the base64 invalid or wrong length (Your error).
		// 2. "Decryption failed" -> The key was valid format but wrong bits (GCM check failed).
		if !strings.Contains(outputStr, "Decryption failed") && !strings.Contains(outputStr, "Invalid key") {
			t.Fatalf("Unexpected error on tamper test: %s", outputStr)
		}
	})
}

// --- Helpers ---

func runCLI(t *testing.T, args ...string) string {
	cmd := exec.Command(cliPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI Command Failed: codedrop %s\nError: %v\nOutput: %s", strings.Join(args, " "), err, string(output))
	}
	return string(output)
}

func createFile(t *testing.T, name string, content []byte) {
	if err := os.WriteFile(name, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
}

func hashFile(t *testing.T, path string) string {
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Failed to open file for hashing: %v", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatalf("Failed to hash file: %v", err)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func extractURL(t *testing.T, output string) string {
	// Regex to find: Secure URL : http://...
	re := regexp.MustCompile(`Secure URL\s+:\s+(http://[^\s]+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		t.Fatalf("Failed to extract URL from output:\n%s", output)
	}
	return matches[1]
}

func isServerUp() bool {
	// Simple curl check
	cmd := exec.Command("curl", "-s", serverURL+"/health")
	err := cmd.Run()
	return err == nil
}
