# 📍 INSTALLATION DÉTAILLÉE — Où placer chaque fichier

## 🎯 OBJECTIF
Vous guider pas à pas pour placer les fichiers au bon endroit.

---

## 📁 STRUCTURE ACTUELLE DE VOTRE PROJET AXIOM

```
axiom/                          ← Racine du projet
├── main.go                      ← Point d'entrée
├── go.mod
├── go.sum
├── README.md
├── .gitignore
│
├── api/                         ← Types d'événements
│   └── events.go
│
├── core/                        ← Cœur du moteur
│   ├── bus/
│   ├── config/
│   ├── engine/
│   ├── filesystem/
│   ├── module/
│   ├── orchestrator/
│   ├── registry/
│   ├── security/
│   ├── tabs/
│   └── workspace/
│
├── modules/                     ← Modules chargés
│   ├── ai-assistant/
│   ├── file-explorer/
│   └── theme-manager/
│
├── adapters/
│   └── wails/
│
├── frontend/                    ← FILES ICI! ⭐
│   ├── index.html              (METTRE À JOUR!)
│   └── ...
│
└── pkg/
    ├── logger/
    └── uid/
```

---

## 📦 FICHIERS À PLACER

### ✅ Fichier 1: `axiom-ui-components.css`

**Source:** Vous avez reçu ce fichier  
**Destination:** `frontend/`  
**Chemin complet:** `axiom/frontend/axiom-ui-components.css`

```bash
# Terminal - Se placer à la racine du projet
cd /chemin/vers/axiom

# Copier le fichier
cp /chemin/vers/axiom-ui-components.css frontend/

# Vérifier
ls -la frontend/axiom-ui-components.css
```

**Structure après placement:**
```
axiom/frontend/
├── axiom-ui-components.css      ✅ NOUVEAU
├── axiom-ui-components.js       (à venir)
├── index.html                   (à mettre à jour)
└── ...
```

---

### ✅ Fichier 2: `axiom-ui-components.js`

**Source:** Vous avez reçu ce fichier  
**Destination:** `frontend/`  
**Chemin complet:** `axiom/frontend/axiom-ui-components.js`

```bash
# Terminal - Se placer à la racine du projet
cd /chemin/vers/axiom

# Copier le fichier
cp /chemin/vers/axiom-ui-components.js frontend/

# Vérifier
ls -la frontend/axiom-ui-components.js
```

**Structure après placement:**
```
axiom/frontend/
├── axiom-ui-components.css      ✅
├── axiom-ui-components.js       ✅ NOUVEAU
├── index.html                   (à mettre à jour)
└── ...
```

---

### ✅ Fichier 3: `frontend-axiom-v0.3.html`

**Source:** Vous avez reçu ce fichier  
**Destination:** `frontend/`  
**Chemin complet:** `axiom/frontend/index-v0.3.html` (renommé!)

```bash
# Terminal - Se placer à la racine du projet
cd /chemin/vers/axiom

# Copier et renommer
cp /chemin/vers/frontend-axiom-v0.3.html frontend/index-v0.3.html

# Vérifier
ls -la frontend/index-v0.3.html
```

**Pourquoi renommer?** Parce que `index.html` existe déjà!

**Structure après placement:**
```
axiom/frontend/
├── axiom-ui-components.css      ✅
├── axiom-ui-components.js       ✅
├── index.html                   (ancien, à mettre à jour)
├── index-v0.3.html              ✅ NOUVEAU
└── ...
```

**Note:** Plus tard, tu peux remplacer `index.html` par la nouvelle version.

---

### ✅ Fichier 4: `tools-module-generator.go`

**Source:** Vous avez reçu ce fichier  
**Destination:** `tools/` (créer le dossier s'il n'existe pas)  
**Chemin complet:** `axiom/tools/generator.go`

```bash
# Terminal - Se placer à la racine du projet
cd /chemin/vers/axiom

# Créer le dossier tools s'il n'existe pas
mkdir -p tools

# Copier et renommer
cp /chemin/vers/tools-module-generator.go tools/generator.go

# Vérifier
ls -la tools/generator.go
```

**Structure après placement:**
```
axiom/tools/                    ✅ NOUVEAU DOSSIER
├── generator.go                 ✅ NOUVEAU
```

