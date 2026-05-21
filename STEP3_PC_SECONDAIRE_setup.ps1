# ╔══════════════════════════════════════════════════════════════════════════════╗
# ║  PC SECONDAIRE — Setup initial + Script de session co-développement        ║
# ║  Projet : GeoMobile137  |  Accès : \\DESKTOP-BTI208H\geomobile137\        ║
# ║  Claude Code + Cowork déjà installés sur ce PC                            ║
# ╚══════════════════════════════════════════════════════════════════════════════╝

$NetworkPath = "\\DESKTOP-BTI208H\geomobile137"
$LocalLink   = "G:\geomobile137"   # Lettre de lecteur mappée (à adapter si besoin)

Write-Host ""
Write-Host "╔══════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║   GeoMobile137 — Setup PC Secondaire                        ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan

# ── Vérifier la connectivité réseau ───────────────────────────────────────────
Write-Host "`n▶ [1/5] Test de connectivité vers DESKTOP-BTI208H..." -ForegroundColor Yellow
$ping = Test-Connection -ComputerName "DESKTOP-BTI208H" -Count 1 -Quiet
if (-not $ping) {
    Write-Host "  ❌ PC principal injoignable." -ForegroundColor Red
    Write-Host "  → Vérifier : même réseau WiFi ? PC principal allumé ?" -ForegroundColor White
    exit 1
}
Write-Host "  ✅ DESKTOP-BTI208H joignable" -ForegroundColor Green

# ── Accéder au partage réseau ─────────────────────────────────────────────────
Write-Host "`n▶ [2/5] Connexion au partage réseau..." -ForegroundColor Yellow
if (Test-Path $NetworkPath) {
    Write-Host "  ✅ Partage accessible : $NetworkPath" -ForegroundColor Green
} else {
    Write-Host "  ❌ Partage introuvable : $NetworkPath" -ForegroundColor Red
    Write-Host "  → Sur le PC principal, exécuter STEP2_PC_PRINCIPAL_session.ps1 d'abord" -ForegroundColor White
    exit 1
}

# Mapper en lettre de lecteur (optionnel, pour confort)
$already = Get-PSDrive -Name "G" -ErrorAction SilentlyContinue
if (-not $already) {
    New-PSDrive -Name "G" -PSProvider FileSystem -Root $NetworkPath -Persist -ErrorAction SilentlyContinue | Out-Null
    Write-Host "  ✅ Lecteur G: mappé → $NetworkPath" -ForegroundColor Green
} else {
    Write-Host "  ℹ️  Lecteur G: déjà mappé" -ForegroundColor Cyan
}

# ── Lire les guides dans docs/ ────────────────────────────────────────────────
Write-Host "`n▶ [3/5] Ouverture des guides de co-développement..." -ForegroundColor Yellow
$guides = @(
    "$NetworkPath\docs\GeoMobile137_Outils_Dependances.docx",
    "$NetworkPath\docs\GeoMobile137_Guide_CoDeveloppement.docx"
)
foreach ($g in $guides) {
    if (Test-Path $g) {
        Start-Process $g
        Write-Host "  ✅ Ouvert : $(Split-Path $g -Leaf)" -ForegroundColor Green
    } else {
        Write-Host "  ⚠️  Absent : $(Split-Path $g -Leaf) — demander au PC principal de lancer STEP1_COPIER_DOCS.ps1" -ForegroundColor Orange
    }
}

