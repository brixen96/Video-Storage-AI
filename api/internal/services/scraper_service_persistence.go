package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// SaveThread saves a scraped thread to the database
func (s *ScraperService) SaveThread(thread *models.ScrapedThread) error {
	// Marshal metadata
	if err := thread.MarshalMetadata(); err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Check if thread already exists
	var existingID int64
	err := s.db.QueryRow(`
		SELECT id FROM scraped_threads
		WHERE external_id = ? AND source = ?
	`, thread.ExternalID, thread.Source).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Insert new thread
		result, err := s.db.Exec(`
			INSERT INTO scraped_threads (
				external_id, source, title, url, author, category,
				view_count, reply_count, post_count, download_count,
				metadata, first_scraped_at, last_scraped_at,
				last_updated_at, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, thread.ExternalID, thread.Source, thread.Title, thread.URL,
			thread.Author, thread.Category, thread.ViewCount, thread.ReplyCount,
			thread.PostCount, thread.DownloadCount, thread.Metadata,
			thread.FirstScrapedAt, thread.LastScrapedAt, thread.LastUpdatedAt,
			time.Now())

		if err != nil {
			return fmt.Errorf("failed to insert thread: %w", err)
		}

		threadID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get thread ID: %w", err)
		}
		thread.ID = threadID
	} else if err != nil {
		return fmt.Errorf("failed to check existing thread: %w", err)
	} else {
		// Update existing thread
		_, err = s.db.Exec(`
			UPDATE scraped_threads SET
				title = ?, url = ?, author = ?, category = ?,
				view_count = ?, reply_count = ?, post_count = ?,
				download_count = ?, metadata = ?, last_scraped_at = ?,
				last_updated_at = ?
			WHERE id = ?
		`, thread.Title, thread.URL, thread.Author, thread.Category,
			thread.ViewCount, thread.ReplyCount, thread.PostCount,
			thread.DownloadCount, thread.Metadata, time.Now(),
			thread.LastUpdatedAt, existingID)

		if err != nil {
			return fmt.Errorf("failed to update thread: %w", err)
		}

		thread.ID = existingID
	}

	return nil
}

// SavePost saves a scraped post to the database
func (s *ScraperService) SavePost(post *models.ScrapedPost) error {
	// Marshal metadata
	if err := post.MarshalMetadata(); err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Check if post already exists
	var existingID int64
	err := s.db.QueryRow(`
		SELECT id FROM scraped_posts
		WHERE external_id = ? AND source = ?
	`, post.ExternalID, post.Source).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Insert new post
		result, err := s.db.Exec(`
			INSERT INTO scraped_posts (
				thread_id, external_id, source, author, content,
				plain_text, post_number, like_count, metadata,
				posted_at, scraped_at, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, post.ThreadID, post.ExternalID, post.Source, post.Author,
			post.Content, post.PlainText, post.PostNumber, post.LikeCount,
			post.Metadata, post.PostedAt, post.ScrapedAt, time.Now())

		if err != nil {
			return fmt.Errorf("failed to insert post: %w", err)
		}

		postID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get post ID: %w", err)
		}
		post.ID = postID
	} else if err != nil {
		return fmt.Errorf("failed to check existing post: %w", err)
	} else {
		// Update existing post
		_, err = s.db.Exec(`
			UPDATE scraped_posts SET
				content = ?, plain_text = ?, like_count = ?,
				metadata = ?, scraped_at = ?
			WHERE id = ?
		`, post.Content, post.PlainText, post.LikeCount,
			post.Metadata, time.Now(), existingID)

		if err != nil {
			return fmt.Errorf("failed to update post: %w", err)
		}

		post.ID = existingID
	}

	return nil
}