---

### 📚 Fichier 5: `AXIOM_v0.3_COMPONENTS_GUIDE.md`

**Source:** Vous avez reçu ce fichier  
**Destination:** Racine du projet  
**Chemin complet:** `axiom/AXIOM_v0.3_COMPONENTS_GUIDE.md`

```bash
# Terminal - Se placer à la racine du projet
cd /chemin/vers/axiom

# Copier le fichier
cp /chemin/vers/AXIOM_v0.3_COMPONENTS_GUIDE.md ./

# Vérifier
ls -la AXIOM_v0.3_COMPONENTS_GUIDE.md
```

**Structure après placement:**
```
axiom/
├── AXIOM_v0.3_COMPONENTS_GUIDE.md      ✅ NOUVEAU
├── main.go
├── ...
```

---

### 📚 Fichier 6: `AXIOM_v0.3_IMPROVEMENTS.md`

**Source:** Vous avez reçu ce fichier  
**Destination:** Racine du projet  
**Chemin complet:** `axiom/AXIOM_v0.3_IMPROVEMENTS.md`

```bash
# Terminal - Se placer à la racine du projet
cd /chemin/vers/axiom

# Copier le fichier
cp /chemin/vers/AXIOM_v0.3_IMPROVEMENTS.md ./

# Vérifier
ls -la AXIOM_v0.3_IMPROVEMENTS.md
```

---

### 📚 Fichier 7: `INDEX_AXIOM_v0.3.md`

**Source:** Vous avez reçu ce fichier  
**Destination:** Racine du projet  
**Chemin complet:** `axiom/INDEX_AXIOM_v0.3.md`

```bash
# Terminal - Se placer à la racine du projet
cd /chemin/vers/axiom

# Copier le fichier
cp /chemin/vers/INDEX_AXIOM_v0.3.md ./

# Vérifier
ls -la INDEX_AXIOM_v0.3.md
```

---

### 📚 Fichier 8: `CHANGELOG_v0.3.md`

**Source:** Vous avez reçu ce fichier  
**Destination:** Racine du projet  
**Chemin complet:** `axiom/CHANGELOG_v0.3.md`

```bash
# Terminal - Se placer à la racine du projet
cd /chemin/vers/axiom

# Copier le fichier
cp /chemin/vers/CHANGELOG_v0.3.md ./

# Vérifier
ls -la CHANGELOG_v0.3.md
```

---

### 📚 Fichier 9: `QUICK_START.md`

**Source:** Vous avez reçu ce fichier  
**Destination:** Racine du projet  
**Chemin complet:** `axiom/QUICK_START.md`

```bash
# Terminal - Se placer à la racine du projet
cd /chemin/vers/axiom

# Copier le fichier
cp /chemin/vers/QUICK_START.md ./

# Vérifier
ls -la QUICK_START.md
```

---

### 📚 Fichier 10: `FINAL_SUMMARY.md`

**Source:** Vous avez reçu ce fichier  
**Destination:** Racine du projet  
**Chemin complet:** `axiom/FINAL_SUMMARY.md`

```bash
# Terminal - Se placer à la racine du projet
cd /chemin/vers/axiom

# Copier le fichier
cp /chemin/vers/FINAL_SUMMARY.md ./

# Vérifier
ls -la FINAL_SUMMARY.md
```

---

## ✅ RÉSUMÉ: OÙ PLACER CHAQUE FICHIER

| Fichier | Destination | Chemin Complet | Renommer? |
|---------|-------------|---|---|
| `axiom-ui-components.css` | `frontend/` | `axiom/frontend/axiom-ui-components.css` | Non |
| `axiom-ui-components.js` | `frontend/` | `axiom/frontend/axiom-ui-components.js` | Non |
| `frontend-axiom-v0.3.html` | `frontend/` | `axiom/frontend/index-v0.3.html` | **OUI** |
| `tools-module-generator.go` | `tools/` | `axiom/tools/generator.go` | **OUI** |
| `AXIOM_v0.3_COMPONENTS_GUIDE.md` | `.` | `axiom/AXIOM_v0.3_COMPONENTS_GUIDE.md` | Non |
| `AXIOM_v0.3_IMPROVEMENTS.md` | `.` | `axiom/AXIOM_v0.3_IMPROVEMENTS.md` | Non |
| `INDEX_AXIOM_v0.3.md` | `.` | `axiom/INDEX_AXIOM_v0.3.md` | Non |
| `CHANGELOG_v0.3.md` | `.` | `axiom/CHANGELOG_v0.3.md` | Non |
| `QUICK_START.md` | `.` | `axiom/QUICK_START.md` | Non |
| `FINAL_SUMMARY.md` | `.` | `axiom/FINAL_SUMMARY.md` | Non |

