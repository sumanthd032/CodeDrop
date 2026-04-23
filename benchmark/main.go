// benchmark/main.go
//
// Measures two real numbers for your resume:
//   1. Redis Lua (single round-trip) vs naive two-command latency
//   2. CAS deduplication savings % from the live stats endpoint
//
// Usage:
//   docker-compose up -d          # start Postgres, Redis, MinIO
//   go run ./cmd/server &         # start server
//   go run ./benchmark            # run this

package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sumanthd032/codedrop/internal/client"
)

const (
	serverURL  = "http://localhost:8080"
	redisAddr  = "localhost:6379"
	iterations = 2000 // round-trips per approach
	uploadReps = 5    // how many times to upload the same file
	chunkSize  = 4 * 1024 * 1024 // 4 MB
)

func main() {
	fmt.Println("=== CodeDrop Benchmark ===")
	fmt.Println()

	benchRedis()
	fmt.Println()
	benchDedup()
}

// ─── 1. Redis latency: Lua vs naive ──────────────────────────────────────────

func benchRedis() {
	fmt.Println("--- Redis: Lua atomic vs naive two-command ---")

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	ctx := context.Background()

	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Printf("  Redis not reachable (%v) — skipping\n", err)
		return
	}
	defer rdb.Close()

	script := redis.NewScript(`
		local key = KEYS[1]
		local max = tonumber(ARGV[1])
		local cur = redis.call("INCR", key)
		if cur == 1 then redis.call("EXPIRE", key, 86400) end
		if cur > max then return 0 end
		return 1
	`)

	// Warm up
	for i := 0; i < 50; i++ {
		k := fmt.Sprintf("bench:warmup:%d", i)
		script.Run(ctx, rdb, []string{k}, 999999)
		rdb.Del(ctx, k)
	}

	// Lua benchmark
	luaStart := time.Now()
	for i := 0; i < iterations; i++ {
		k := fmt.Sprintf("bench:lua:%d", i)
		script.Run(ctx, rdb, []string{k}, 999999)
		rdb.Del(ctx, k)
	}
	luaTotal := time.Since(luaStart)
	luaPerOp := luaTotal / time.Duration(iterations)

	// Naive benchmark (INCR + EXPIRE as separate commands — TOCTOU window between them)
	naiveStart := time.Now()
	for i := 0; i < iterations; i++ {
		k := fmt.Sprintf("bench:naive:%d", i)
		val, _ := rdb.Incr(ctx, k).Result()
		_ = val > 999999
		rdb.Expire(ctx, k, 24*time.Hour)
		rdb.Del(ctx, k)
	}
	naiveTotal := time.Since(naiveStart)
	naivePerOp := naiveTotal / time.Duration(iterations)

	reduction := 100.0 * float64(naivePerOp-luaPerOp) / float64(naivePerOp)

	fmt.Printf("  Lua (atomic, 1 round-trip) : %v/op\n", luaPerOp)
	fmt.Printf("  Naive (2 commands)         : %v/op\n", naivePerOp)
	fmt.Printf("  Latency reduction          : %.0f%%\n", reduction)
	fmt.Println()
	fmt.Printf("  >> RESUME NUMBER: \"reduced per-download enforcement latency by ~%.0f%%\n", reduction)
	fmt.Printf("     vs a two-command approach, eliminating TOCTOU races atomically\"\n")
}

// ─── 2. CAS dedup savings ─────────────────────────────────────────────────────

func benchDedup() {
	fmt.Println("--- CAS deduplication savings ---")

	api := client.NewAPIClient(serverURL)

	// Check server is up
	before, err := api.GetStats()
	if err != nil {
		fmt.Printf("  Server not reachable (%v) — skipping\n  Start with: go run ./cmd/server\n", err)
		return
	}

	// Generate one random 4 MB chunk (simulates a real file)
	chunk := make([]byte, chunkSize)
	if _, err := rand.Read(chunk); err != nil {
		fmt.Fprintf(os.Stderr, "rand error: %v\n", err)
		return
	}

	fmt.Printf("  Uploading the same 4 MB chunk %d times...\n", uploadReps)

	for i := 0; i < uploadReps; i++ {
		drop, err := api.CreateDrop(client.CreateDropRequest{
			FileName:       "bench_file.bin",
			FileSize:       int64(len(chunk)),
			EncryptionSalt: "v1-aes-gcm",
			ExpiresIn:      "1h",
			MaxDownloads:   1,
		})
		if err != nil {
			fmt.Printf("  CreateDrop failed (rep %d): %v\n", i+1, err)
			return
		}
		if err := api.UploadChunk(drop.DropID, 0, chunk); err != nil {
			fmt.Printf("  UploadChunk failed (rep %d): %v\n", i+1, err)
			return
		}
		fmt.Printf("  [%d/%d] uploaded drop %s\n", i+1, uploadReps, drop.DropID)
	}

	after, err := api.GetStats()
	if err != nil {
		fmt.Printf("  Failed to fetch final stats: %v\n", err)
		return
	}

	addedSaved := after.StorageSaved - before.StorageSaved
	addedUsed  := after.StorageUsed  - before.StorageUsed

	intendedBytes := int64(uploadReps) * int64(chunkSize)
	var savingsPct float64
	if intendedBytes > 0 {
		savingsPct = 100.0 * float64(addedSaved) / float64(intendedBytes)
	}

	fmt.Println()
	fmt.Printf("  Intended storage (no CAS)  : %s\n", fmtBytes(intendedBytes))
	fmt.Printf("  Actual storage used        : %s\n", fmtBytes(addedUsed))
	fmt.Printf("  Storage saved (CAS dedup)  : %s\n", fmtBytes(addedSaved))
	fmt.Printf("  Savings %%                  : %.1f%%\n", savingsPct)
	fmt.Println()
	fmt.Printf("  >> RESUME NUMBER: \"eliminated %.0f%% of redundant S3 writes for duplicate\n", savingsPct)
	fmt.Printf("     content via SHA-256 content-addressed storage\"\n")
}

func fmtBytes(b int64) string {
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
