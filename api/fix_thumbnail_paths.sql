-- Fix thumbnail paths in the database
-- Remove 'assets\' or 'assets/' prefix and convert backslashes to forward slashes

UPDATE videos
SET thumbnail_path = REPLACE(REPLACE(REPLACE(thumbnail_path, 'assets\', ''), 'assets/', ''), '\', '/')
WHERE thumbnail_path IS NOT NULL
  AND (thumbnail_path LIKE 'assets\%' OR thumbnail_path LIKE 'assets/%');