---

## 🚀 INSTALLATION COMPLÈTE (Copier-Coller)

### Option 1: Copier chaque fichier individuellement

```bash
# Se placer à la racine du projet
cd /chemin/vers/axiom

# 1. Fichiers frontend (CSS + JS)
cp /chemin/vers/axiom-ui-components.css frontend/
cp /chemin/vers/axiom-ui-components.js frontend/
cp /chemin/vers/frontend-axiom-v0.3.html frontend/index-v0.3.html

# 2. Générateur de modules
mkdir -p tools
cp /chemin/vers/tools-module-generator.go tools/generator.go

# 3. Documentations
cp /chemin/vers/AXIOM_v0.3_COMPONENTS_GUIDE.md ./
cp /chemin/vers/AXIOM_v0.3_IMPROVEMENTS.md ./
cp /chemin/vers/INDEX_AXIOM_v0.3.md ./
cp /chemin/vers/CHANGELOG_v0.3.md ./
cp /chemin/vers/QUICK_START.md ./
cp /chemin/vers/FINAL_SUMMARY.md ./

# Vérifier
ls -la frontend/axiom-ui-components.*
ls -la tools/generator.go
ls -la *.md | grep -i axiom
```

### Option 2: Script Bash (Automatisé)

Créer un fichier `install-v0.3.sh`:

```bash
#!/bin/bash

# Installation Axiom v0.3.0
# Usage: bash install-v0.3.sh /path/to/files

FILES_DIR=$1

if [ -z "$FILES_DIR" ]; then
    echo "Usage: bash install-v0.3.sh /path/to/downloaded/files"
    exit 1
fi

echo "🚀 Installing Axiom v0.3.0..."

# Frontend files
echo "📦 Installing frontend files..."
cp "$FILES_DIR/axiom-ui-components.css" frontend/
cp "$FILES_DIR/axiom-ui-components.js" frontend/
cp "$FILES_DIR/frontend-axiom-v0.3.html" frontend/index-v0.3.html

# Tools
echo "🛠️  Installing tools..."
mkdir -p tools
cp "$FILES_DIR/tools-module-generator.go" tools/generator.go

# Documentation
echo "📚 Installing documentation..."
cp "$FILES_DIR/AXIOM_v0.3_COMPONENTS_GUIDE.md" ./
cp "$FILES_DIR/AXIOM_v0.3_IMPROVEMENTS.md" ./
cp "$FILES_DIR/INDEX_AXIOM_v0.3.md" ./
cp "$FILES_DIR/CHANGELOG_v0.3.md" ./
cp "$FILES_DIR/QUICK_START.md" ./
cp "$FILES_DIR/FINAL_SUMMARY.md" ./

echo "✅ Installation complete!"
echo ""
echo "📍 Files installed:"
ls -la frontend/axiom-ui-components.* 2>/dev/null | awk '{print "   " $NF}'
ls -la tools/generator.go 2>/dev/null | awk '{print "   " $NF}'
echo "   AXIOM_v0.3_COMPONENTS_GUIDE.md"
echo "   AXIOM_v0.3_IMPROVEMENTS.md"
echo "   INDEX_AXIOM_v0.3.md"
echo "   CHANGELOG_v0.3.md"
echo "   QUICK_START.md"
echo "   FINAL_SUMMARY.md"
echo ""
echo "🚀 Next steps:"
echo "   1. Read: QUICK_START.md"
echo "   2. Run: wails dev"
echo "   3. Generate module: go run tools/generator.go -name test-module -level L1"
```

