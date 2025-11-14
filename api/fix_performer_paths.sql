-- Fix performer preview paths by removing the leading /../
UPDATE performers
SET preview_path = REPLACE(preview_path, '/../assets/', '/assets/')
WHERE preview_path LIKE '/../assets/%';