// SaveDownloadLink saves a download link to the database
func (s *ScraperService) SaveDownloadLink(link *models.ScrapedDownloadLink) error {
	// Marshal metadata
	if err := link.MarshalMetadata(); err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Check if link already exists
	var existingID int64
	err := s.db.QueryRow(`
		SELECT id FROM scraped_download_links
		WHERE url = ? AND source = ?
	`, link.URL, link.Source).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Insert new link
		result, err := s.db.Exec(`
			INSERT INTO scraped_download_links (
				thread_id, post_id, source, provider, url, original_url,
				filename, file_size, file_type, status, metadata,
				discovered_at, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, link.ThreadID, link.PostID, link.Source, link.Provider,
			link.URL, link.OriginalURL, link.Filename, link.FileSize,
			link.FileType, link.Status, link.Metadata, link.DiscoveredAt,
			time.Now())

		if err != nil {
			return fmt.Errorf("failed to insert download link: %w", err)
		}

		linkID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get download link ID: %w", err)
		}
		link.ID = linkID
	} else if err != nil {
		return fmt.Errorf("failed to check existing link: %w", err)
	} else {
		// Update existing link
		link.ID = existingID
		// Don't update existing links, just track they were found again
	}

	return nil
}

// notifyAICompanion sends scraping results to AI Companion
func (s *ScraperService) notifyAICompanion(thread *models.ScrapedThread, postCount, linkCount int) {
	if s.aiCompanionService == nil {
		return
	}

	// Create memory entry
	memoryData := map[string]interface{}{
		"type":           "scraper_result",
		"source":         "simpcity",
		"thread_title":   thread.Title,
		"thread_url":     thread.URL,
		"author":         thread.Author,
		"category":       thread.Category,
		"post_count":     postCount,
		"download_count": linkCount,
		"performers":     thread.MetadataObj.PerformerNames,
		"studios":        thread.MetadataObj.StudioNames,
		"tags":           thread.MetadataObj.Tags,
		"timestamp":      time.Now(),
	}

	memoryJSON, _ := json.Marshal(memoryData)

	// Store in AI Companion memory
	s.db.Exec(`
		INSERT INTO ai_companion_memory (type, content, metadata, created_at)
		VALUES (?, ?, ?, ?)
	`, "scraper_result", thread.Title, string(memoryJSON), time.Now())

	log.Printf("AI Companion notified of scraping results: %s", thread.Title)
}

// DeleteThread deletes a thread and all associated posts and download links
func (s *ScraperService) DeleteThread(threadID int64) error {
	// Start a transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete download links first
	_, err = tx.Exec("DELETE FROM scraped_download_links WHERE thread_id = ?", threadID)
	if err != nil {
		return fmt.Errorf("failed to delete download links: %w", err)
	}

	// Delete posts
	_, err = tx.Exec("DELETE FROM scraped_posts WHERE thread_id = ?", threadID)
	if err != nil {
		return fmt.Errorf("failed to delete posts: %w", err)
	}

	// Delete thread
	result, err := tx.Exec("DELETE FROM scraped_threads WHERE id = ?", threadID)
	if err != nil {
		return fmt.Errorf("failed to delete thread: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("thread not found")
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Deleted thread %d and all associated data", threadID)
	return nil
}

// DeleteThreads deletes multiple threads by their IDs
func (s *ScraperService) DeleteThreads(threadIDs []int64) error {
	if len(threadIDs) == 0 {
		return fmt.Errorf("no thread IDs provided")
	}

	// Start a transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete download links for these threads
	placeholders := make([]string, len(threadIDs))
	args := make([]interface{}, len(threadIDs))
	for i, id := range threadIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf("DELETE FROM scraped_download_links WHERE thread_id IN (%s)", strings.Join(placeholders, ","))
	_, err = tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete download links: %w", err)
	}

	// Delete posts for these threads
	query = fmt.Sprintf("DELETE FROM scraped_posts WHERE thread_id IN (%s)", strings.Join(placeholders, ","))
	_, err = tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete posts: %w", err)
	}

	// Delete performer-thread links
	query = fmt.Sprintf("DELETE FROM performer_scraped_threads WHERE thread_id IN (%s)", strings.Join(placeholders, ","))
	_, err = tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete performer links: %w", err)
	}

	// Delete the threads themselves
	query = fmt.Sprintf("DELETE FROM scraped_threads WHERE id IN (%s)", strings.Join(placeholders, ","))
	_, err = tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete threads: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Successfully deleted %d threads", len(threadIDs))
	return nil
}

// DeleteAllThreads deletes all scraped threads from the database
func (s *ScraperService) DeleteAllThreads() error {
	// Start a transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete all download links
	_, err = tx.Exec("DELETE FROM scraped_download_links")
	if err != nil {
		return fmt.Errorf("failed to delete download links: %w", err)
	}

	// Delete all posts
	_, err = tx.Exec("DELETE FROM scraped_posts")
	if err != nil {
		return fmt.Errorf("failed to delete posts: %w", err)
	}

	// Delete all performer-thread links
	_, err = tx.Exec("DELETE FROM performer_scraped_threads")
	if err != nil {
		return fmt.Errorf("failed to delete performer links: %w", err)
	}

	// Delete all threads
	_, err = tx.Exec("DELETE FROM scraped_threads")
	if err != nil {
		return fmt.Errorf("failed to delete threads: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Println("Successfully deleted all scraped threads")
	return nil
}