**Utilisation:**
```bash
# Rendre exécutable
chmod +x install-v0.3.sh

# Lancer
./install-v0.3.sh /chemin/vers/fichiers/téléchargés
```

---

## 🔍 VÉRIFICATION APRÈS INSTALLATION

### Vérifier que tout est en place

```bash
# Se placer à la racine du projet
cd /chemin/vers/axiom

# Vérifier frontend
echo "✅ Frontend files:"
ls -la frontend/axiom-ui-components.css frontend/axiom-ui-components.js frontend/index-v0.3.html

# Vérifier tools
echo "✅ Tools:"
ls -la tools/generator.go

# Vérifier documentations
echo "✅ Documentation files:"
ls -la AXIOM_v0.3_* CHANGELOG_v0.3.md QUICK_START.md FINAL_SUMMARY.md

# Test complet
echo "✅ Checking content..."
head -n 1 frontend/axiom-ui-components.css | grep -q "/* ===" && echo "   ✓ CSS file is correct"
head -n 1 frontend/axiom-ui-components.js | grep -q "/* ===" && echo "   ✓ JS file is correct"
head -n 1 tools/generator.go | grep -q "^//" && echo "   ✓ Generator file is correct"
```

**Résultat attendu:**
```
✅ Frontend files:
-rw-r--r--  axiom/frontend/axiom-ui-components.css
-rw-r--r--  axiom/frontend/axiom-ui-components.js
-rw-r--r--  axiom/frontend/index-v0.3.html

✅ Tools:
-rw-r--r--  axiom/tools/generator.go

✅ Documentation files:
-rw-r--r--  axiom/AXIOM_v0.3_COMPONENTS_GUIDE.md
-rw-r--r--  axiom/AXIOM_v0.3_IMPROVEMENTS.md
-rw-r--r--  axiom/INDEX_AXIOM_v0.3.md
-rw-r--r--  axiom/CHANGELOG_v0.3.md
-rw-r--r--  axiom/QUICK_START.md
-rw-r--r--  axiom/FINAL_SUMMARY.md

✅ Checking content...
   ✓ CSS file is correct
   ✓ JS file is correct
   ✓ Generator file is correct
```

---

## 📝 METTRE À JOUR `frontend/index.html`

Maintenant que les fichiers sont en place, il faut mettre à jour ton `index.html`.

### Chercher ces lignes dans `frontend/index.html`:

```html
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Axiom IDE</title>
    <!-- Ajouter ICI ⬇️ -->
</head>
```

### Ajouter ces deux lignes après `<title>`:

```html
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Axiom IDE v0.3.0</title>
    
    <!-- 🆕 Axiom UI Components Library v0.3.0 -->
    <link rel="stylesheet" href="./axiom-ui-components.css">
    <script src="./axiom-ui-components.js" defer></script>
    
    <!-- Reste de tes <link> et <script> -->
</head>
```

### Avant (v0.2):
```html
<head>
    <title>Axiom IDE</title>
    <style>/* Ton CSS */</style>
</head>
```

### Après (v0.3):
```html
<head>
    <title>Axiom IDE v0.3.0</title>
    <link rel="stylesheet" href="./axiom-ui-components.css">  ✅ NOUVEAU
    <script src="./axiom-ui-components.js" defer></script>    ✅ NOUVEAU
    <style>/* Ton CSS */</style>
</head>
```

---

## 🧪 TESTER APRÈS INSTALLATION

### Test 1: Vérifier que les composants se chargent

```bash
# Terminal
cd axiom

# Lancer Wails (ou go run ./main.go)
wails dev

# Ouvrir http://localhost:34115 dans le navigateur
# Ouvrir la console du navigateur (F12)
# Taper dans la console:
console.log(window.Axiom.UI)
# Doit afficher: AxiomComponents { ... }
```

**Résultat attendu:**
```javascript
> console.log(window.Axiom.UI)
AxiomComponents {
  components: Map(0)
  init: ƒ init()
  initTabs: ƒ initTabs()
  ...
}
```

### Test 2: Tester un composant

```javascript
// Dans la console du navigateur
axiomButton({ label: 'Test' })
```

**Résultat attendu:** Un bouton bleu "Test" apparaît! ✨

### Test 3: Générer un module

