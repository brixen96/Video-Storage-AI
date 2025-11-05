# Video Storage AI - API Examples

This document provides example API requests you can use to test the endpoints.

## Base URL

```
http://localhost:8080
```

## Health Check

```bash
GET http://localhost:8080/health
```

**Response:**
```json
{
  "status": "healthy",
  "database": "connected",
  "version": "0.1.0"
}
```

---

## Performer Endpoints

### 1. Get All Performers

```bash
GET http://localhost:8080/api/v1/performers
```

**Response:**
```json
{
  "success": true,
  "message": "Performers retrieved successfully",
  "data": [
    {
      "id": 1,
      "name": "Dredd",
      "preview_path": "c:\\Repos\\Video Storage AI\\api\\assets\\performers\\Dredd\\37384681a_1_chr2_prob4.mkv",
      "folder_path": "c:\\Repos\\Video Storage AI\\api\\assets\\performers\\Dredd",
      "scene_count": 0,
      "created_at": "2025-11-04T21:00:00Z",
      "updated_at": "2025-11-04T21:00:00Z"
    }
  ],
  "timestamp": "2025-11-04T21:00:00Z"
}
```

### 2. Search Performers

```bash
GET http://localhost:8080/api/v1/performers?search=Dredd
```

### 3. Get Single Performer

```bash
GET http://localhost:8080/api/v1/performers/1
```

### 4. Create Performer

```bash
POST http://localhost:8080/api/v1/performers
Content-Type: application/json

{
  "name": "New Performer",
  "preview_path": "./assets/performers/New Performer/preview.mkv",
  "folder_path": "./assets/performers/New Performer",
  "metadata": {
    "bio": "Bio goes here",
    "birthdate": "1990-01-01",
    "height": "5'6\"",
    "ethnicity": "Caucasian"
  }
}
```

### 5. Update Performer

```bash
PUT http://localhost:8080/api/v1/performers/1
Content-Type: application/json

{
  "name": "Updated Name",
  "scene_count": 5
}
```

### 6. Delete Performer

```bash
DELETE http://localhost:8080/api/v1/performers/1
```

### 7. Reset Performer Metadata

```bash
POST http://localhost:8080/api/v1/performers/1/reset-metadata
```

---

## File Scanning Endpoints

### 1. Scan Performer Directory

This will scan your performer assets directory and automatically create database entries for each performer.

```bash
POST http://localhost:8080/api/v1/files/scan
Content-Type: application/json

{
  "directory": "c:\\Repos\\Video Storage AI\\api\\assets\\performers",
  "type": "performers"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Performer directory scanned successfully",
  "data": {
    "performers_found": 25,
    "performers_added": 25,
    "videos_found": 95,
    "errors": []
  },
  "timestamp": "2025-11-04T21:00:00Z"
}
```

### 2. Scan Video Directory

```bash
POST http://localhost:8080/api/v1/files/scan
Content-Type: application/json

{
  "directory": "D:\\Videos\\MyVideos",
  "type": "videos",
  "recursive": true
}
```

---

## Testing with cURL

### Get All Performers

```bash
curl http://localhost:8080/api/v1/performers
```

### Scan Performer Directory

```bash
curl -X POST http://localhost:8080/api/v1/files/scan \
  -H "Content-Type: application/json" \
  -d "{\"directory\": \"c:\\\\Repos\\\\Video Storage AI\\\\api\\\\assets\\\\performers\", \"type\": \"performers\"}"
```

### Create a Performer

```bash
curl -X POST http://localhost:8080/api/v1/performers \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"Test Performer\", \"folder_path\": \"./assets/performers/Test Performer\"}"
```

---

## Testing with PowerShell

### Get All Performers

```powershell
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/performers" -Method GET
```

### Scan Performer Directory

```powershell
$body = @{
    directory = "c:\Repos\Video Storage AI\api\assets\performers"
    type = "performers"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8080/api/v1/files/scan" -Method POST -Body $body -ContentType "application/json"
```

### Create a Performer

```powershell
$body = @{
    name = "Test Performer"
    folder_path = "./assets/performers/Test Performer"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8080/api/v1/performers" -Method POST -Body $body -ContentType "application/json"
```

---

## Quick Start: Populate Your Database

1. **Start the server:**
   ```bash
   cd "c:\Repos\Video Storage AI\api"
   make run
   # or
   go run cmd/server/main.go
   ```

2. **Scan your performer directory** (PowerShell):
   ```powershell
   $body = @{
       directory = "c:\Repos\Video Storage AI\api\assets\performers"
       type = "performers"
   } | ConvertTo-Json

   Invoke-RestMethod -Uri "http://localhost:8080/api/v1/files/scan" -Method POST -Body $body -ContentType "application/json"
   ```

3. **View all performers:**
   ```powershell
   Invoke-RestMethod -Uri "http://localhost:8080/api/v1/performers" -Method GET
   ```

4. **Access performer preview videos:**
   ```
   http://localhost:8080/assets/performers/Dredd/37384681a_1_chr2_prob4.mkv
   ```

---

## Error Responses

All errors follow this format:

```json
{
  "success": false,
  "error": "Error message",
  "details": "Detailed error information",
  "timestamp": "2025-11-04T21:00:00Z"
}
```

---

## Common HTTP Status Codes

- `200 OK` - Request succeeded
- `201 Created` - Resource created successfully
- `400 Bad Request` - Invalid request data
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

---

## Next Steps

1. Test the performer endpoints with your existing data
2. Scan the performer directory to populate the database
3. Access performer preview videos through the `/assets` endpoint
4. Build the Vue 3 frontend to visualize the data

---

**Need help?** Check the [README.md](README.md) for more information.
