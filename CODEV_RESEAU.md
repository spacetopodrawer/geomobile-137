# 🤝 Co-développement GeoMobile137 — Partage Réseau WiFi

**PC Principal :** DESKTOP-BTI208H — `F:\geomobile137` (partagé depuis des semaines)
**PC Secondaire :** accès via `\\DESKTOP-BTI208H\geomobile137\`
**Outils IA :** Claude Code CLI + Cowork Desktop (installés sur les deux PCs)
**Mise à jour :** 2026-05-21

---

## ⚡ Démarrage immédiat

### PC Principal — lancer la stack
```powershell
cd F:\geomobile137
.\STEP2_PC_PRINCIPAL_session.ps1
```

### PC Secondaire — accès direct (partage déjà actif)
```powershell
# Le dossier est déjà accessible — accéder directement
cd \\DESKTOP-BTI208H\geomobile137

# Lancer le setup (une seule fois)
.\STEP3_PC_SECONDAIRE_setup.ps1

# Ouvrir le projet dans Claude Code
claude code \\DESKTOP-BTI208H\geomobile137

# Ou mapper en lecteur pour plus de confort (optionnel)
New-PSDrive -Name "G" -PSProvider FileSystem -Root \\DESKTOP-BTI208H\geomobile137 -Persist
claude code G:\
```

---

## 🏗️ Architecture

```
┌──────────────────────────────────┐  WiFi  ┌──────────────────────────────────┐
│  PC PRINCIPAL (DESKTOP-BTI208H)  │◄──────►│  PC SECONDAIRE                   │
│  F:\geomobile137\  (partagé)     │        │  \\DESKTOP-BTI208H\geomobile137\ │
│                                  │        │                                  │
│  Backend Go      → port 8080     │        │  Mêmes fichiers sources          │
│  PostgreSQL      → port 3779     │        │  en lecture/écriture directe     │
│  Redis           → port 6379     │        │                                  │
│  Frontend        → port 3000     │        │  Frontend propre → port 3001     │
│                                  │        │  API → DESKTOP-BTI208H:8080      │
└──────────────────────────────────┘        └──────────────────────────────────┘
           │                                              │
           └──────────── git push / git pull ─────────────┘
                  github.com/spacetopodrawer/geomobile-137
```

---

## 📋 Règles de co-développement

| # | Règle |
|---|-------|
| 1 | **Ne jamais modifier le même fichier en même temps** (pas de verrou sur partage réseau) |
| 2 | `git pull` avant de commencer à coder |
| 3 | `git commit` + `git push` en fin de session |
| 4 | Communiquer avant de toucher un fichier partagé |
| 5 | PC Principal seul héberge PostgreSQL et le backend Go |
| 6 | PC Secondaire pointe son frontend vers `http://DESKTOP-BTI208H:8080` |

---

## 🌿 Répartition des domaines

| Domaine | PC Principal | PC Secondaire |
|---------|-------------|---------------|
| Backend Go (`cmd/`, `pkg/`, `internal/`) | ✅ Lead | Review |
| Frontend React (`frontend/`) | Review | ✅ Lead |
| geo-mobile (`geo-mobile/`) | Review | ✅ Lead |
| App Android (`android/`) | Review | ✅ Lead |
| Base de données (`migrations/`) | ✅ Lead | Consultation |
| VR UE5.3 (`GeoMobile_Clean_v2/`) | ✅ Lead | — |
| Tests Go | ✅ Lead | Review |
| Tests Frontend (vitest/playwright) | Review | ✅ Lead |
| Documentation (`docs/`) | Partagé | Partagé |

---

## 🔧 Config frontend PC Secondaire

Créer `frontend/.env.local` sur le PC Secondaire uniquement :
```env
VITE_API_URL=http://DESKTOP-BTI208H:8080
VITE_WS_URL=ws://DESKTOP-BTI208H:8080
```

Démarrer sur un port différent pour éviter le conflit avec le PC principal :
```powershell
cd \\DESKTOP-BTI208H\geomobile137\frontend
npm run dev -- --port 3001
```

---

## 🤖 Claude Code & Cowork sur le PC Secondaire

```bash
# Claude Code — travailler directement dans le projet partagé
claude code \\DESKTOP-BTI208H\geomobile137

# Exemples de commandes
claude -p "Ajoute une page carte dans geo-mobile/src/pages/"
claude -p "Corrige les erreurs TypeScript dans frontend/src/"
claude -p "Génère les tests pour internal/sync/sync_engine.go"
claude commit   # Message de commit généré automatiquement
```

**Cowork :** Ouvrir → Sélectionner le dossier `\\DESKTOP-BTI208H\geomobile137` → piloter fichiers, docs, logs.

---

## 📁 Documents de référence

| Document | Emplacement |
|----------|-------------|
| Liste des outils & dépendances | `docs/GeoMobile137_Outils_Dependances.docx` |
| Guide co-développement complet | `docs/GeoMobile137_Guide_CoDeveloppement.docx` |
| Instructions démarrage stack | `CLAUDE.md` |
| Architecture master | `01_ARCHITECTURE_MASTER_PLAN.md` |
| Roadmap 2026 | `GEOMOBILE137_ROADMAP_2026.md` |
| GitHub (backup/branches) | https://github.com/spacetopodrawer/geomobile-137 |
