# ╔══════════════════════════════════════════════════════════════════════════════╗
# ║  PC PRINCIPAL (DESKTOP-BTI208H) — Script de session de co-développement    ║
# ║  Projet : GeoMobile137  |  Dossier : F:\geomobile137                       ║
# ║  À exécuter en DÉBUT de chaque session de travail                          ║
# ╚══════════════════════════════════════════════════════════════════════════════╝

$Repo = "F:\geomobile137"
Set-Location $Repo

Write-Host ""
Write-Host "╔══════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║   GeoMobile137 — Démarrage session PC PRINCIPAL             ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan

# ── 1. Vérifier que le partage réseau est actif ──────────────────────────────
Write-Host "`n▶ [1/6] Vérification du partage réseau..." -ForegroundColor Yellow
$share = Get-SmbShare -Name "geomobile137" -ErrorAction SilentlyContinue
if (-not $share) {
    Write-Host "  ⚠️  Partage absent — création du partage réseau..." -ForegroundColor Orange
    New-SmbShare -Name "geomobile137" -Path $Repo -FullAccess "Everyone" -ErrorAction SilentlyContinue
    Write-Host "  ✅ Partage créé : \\DESKTOP-BTI208H\geomobile137" -ForegroundColor Green
} else {
    Write-Host "  ✅ Partage actif : \\DESKTOP-BTI208H\geomobile137" -ForegroundColor Green
}

# ── 2. Vérifier l'état Git ────────────────────────────────────────────────────
Write-Host "`n▶ [2/6] État Git local..." -ForegroundColor Yellow
$branch = git rev-parse --abbrev-ref HEAD 2>&1
Write-Host "  Branche : $branch" -ForegroundColor White
$status = git status --porcelain
if ($status) {
    Write-Host "  ⚠️  Fichiers modifiés localement :" -ForegroundColor Orange
    $status | ForEach-Object { Write-Host "     $_" -ForegroundColor White }
    $resp = Read-Host "  → Committer avant de continuer ? (o/n)"
    if ($resp -eq "o") {
        $msg = Read-Host "  Message de commit"
        git add -A
        git commit -m $msg
        Write-Host "  ✅ Commit effectué" -ForegroundColor Green
    }
} else {
    Write-Host "  ✅ Dépôt propre — aucune modification en attente" -ForegroundColor Green
}

# ── 3. Démarrer PostgreSQL ────────────────────────────────────────────────────
Write-Host "`n▶ [3/6] Démarrage PostgreSQL (port 3779)..." -ForegroundColor Yellow
$pg = Get-Service -Name "postgresql-x64-18" -ErrorAction SilentlyContinue
if ($pg -and $pg.Status -ne "Running") {
    Start-Service -Name "postgresql-x64-18"
    Start-Sleep -Seconds 2
}
$pgStatus = (Get-Service -Name "postgresql-x64-18" -ErrorAction SilentlyContinue).Status
Write-Host "  PostgreSQL : $pgStatus" -ForegroundColor $(if ($pgStatus -eq "Running") { "Green" } else { "Red" })

# ── 4. Démarrer Redis (Docker) ────────────────────────────────────────────────
Write-Host "`n▶ [4/6] Démarrage Redis via Docker..." -ForegroundColor Yellow
docker compose up -d redis 2>&1 | Select-String -Pattern "Started|Running|up-to-date" | ForEach-Object { Write-Host "  $_" -ForegroundColor Green }
Write-Host "  ✅ Redis disponible sur port 6379" -ForegroundColor Green

# ── 5. Compiler et démarrer le Backend Go ────────────────────────────────────
Write-Host "`n▶ [5/6] Démarrage du Backend Go (port 8080)..." -ForegroundColor Yellow
$dbConn = "postgres://postgres:admin123@127.0.0.1:3779/geomobile137?sslmode=disable"

# Recompiler si nécessaire
if (-not (Test-Path ".\bin\cadastre-server.exe") -or
    ((Get-Item ".\cmd\cadastre-server\main.go").LastWriteTime -gt (Get-Item ".\bin\cadastre-server.exe").LastWriteTime)) {
    Write-Host "  → Recompilation nécessaire..." -ForegroundColor Yellow
    go build -o bin\cadastre-server.exe .\cmd\cadastre-server
    Write-Host "  ✅ Compilation OK" -ForegroundColor Green
}

# Démarrer en arrière-plan
Start-Process -FilePath ".\bin\cadastre-server.exe" `
    -ArgumentList "-port 8080 -db postgres -db-conn `"$dbConn`"" `
    -RedirectStandardOutput "backend.log" `
    -RedirectStandardError  "backend-error.log" `
    -NoNewWindow
Start-Sleep -Seconds 2

# Vérifier le health
try {
    $health = Invoke-RestMethod -Uri "http://localhost:8080/health" -TimeoutSec 5
    Write-Host "  ✅ Backend opérationnel — http://localhost:8080" -ForegroundColor Green
} catch {
    Write-Host "  ⚠️  Backend pas encore prêt — vérifier backend-error.log" -ForegroundColor Orange
}

# ── 6. Démarrer le Frontend ───────────────────────────────────────────────────
Write-Host "`n▶ [6/6] Démarrage Frontend React (port 3000)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit -Command `"cd '$Repo\frontend'; npm run dev`""
Start-Sleep -Seconds 3
Write-Host "  ✅ Frontend disponible — http://localhost:3000" -ForegroundColor Green

# ── Résumé ────────────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "╔══════════════════════════════════════════════════════════════╗" -ForegroundColor Green
Write-Host "║   ✅  Stack GeoMobile137 OPÉRATIONNELLE                     ║" -ForegroundColor Green
Write-Host "║                                                              ║" -ForegroundColor Green
Write-Host "║   Backend  : http://localhost:8080/health                   ║" -ForegroundColor Green
Write-Host "║   Frontend : http://localhost:3000                          ║" -ForegroundColor Green
Write-Host "║   PostgreSQL: 127.0.0.1:3779 / geomobile137                ║" -ForegroundColor Green
Write-Host "║   Redis     : localhost:6379                                ║" -ForegroundColor Green
Write-Host "║                                                              ║" -ForegroundColor Green
Write-Host "║   Partage réseau : \\DESKTOP-BTI208H\geomobile137\         ║" -ForegroundColor Green
Write-Host "║   → L'autre PC peut maintenant accéder au projet           ║" -ForegroundColor Green
Write-Host "╚══════════════════════════════════════════════════════════════╝" -ForegroundColor Green
Write-Host ""
