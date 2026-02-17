package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sumanthd032/codedrop/internal/db"
	"github.com/sumanthd032/codedrop/internal/store"
)

// GarbageCollector handles the background cleanup of expired drops
type GarbageCollector struct {
	DB    *db.DB
	Store *store.Store
}

func NewGarbageCollector(db *db.DB, store *store.Store) *GarbageCollector {
	return &GarbageCollector{
		DB:    db,
		Store: store,
	}
}

// Start begins the garbage collection loop
func (gc *GarbageCollector) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Garbage Collector started. Running every %v\n", interval)

	for {
		select {
		case <-ctx.Done(): // Graceful shutdown
			log.Println("Garbage Collector stopping...")
			return
		case <-ticker.C:
			gc.sweep()
		}
	}
}

// sweep does the actual work of finding and deleting old drops
func (gc *GarbageCollector) sweep() {
	// 1. Find all expired drops
	// We only select the ID. We don't need the rest of the metadata.
	rows, err := gc.DB.Query("SELECT id FROM drops WHERE expires_at < NOW()")
	if err != nil {
		log.Printf("[GC Error] Failed to query expired drops: %v", err)
		return
	}
	defer rows.Close()

	var expiredIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Printf("[GC Error] Failed to scan drop ID: %v", err)
			continue
		}
		expiredIDs = append(expiredIDs, id)
	}

	if len(expiredIDs) == 0 {
		return // Nothing to clean up
	}

	log.Printf("GC sweeping %d expired drops...\n", len(expiredIDs))

	// 2. Process each expired drop
	for _, dropID := range expiredIDs {
		gc.deleteDrop(dropID)
	}
}

// deleteDrop wipes a single drop from S3 and Postgres safely using Reference Counting
func (gc *GarbageCollector) deleteDrop(dropID string) {
	// A. Find all chunk hashes associated with this drop
	rows, err := gc.DB.Query("SELECT chunk_hash FROM chunks WHERE drop_id = $1", dropID)
	if err != nil {
		log.Printf("[GC Error] Failed to fetch chunks for drop %s: %v", dropID, err)
		return
	}
	defer rows.Close()

	var hashes []string
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err == nil {
			hashes = append(hashes, hash)
		}
	}

	// B. Delete metadata from Postgres FIRST
	// This removes this drop's "claim" on the chunks.
	_, err = gc.DB.Exec("DELETE FROM drops WHERE id = $1", dropID)
	if err != nil {
		log.Printf("[GC Error] Failed to delete drop %s from DB: %v", dropID, err)
		return
	}

	// C. Safely delete from S3 (Reference Counting)
	for _, hash := range hashes {
		// Check if any OTHER active drops are still using this exact chunk
		var count int
		err := gc.DB.QueryRow("SELECT COUNT(*) FROM chunks WHERE chunk_hash = $1", hash).Scan(&count)
		
		if err == nil && count == 0 {
			// NO ONE else is using this chunk. It is safe to destroy physically.
			s3Key := fmt.Sprintf("chunks/%s", hash)
			if err := gc.Store.DeleteChunk(s3Key); err != nil {
				log.Printf("[GC Error] Failed to delete orphaned chunk %s from S3: %v", s3Key, err)
			} else {
				log.Printf("GC reclaimed storage space for chunk: %s", hash[:8])
			}
		} else {
			log.Printf("GC skipped S3 deletion for chunk %s (still in use by %d other drops)", hash[:8], count)
		}
	}

	log.Printf("GC permanently destroyed drop: %s", dropID)
}