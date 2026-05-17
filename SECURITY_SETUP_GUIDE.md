# Security Setup Guide - Phase 1 Completion

**Date:** 2026-05-17  
**Status:** CRITICAL - Complete before production deployment  
**Priority:** HIGH

---

## Overview

This guide covers securing the GeoMobile137 system before production deployment, specifically:
1. Changing PostgreSQL default password
2. Setting up environment variables for credentials
3. Securing the git repository
4. Verifying no credentials are exposed

---

## Step 1: Change PostgreSQL Password

### Why This Is Critical

The default password `admin123` is currently hardcoded in documentation and used for development. This must be changed before any production deployment.

### How to Change the Password

#### Method 1: Using the Provided Script (Recommended)

```powershell
# Run from F:\geomobile137
powershell -ExecutionPolicy Bypass -File "CHANGE_POSTGRES_PASSWORD.ps1"
```

This script will:
- Generate a new secure password
- Change it in PostgreSQL
- Test the new password
- Display instructions for using the new password

#### Method 2: Manual Change via psql

```powershell
# Set the old password in environment
$env:PGPASSWORD = "admin123"

# Connect and change password
psql -U postgres -h 127.0.0.1 -p 3779 -d postgres -c "ALTER USER postgres WITH PASSWORD 'YOUR_NEW_PASSWORD';"

# Test the new password
$env:PGPASSWORD = "YOUR_NEW_PASSWORD"
psql -U postgres -h 127.0.0.1 -p 3779 -d geomobile137 -c "SELECT version();"

# Clear environment
Remove-Item env:PGPASSWORD
```

### Recommended Secure Password

Use a password with:
- At least 12 characters
- Mix of uppercase and lowercase
- Numbers and special characters
- Example: `GeoM0b1le137@SecureDB#2026`

---

## Step 2: Configure Environment Variables

### Create .env File (Development)

```powershell
# From F:\geomobile137
Copy-Item ".env.example" ".env"

# Edit .env and update DATABASE_URL with your new password:
# DATABASE_URL=postgres://postgres:YOUR_NEW_PASSWORD@127.0.0.1:3779/geomobile137?sslmode=disable
```

### Important: Never Commit .env to Git

Add to .gitignore (if not already there):
```
.env
.env.local
.env.*.local
*.key
*.pem
```

---

## Step 3: Using the New Password

### Option 1: Environment Variable (Recommended for Production)

```powershell
# Set in PowerShell
$env:DATABASE_URL = "postgres://postgres:YOUR_NEW_PASSWORD@127.0.0.1:3779/geomobile137?sslmode=disable"

# Run backend
.\bin\cadastre-server.exe -port 8080 -db postgres
```

### Option 2: Command-Line Flag

```powershell
.\bin\cadastre-server.exe -port 8080 -db postgres -db-conn "postgres://postgres:YOUR_NEW_PASSWORD@127.0.0.1:3779/geomobile137?sslmode=disable"
```

### Option 3: .env File (Development Only)

```bash
# Create .env file with:
DATABASE_URL=postgres://postgres:YOUR_NEW_PASSWORD@127.0.0.1:3779/geomobile137?sslmode=disable

# Backend will automatically load from environment
.\bin\cadastre-server.exe -port 8080 -db postgres
```

---

## Step 4: Verify No Credentials Are Exposed in Git

### Check Git History for Passwords

```bash
# Search for common password patterns in git history
git log --all --patch --grep="password" -- "*.md" "*.txt" "*.go"

# Search for hardcoded credentials
git log --all --oneline | grep -i "admin123\|password\|credential"

# Check all commits containing admin123
git log --all --source --remotes -S "admin123"
```

### If Credentials Were Exposed

```bash
# If admin123 appears in git history:
# 1. Change the password immediately in PostgreSQL
# 2. Invalidate the old password on all systems
# 3. Rewrite git history (CAUTION - advanced operation):

git filter-branch --env-filter '
if [ "$GIT_COMMIT" = "COMMIT_HASH" ]; then
    export GIT_AUTHOR_DATE="2026-05-17T00:00:00 +0000"
    export GIT_COMMITTER_DATE="2026-05-17T00:00:00 +0000"
fi' -f -- --all
```

---

## Step 5: Securing the Repository Before Push

### Verify No Secrets in Staged Files

