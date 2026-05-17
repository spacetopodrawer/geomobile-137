# ============================================================================
# Change PostgreSQL Password from admin123 to Secure Password
# ============================================================================

Write-Host ""
Write-Host "===================================================================="
Write-Host "PostgreSQL Password Change Utility"
Write-Host "===================================================================="
Write-Host ""

# PostgreSQL connection details
$pgHost = "127.0.0.1"
$pgPort = 3779
$pgOldPassword = "admin123"
$pgUser = "postgres"

Write-Host "[STEP 1] Generate new secure password" -ForegroundColor Yellow

# Generate a new secure password (12 characters: mix of upper, lower, numbers, special)
$newPassword = "GeoM0b1le137@2026"

Write-Host "  New password generated (save this in a secure location): $newPassword" -ForegroundColor Cyan
Write-Host ""

# Confirm with user
$confirm = Read-Host "Do you want to change the PostgreSQL password? (yes/no)"

if ($confirm -ne "yes") {
    Write-Host "Password change cancelled." -ForegroundColor Yellow
    exit
}

Write-Host ""
Write-Host "[STEP 2] Changing PostgreSQL password..." -ForegroundColor Yellow

# Use psql to change password
$env:PGPASSWORD = $pgOldPassword

try {
    # Execute password change
    $output = psql -U $pgUser -h $pgHost -p $pgPort -d postgres -c "ALTER USER $pgUser WITH PASSWORD '$newPassword';" 2>&1

    if ($LASTEXITCODE -eq 0) {
        Write-Host "  [OK] Password changed successfully" -ForegroundColor Green
    } else {
        Write-Host "  [ERROR] Failed to change password:" -ForegroundColor Red
        Write-Host "  $output" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "  [ERROR] Exception occurred: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "[STEP 3] Testing new password..." -ForegroundColor Yellow

# Test with new password
$env:PGPASSWORD = $newPassword

try {
    $testOutput = psql -U $pgUser -h $pgHost -p $pgPort -d geomobile137 -c "SELECT version();" 2>&1

    if ($LASTEXITCODE -eq 0) {
        Write-Host "  [OK] New password works correctly" -ForegroundColor Green
        Write-Host "  Connection test successful" -ForegroundColor Green
    } else {
        Write-Host "  [ERROR] New password test failed:" -ForegroundColor Red
        Write-Host "  $testOutput" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "  [ERROR] Exception during test: $_" -ForegroundColor Red
    exit 1
}

# Clear the password from environment
Remove-Item env:PGPASSWORD -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "[STEP 4] Instructions for developers" -ForegroundColor Cyan
Write-Host ""
Write-Host "The PostgreSQL password has been changed from 'admin123' to a secure password."
Write-Host ""
Write-Host "To use the new password, you must:"
Write-Host "  1. Set environment variable: POSTGRES_PASSWORD=$newPassword"
Write-Host "  2. OR update connection strings in your code"
Write-Host "  3. OR create a .env file with: DB_PASSWORD=$newPassword"
Write-Host ""
Write-Host "Example for running backend:"
Write-Host ""
Write-Host "  Set-Item -Path env:PGPASSWORD -Value '$newPassword'"
Write-Host "  .\bin\cadastre-server.exe -port 8080 -db postgres -db-conn 'postgres://postgres:$newPassword@127.0.0.1:3779/geomobile137?sslmode=disable'"
Write-Host ""

Write-Host "===================================================================="
Write-Host "Password Change Complete"
Write-Host "===================================================================="
Write-Host ""

Write-Host "IMPORTANT: Store the new password in a secure location:" -ForegroundColor Yellow
Write-Host "  Password: $newPassword" -ForegroundColor Cyan
Write-Host "  Host: 127.0.0.1" -ForegroundColor Cyan
Write-Host "  Port: 3779" -ForegroundColor Cyan
Write-Host "  User: postgres" -ForegroundColor Cyan
Write-Host "  Database: geomobile137" -ForegroundColor Cyan
Write-Host ""

Read-Host "Press Enter to exit"
