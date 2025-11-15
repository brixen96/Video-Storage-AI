package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Open database
	db, err := sql.Open("sqlite3", "data/video_storage.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Get all videos with thumbnail paths
	rows, err := db.Query("SELECT id, thumbnail_path FROM videos WHERE thumbnail_path IS NOT NULL AND thumbnail_path != ''")
	if err != nil {
		log.Fatalf("Failed to query videos: %v", err)
	}
	defer rows.Close()

	var updates []struct {
		id   int
		path string
	}

	for rows.Next() {
		var id int
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}

		// Fix the path
		newPath := path
		// Remove 'assets\' or 'assets/' prefix
		newPath = strings.TrimPrefix(newPath, "assets\\")
		newPath = strings.TrimPrefix(newPath, "assets/")
		// Convert backslashes to forward slashes
		newPath = strings.ReplaceAll(newPath, "\\", "/")

		if newPath != path {
			updates = append(updates, struct {
				id   int
				path string
			}{id, newPath})
			fmt.Printf("Video ID %d: '%s' -> '%s'\n", id, path, newPath)
		}
	}

	// Update the paths
	stmt, err := db.Prepare("UPDATE videos SET thumbnail_path = ? WHERE id = ?")
	if err != nil {
		log.Fatalf("Failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	for _, update := range updates {
		_, err := stmt.Exec(update.path, update.id)
		if err != nil {
			log.Printf("Failed to update video %d: %v", update.id, err)
		}
	}

	fmt.Printf("\nUpdated %d thumbnail paths\n", len(updates))
}
