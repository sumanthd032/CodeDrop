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

// deleteDrop wipes a single drop from S3 and Postgres
func (gc *GarbageCollector) deleteDrop(dropID string) {
	// 1. Find how many chunks this drop has so we can delete them from S3
	var chunkCount int
	err := gc.DB.QueryRow("SELECT COUNT(*) FROM chunks WHERE drop_id = $1", dropID).Scan(&chunkCount)
	if err != nil {
		log.Printf("[GC Error] Failed to count chunks for drop %s: %v", dropID, err)
		return
	}

	// 2. Delete chunks from S3
	for i := 0; i < chunkCount; i++ {
		s3Key := fmt.Sprintf("drops/%s/%d", dropID, i)
		if err := gc.Store.DeleteChunk(s3Key); err != nil {
			log.Printf("[GC Error] Failed to delete chunk %s from S3: %v", s3Key, err)
			// We continue even if one chunk fails, to try and clean up the rest
		}
	}

	// 3. Delete metadata from Postgres
	// Because we used ON DELETE CASCADE in our schema, deleting the drop 
	// will automatically delete all associated chunk records in the DB!
	_, err = gc.DB.Exec("DELETE FROM drops WHERE id = $1", dropID)
	if err != nil {
		log.Printf("[GC Error] Failed to delete drop %s from DB: %v", dropID, err)
		return
	}

	log.Printf("GC permanently destroyed drop: %s", dropID)
}