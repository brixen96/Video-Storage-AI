# Video Storage AI - API Test Script
# Quick test commands for the API

$baseUrl = "http://localhost:8080"

Write-Host "Video Storage AI - API Test Script" -ForegroundColor Cyan
Write-Host "====================================" -ForegroundColor Cyan
Write-Host ""

# Check if server is running
Write-Host "1. Checking if server is running..." -ForegroundColor Yellow
try {
    $health = Invoke-RestMethod -Uri "$baseUrl/health" -Method GET -ErrorAction Stop
    Write-Host "   [OK] Server is running!" -ForegroundColor Green
    Write-Host "   Status: $($health.status)" -ForegroundColor Gray
    Write-Host "   Database: $($health.database)" -ForegroundColor Gray
    Write-Host ""
} catch {
    Write-Host "   [ERROR] Server is not running!" -ForegroundColor Red
    Write-Host "   Please start the server with: make run" -ForegroundColor Yellow
    exit 1
}

# Scan performer directory
Write-Host "2. Scanning performer directory..." -ForegroundColor Yellow
$scanBody = @{
    directory = "c:\Repos\Video Storage AI\api\assets\performers"
    type = "performers"
} | ConvertTo-Json

try {
    $scanResult = Invoke-RestMethod -Uri "$baseUrl/api/v1/files/scan" -Method POST -Body $scanBody -ContentType "application/json" -ErrorAction Stop
    Write-Host "   [OK] Scan completed!" -ForegroundColor Green
    Write-Host "   Performers found: $($scanResult.data.performers_found)" -ForegroundColor Gray
    Write-Host "   Performers added: $($scanResult.data.performers_added)" -ForegroundColor Gray
    Write-Host "   Videos found: $($scanResult.data.videos_found)" -ForegroundColor Gray
    if ($scanResult.data.errors.Count -gt 0) {
        Write-Host "   Errors: $($scanResult.data.errors.Count)" -ForegroundColor Red
        foreach ($err in $scanResult.data.errors) {
            Write-Host "     - $err" -ForegroundColor Red
        }
    }
    Write-Host ""
} catch {
    Write-Host "   [ERROR] Scan failed!" -ForegroundColor Red
    Write-Host "   $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ""
}

# Get all performers
Write-Host "3. Getting all performers..." -ForegroundColor Yellow
try {
    $performers = Invoke-RestMethod -Uri "$baseUrl/api/v1/performers" -Method GET -ErrorAction Stop
    Write-Host "   [OK] Retrieved $($performers.data.Count) performers" -ForegroundColor Green

    if ($performers.data.Count -gt 0) {
        Write-Host ""
        Write-Host "   First 5 performers:" -ForegroundColor Cyan
        $performers.data | Select-Object -First 5 | ForEach-Object {
            Write-Host "     - ID: $($_.id) | Name: $($_.name) | Videos: $($_.scene_count)" -ForegroundColor Gray
        }
    }
    Write-Host ""
} catch {
    Write-Host "   [ERROR] Failed to get performers!" -ForegroundColor Red
    Write-Host "   $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ""
}

Write-Host "====================================" -ForegroundColor Cyan
Write-Host "Test completed!" -ForegroundColor Green
Write-Host ""
Write-Host "Try these commands:" -ForegroundColor Yellow
Write-Host "  Get all performers:" -ForegroundColor Gray
Write-Host "    Invoke-RestMethod -Uri '$baseUrl/api/v1/performers' -Method GET" -ForegroundColor White
Write-Host ""
Write-Host "  Search performers:" -ForegroundColor Gray
Write-Host "    Invoke-RestMethod -Uri '$baseUrl/api/v1/performers?search=Dredd' -Method GET" -ForegroundColor White
Write-Host ""
Write-Host "  Get single performer:" -ForegroundColor Gray
Write-Host "    Invoke-RestMethod -Uri '$baseUrl/api/v1/performers/1' -Method GET" -ForegroundColor White
Write-Host ""