# ── Vérifier les outils requis sur ce PC ─────────────────────────────────────
Write-Host "`n▶ [4/5] Vérification des outils requis sur ce PC..." -ForegroundColor Yellow
$checks = @(
    @{ name="Git";          cmd="git --version" },
    @{ name="Go 1.23+";     cmd="go version" },
    @{ name="Node.js 20+";  cmd="node --version" },
    @{ name="npm";          cmd="npm --version" },
    @{ name="Claude Code";  cmd="claude --version" }
)
$missing = @()
foreach ($c in $checks) {
    try {
        $ver = & cmd /c $c.cmd 2>&1
        Write-Host ("  ✅ {0,-15} {1}" -f $c.name, $ver) -ForegroundColor Green
    } catch {
        Write-Host ("  ❌ {0,-15} NON INSTALLÉ" -f $c.name) -ForegroundColor Red
        $missing += $c.name
    }
}
if ($missing.Count -gt 0) {
    Write-Host "`n  ⚠️  Installer les outils manquants avant de continuer." -ForegroundColor Orange
    Write-Host "  Voir : $NetworkPath\docs\GeoMobile137_Outils_Dependances.docx" -ForegroundColor White
}

# ── Ouvrir le projet dans Claude Code ────────────────────────────────────────
Write-Host "`n▶ [5/5] Ouverture du projet dans Claude Code..." -ForegroundColor Yellow
Write-Host "  Le projet s'ouvre directement depuis le partage réseau." -ForegroundColor White
Write-Host "  Claude Code peut lire et modifier les fichiers via le chemin réseau." -ForegroundColor White

$resp = Read-Host "  Ouvrir Claude Code sur \\DESKTOP-BTI208H\geomobile137 ? (o/n)"
if ($resp -eq "o") {
    # Claude Code s'ouvre dans le dossier réseau
    Start-Process "claude" -ArgumentList "code `"$NetworkPath`"" -ErrorAction SilentlyContinue
    if ($LASTEXITCODE -ne 0) {
        # Fallback: ouvrir dans l'explorateur pour naviguer
        Start-Process "explorer.exe" $NetworkPath
        Write-Host "  → Dans Claude Code, taper : claude code G:\" -ForegroundColor White
    }
}

# ── Résumé & règles de co-développement ──────────────────────────────────────
Write-Host ""
Write-Host "╔══════════════════════════════════════════════════════════════════════╗" -ForegroundColor Green
Write-Host "║   ✅  PC Secondaire prêt pour le co-développement                  ║" -ForegroundColor Green
Write-Host "╠══════════════════════════════════════════════════════════════════════╣" -ForegroundColor Green
Write-Host "║                                                                      ║" -ForegroundColor Green
Write-Host "║   📁 Projet   : \\DESKTOP-BTI208H\geomobile137\  (ou G:\)          ║" -ForegroundColor Green
Write-Host "║   📄 Guides   : docs\GeoMobile137_Guide_CoDeveloppement.docx       ║" -ForegroundColor Green
Write-Host "║   🤖 Claude Code : claude code G:\                                 ║" -ForegroundColor Green
Write-Host "║   🖥️  Cowork      : ouvrir le dossier G:\ dans Cowork              ║" -ForegroundColor Green
Write-Host "║                                                                      ║" -ForegroundColor Green
Write-Host "║   ⚠️  RÈGLES DE CO-DÉVELOPPEMENT (partage réseau)                  ║" -ForegroundColor Yellow
Write-Host "║   1. Ne pas modifier le même fichier en même temps                 ║" -ForegroundColor White
Write-Host "║   2. Toujours faire git pull avant de commencer à coder            ║" -ForegroundColor White
Write-Host "║   3. Faire git commit + git push après chaque session              ║" -ForegroundColor White
Write-Host "║   4. Communiquer (chat/voix) avant de modifier un fichier partagé  ║" -ForegroundColor White
Write-Host "║   5. Backend Go : seul le PC principal le lance (port 8080)        ║" -ForegroundColor White
Write-Host "║   6. PostgreSQL  : seul le PC principal héberge la BDD (port 3779) ║" -ForegroundColor White
Write-Host "║   7. Frontend : chaque PC peut lancer son propre npm run dev       ║" -ForegroundColor White
Write-Host "╚══════════════════════════════════════════════════════════════════════╝" -ForegroundColor Green
Write-Host ""