```bash
# Terminal
cd axiom
go run tools/generator.go -name test-module -level L0

# Vérifier les fichiers générés
ls -la modules/test-module/
# Doit montrer:
# - manifest.json
# - testmodule.go
# - testmodule_test.go
# - README.md
```

---

## 🎯 ÉTAPES FINALES

### Après avoir placé tous les fichiers:

1. **Vérifier l'installation** ✅
   ```bash
   ls -la frontend/axiom-ui-components.* tools/generator.go
   ```

2. **Mettre à jour index.html** ✅
   ```bash
   # Ajouter les deux lignes dans <head>
   ```

3. **Lancer Axiom** ✅
   ```bash
   wails dev
   # ou
   go run ./main.go
   ```

4. **Tester les composants** ✅
   ```javascript
   // Dans la console
   axiomButton({label: 'Test'})
   ```

5. **Générer un module** ✅
   ```bash
   go run tools/generator.go -name my-module -level L1
   ```

6. **Lire la documentation** ✅
   ```bash
   cat QUICK_START.md
   ```

---

## 📍 STRUCTURE FINALE APRÈS INSTALLATION

```
axiom/
├── 📄 main.go
├── 📄 go.mod
├── 📄 README.md
├── 📄 QUICK_START.md                    ✅ NOUVEAU
├── 📄 AXIOM_v0.3_COMPONENTS_GUIDE.md    ✅ NOUVEAU
├── 📄 AXIOM_v0.3_IMPROVEMENTS.md        ✅ NOUVEAU
├── 📄 INDEX_AXIOM_v0.3.md               ✅ NOUVEAU
├── 📄 CHANGELOG_v0.3.md                 ✅ NOUVEAU
├── 📄 FINAL_SUMMARY.md                  ✅ NOUVEAU
│
├── 📁 frontend/
│   ├── 📄 index.html                    (METTRE À JOUR!)
│   ├── 📄 axiom-ui-components.css       ✅ NOUVEAU
│   ├── 📄 axiom-ui-components.js        ✅ NOUVEAU
│   └── 📄 index-v0.3.html               ✅ NOUVEAU
│
├── 📁 tools/                            ✅ NOUVEAU DOSSIER
│   └── 📄 generator.go                  ✅ NOUVEAU
│
├── 📁 core/
│   ├── ...
│   └── ...
│
├── 📁 modules/
│   ├── ai-assistant/
│   ├── file-explorer/
│   └── theme-manager/
│
└── ... (autres dossiers inchangés)
```

---

## ❓ QUESTIONS FRÉQUENTES

### Q: "Je n'arrive pas à trouver mon dossier axiom"
**R:** 
```bash
# Cherche le dossier
find ~ -type d -name "axiom" 2>/dev/null

# Ou demande où est le projet
pwd  # Affiche le répertoire courant
```

### Q: "Dois-je renommer tous les fichiers?"
**R:** Non, seulement:
- `frontend-axiom-v0.3.html` → `index-v0.3.html`
- `tools-module-generator.go` → `generator.go`

### Q: "Puis-je supprimer l'ancien index.html?"
**R:** Non! Garde-le comme backup. Tu peux le renommer:
```bash
cd axiom/frontend
mv index.html index-v0.2.html
mv index-v0.3.html index.html  # Remplacer par la nouvelle version
```

### Q: "Où est le dossier tools s'il n'existe pas?"
**R:** Le créer:
```bash
cd axiom
mkdir -p tools
```

### Q: "Je veux que v0.2 et v0.3 coexistent"
**R:** Aucun problème! Les deux versions peuvent coexister:
```
frontend/
├── index.html           (v0.2)
├── index-v0.3.html      (v0.3)  ← Utiliser celle-ci
├── axiom-ui-components.css      (v0.3)
└── axiom-ui-components.js       (v0.3)
```

---

## ✨ C'EST PRÊT!

Tous les fichiers sont en place! 

**Prochaines étapes:**
1. Vérifier l'installation ✅
2. Lire `QUICK_START.md` (5 min)
3. Lancer `wails dev` ou `go run ./main.go`
4. Tester les composants

---

**Vous êtes maintenant prêt pour Axiom v0.3.0!** 🚀

👉 **Lisez:** `QUICK_START.md`