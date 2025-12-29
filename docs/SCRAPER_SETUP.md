# Web Scraper Setup Guide

## Authentication Setup for simpcity.is

Since simpcity.is requires login to access forum threads, you need to provide your session cookie to the scraper.

### How to Get Your Session Cookie

1. **Login to simpcity.is** in your browser (Chrome/Edge recommended)

2. **Open Developer Tools**:
   - Press `F12` or right-click → "Inspect"
   - Go to the "Application" tab (Chrome) or "Storage" tab (Firefox)

3. **Find the Session Cookie**:
   - In the left sidebar, expand "Cookies"
   - Click on `https://simpcity.is`
   - Look for cookies named like `xf_session`, `xf_user`, or similar
   - Copy the entire cookie string including name and value

4. **Alternative - Copy All Cookies** (Recommended for simpcity.is):
   - Go to "Network" tab in Dev Tools
   - Refresh the page
   - Click on any request to simpcity.is
   - In the "Headers" section, find "Cookie:" under "Request Headers"
   - Copy the entire cookie string (looks like: `cookie1=value1; cookie2=value2; ...`)

**Important for simpcity.is**: The cookies use `ogaddgmetaprof_` prefix, not `xf_`. You need these cookies:
- `ogaddgmetaprof_csrf`
- `ogaddgmetaprof_session`
- `ogaddgmetaprof_user`
- `cucksed`
- `cucksid`

### Setting the Session Cookie

#### Method 1: Using the Frontend (Recommended)

1. Navigate to the **Scraper** page in your app
2. Click "**Settings**" or "**Configure Authentication**" button
3. Paste your cookie string
4. Click "**Save**"

#### Method 2: Using API Directly

Send a POST request to set the cookie:

```bash
curl -X POST http://localhost:8080/api/v1/scraper/session \
  -H "Content-Type: application/json" \
  -d '{"cookie":"your_cookie_string_here"}'
```

**Example Cookie String (simpcity.is):**
```
ogaddgmetaprof_csrf=YwWS-c1Le7F024lc; ogaddgmetaprof_session=NQDij4aJxP8-YUBuY0-vNBD1HKQ_peuB; ogaddgmetaprof_user=22148309%2CpJqMDnLPhG7QGA68Laizns9IWnGbGI5oef-9VhJB; cucksed=b3; cucksid=583ae1f4649e30d9968f89a97840ac73e0ad10b22d40590a45478e97964a2165
```

### Verify Cookie is Set

Check if authentication is configured:

```bash
curl http://localhost:8080/api/v1/scraper/session
```

Response:
```json
{
  "success": true,
  "data": {
    "is_set": true,
    "length": 245
  },
  "message": "Session cookie status retrieved"
}
```

## Usage

### Scraping a Thread

1. Navigate to the **Scraper** page
2. Click "**New Scrape Job**"
3. Paste the thread URL (e.g., `https://simpcity.is/threads/performer-name.123456/`)
4. Click "**Start Scraping**"
5. Monitor progress in the **Tasks** page

### What Gets Scraped

- **Thread Metadata**: Title, author, category, view count, reply count
- **All Posts**: Content, author, post date, attachments
- **Download Links**: Automatically detects gofile, pixeldrain, and bunkr links
- **Metadata Extraction**: Performer names, studio names from titles
- **Tags**: Thread tags if available

### AI Companion Integration

All scraped data is automatically:
- Stored in the AI Companion's memory
- Available for the AI to reference in conversations
- Used for intelligent recommendations

## Troubleshooting

### Error: "authentication required - please set session cookie"

Your session cookie is not set or has expired. Follow the steps above to set a new cookie.

### Error: "unexpected status code: 403"

Your session cookie has expired. Login to simpcity.is again and get a new cookie.

### Downloads Links Not Found

The scraper uses regex patterns to detect links. If links aren't being found:
- Check if they're in a supported format (gofile, pixeldrain, bunkr)
- The links might be in images/attachments (not yet supported)
- Links might be obfuscated or shortened

### Thread Not Scraping

- Ensure the URL is valid and starts with `https://simpcity.is/threads/`
- Check the Tasks page for error messages
- Verify your session cookie is still valid

## Security Notes

- **Cookie Security**: Your session cookie is stored in memory only (not persisted to disk)
- **Privacy**: Only you have access to scraped data on your local machine
- **Session Expiry**: Cookies typically expire after 30 days or when you logout
- **Multi-Account**: The scraper uses a single session - logging out of the browser will invalidate it

## Advanced Usage

### Batch Scraping

To scrape multiple threads, use the API:

```bash
for url in thread1_url thread2_url thread3_url; do
  curl -X POST http://localhost:8080/api/v1/scraper/threads/scrape \
    -H "Content-Type: application/json" \
    -d "{\"url\":\"$url\"}"
  sleep 5  # Wait 5 seconds between requests
done
```

### Rescraping for Updates

Click the "Rescrape" button on any thread to fetch new posts and updated metadata.

### Search Scraped Data

Use the search box on the Scraper page to find threads by:
- Title keywords
- Author name
- Performer names (extracted from titles)

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/scraper/stats` | GET | Get scraper statistics |
| `/api/v1/scraper/threads` | GET | List all scraped threads |
| `/api/v1/scraper/threads/search?q=query` | GET | Search threads |
| `/api/v1/scraper/threads/:id` | GET | Get single thread |
| `/api/v1/scraper/threads/scrape` | POST | Start scraping |
| `/api/v1/scraper/threads/:id/rescrape` | POST | Rescrape existing thread |
| `/api/v1/scraper/session` | POST | Set session cookie |
| `/api/v1/scraper/session` | GET | Get cookie status |

## Tips

1. **Rate Limiting**: Add delays between scrapes to avoid being rate-limited
2. **Cookie Rotation**: If you have multiple accounts, you can switch cookies
3. **Backup Data**: Scraped data is stored in `video-storage.db` - back it up regularly
4. **Privacy Mode**: Use browser private mode to avoid logging out of your main session when getting cookies