```powershell
# Check what's about to be committed
git diff --cached | grep -i "password\|admin123\|secret" | head -20

# If credentials found, unstage:
git reset HEAD <filename>
git checkout -- <filename>
```

### Create .gitignore for Secrets

```
# Add to .gitignore
.env
.env.local
.env.*.local
*.key
*.pem
*.p12
.aws/
.azure/
.gcloud/
credentials/
secrets/
```

### Scan Repository with TruffleHog (Optional but Recommended)

```powershell
# Install trufflehog
pip install trufflehog --break-system-packages

# Scan repository
trufflehog filesystem . --json

# Or scan git history
trufflehog git file://. --json
```

---

## Step 6: Documentation Updates

### Remove Credentials from Public Documentation

Files to review:
- `PHASE_1_FINAL_STATUS_REPORT.txt` - Update password references
- `PHASE_1_COMPLETION_FINAL.md` - Update connection strings
- `CLAUDE.md` - Update startup instructions
- `SYNC_PHASE_1_AND_COMMIT.ps1` - Already fixed to use placeholders

### Safe Documentation Pattern

Instead of:
```
Connection: postgres://postgres:admin123@127.0.0.1:3779/...
```

Use:
```
Connection: postgres://postgres:YOUR_PASSWORD@127.0.0.1:3779/...
Setup: Run CHANGE_POSTGRES_PASSWORD.ps1 to set your password
```

---

## Step 7: Production Deployment Checklist

- [ ] PostgreSQL password changed from admin123
- [ ] New password stored in secure location (password manager)
- [ ] Environment variables configured for all systems
- [ ] .env file created locally (never committed)
- [ ] Git history checked for exposed credentials
- [ ] All documentation uses placeholder passwords only
- [ ] .gitignore includes .env and credential files
- [ ] Code updated to read from environment variables
- [ ] All three systems tested with new password
- [ ] Backup of database with new password verified
- [ ] Commit message includes security update note
- [ ] Repository scanned for secrets before push

---

## Step 8: Ongoing Security Maintenance

### Regular Tasks

1. **Rotate Passwords Quarterly**
   ```powershell
   powershell -ExecutionPolicy Bypass -File "CHANGE_POSTGRES_PASSWORD.ps1"
   ```

2. **Audit Git History Monthly**
   ```bash
   git log --all --patch --oneline | grep -i "password\|secret\|credential" | head -10
   ```

3. **Check for Exposed Secrets**
   ```bash
   git ls-files | xargs grep -l "password\|admin123\|secret" 2>/dev/null
   ```

4. **Review Environment Variables**
   ```powershell
   Get-ChildItem env: | grep -i "database\|password\|secret"
   ```

---

## Step 9: What NOT to Do

❌ **DON'T:**
- Commit .env files to git
- Hardcode passwords in source code
- Share passwords via email or Slack
- Use weak passwords (less than 12 characters)
- Leave admin123 in production
- Upload backups with embedded credentials
- Commit SQL dumps with data

✅ **DO:**
- Use environment variables for all secrets
- Store passwords in a password manager
- Rotate passwords regularly
- Use strong passwords
- Review git history before pushing
- Use .env.example as a template only
- Keep production passwords secure

---

## Security Policy Summary

| Item | Development | Staging | Production |
|------|-------------|---------|------------|
| **Password** | admin123 (dev only) | Random 12+ char | Random 16+ char |
| **Storage** | .env file | Environment vars | Secrets manager |
| **Rotation** | Never | Monthly | Quarterly |
| **Auditing** | Manual | Automated | Automated |
| **Backup** | Optional | Required | Required + encrypted |

---

## Questions or Issues?

1. Review git logs: `git log --all --oneline`
2. Check environment: `Get-ChildItem env: | Select-String Database`
3. Test connection: `psql -U postgres -h 127.0.0.1 -p 3779 -d geomobile137`
4. Review PostgreSQL logs: `Get-Content "C:\Program Files\PostgreSQL\18\data\log\*" -Tail 50`

---

## References

- PostgreSQL Password Security: https://www.postgresql.org/docs/18/libpq-envars.html
- Git Secrets Management: https://git-scm.com/docs/git-secrets
- OWASP Password Guidelines: https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html

---

**Status:** Complete this guide before production deployment  
**Last Updated:** 2026-05-17  
**Reviewed By:** Security Review Process
