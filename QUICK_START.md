# ⚡ QUICK START — Axiom v0.3.0 (15 minutes)

## 🎯 GOAL
Voir les améliorations de Axiom v0.3.0 en fonctionnement en **15 minutes**.

---

## 📋 PRÉ-REQUIS (2 min)

```bash
✅ Go 1.22+
✅ Git
✅ Wails v2 (optionnel pour interface graphique)
```

**Check:**
```bash
go version        # Should be 1.22+
wails version     # Should be installed
```

---

## 🚀 INSTALLATION (5 min)

### Étape 1: Copier les fichiers

```bash
# Depuis votre projet Axiom
cd your-axiom-project

# Copier les composants
cp axiom-ui-components.css frontend/
cp axiom-ui-components.js frontend/

# Copier le générateur
mkdir -p tools
cp tools-module-generator.go tools/generator.go

# Copier le frontend d'exemple
cp frontend-axiom-v0.3.html frontend/index-v0.3.html
```

### Étape 2: Mettre à jour frontend/index.html

Ajouter au début de `<head>`:
```html
<link rel="stylesheet" href="./axiom-ui-components.css">
<script src="./axiom-ui-components.js" defer></script>
```

**C'est prêt!** ✨

---

## 🎮 ESSAYER LES COMPOSANTS (5 min)

### Option 1: Via le Frontend Exemple
```bash
# Lancer Wails
wails dev

# Ouvrir http://localhost:34115
# Voir le frontend v0.3 avec tous les composants!
```

### Option 2: Via HTML Simple
```html
<!DOCTYPE html>
<html>
<head>
    <link rel="stylesheet" href="./axiom-ui-components.css">
</head>
<body>
    <!-- Utiliser les composants! -->
    <axiom-button>Click me!</axiom-button>
    <axiom-input placeholder="Type..."></axiom-input>
    
    <script src="./axiom-ui-components.js"></script>
    <script>
        // C'est tout! Les composants sont actifs
        console.log(window.Axiom.UI);
    </script>
</body>
</html>
```

---

## 🛠️ GÉNÉRER UN MODULE (3 min)

### En 30 secondes
```bash
go run tools/generator.go -name my-awesome-module -level L1

# ✨ Généré:
# - modules/my-awesome-module/manifest.json
# - modules/my-awesome-module/myawesomemodule.go
# - modules/my-awesome-module/myawesomemodule_test.go
# - modules/my-awesome-module/README.md
```

### Utiliser le module
```go
// Dans main.go
import mymod "github.com/axiom-ide/axiom/modules/my-awesome-module"

func main() {
    // ...
    runner.Register(mymod.New(slog.Default()))
    // ...
}
```

---

## 🎨 VOIR LA DÉMO (5 min)

### Ouvrir le Frontend v0.3 dans le Navigateur

```bash
# 1. Lancer Wails
wails dev

# 2. Ouvrir http://localhost:34115
# ou http://localhost:34115/index-v0.3.html

# 3. Vous verrez:
# ✅ Toolbar avec buttons
# ✅ Sidebar avec fichiers + modules
# ✅ Editor avec onglets
# ✅ AI Panel
# ✅ Theme selector
```

**Cliquer sur les éléments pour tester l'interactivité!**

---

## 💡 EXEMPLES RAPIDES (à essayer maintenant)

### Exemple 1: Créer un Button
```javascript
// Dans la console du navigateur
const btn = axiomButton({
    label: 'Hello World',
    action: 'hello',
    variant: 'success'
});
document.body.appendChild(btn);
```

**Résultat:** Button vert apparaît! ✨

### Exemple 2: Créer un Alert
```javascript
axiomAlert({
    message: 'Welcome to Axiom v0.3.0!',
    type: 'success',
    duration: 3000
});
```

**Résultat:** Notification verte qui disparaît après 3s! 🎉

### Exemple 3: Créer un Panel
```javascript
const panel = axiomPanel({
    title: '🎉 My First Panel',
    content: '<p>This was created in v0.3.0!</p>',
    collapsible: true,
    footer: '<axiom-button>Close</axiom-button>'
});
document.body.appendChild(panel);
```

**Résultat:** Panel complet avec header, body, footer! 📦

### Exemple 4: Écouter les Événements
```javascript
axiomOn('button:clicked', (data) => {
    console.log('Button clicked!', data);
});
```

**Cliquer sur un button:** Le log s'affiche! 🔔

---

## 📚 DOCUMENTATION RAPIDE

| Besoin | Aller vers | Temps |
|--------|------------|-------|
| Comprendre v0.3 | `AXIOM_v0.3_IMPROVEMENTS.md` | 10 min |
| Apprendre les composants | `AXIOM_v0.3_COMPONENTS_GUIDE.md` | 30 min |
| Générer un module | `AXIOM_v0.3_COMPONENTS_GUIDE.md` → Section "Module Generator" | 5 min |
| Voir un exemple | `frontend-axiom-v0.3.html` | 10 min |
| Tout comprendre | `INDEX_AXIOM_v0.3.md` | 30 min |

---

## 🎓 STRUCTURE DES FICHIERS

```
Vous avez reçu:

axiom-ui-components.css      ← Styles (copier en frontend/)
axiom-ui-components.js       ← JS (copier en frontend/)
frontend-axiom-v0.3.html     ← Frontend d'exemple (copier en frontend/)
tools-module-generator.go    ← Generator (copier en tools/)

+ Documentations:
AXIOM_v0.3_COMPONENTS_GUIDE.md   ← Comment utiliser
AXIOM_v0.3_IMPROVEMENTS.md        ← Quoi de neuf
INDEX_AXIOM_v0.3.md              ← Guide complet
CHANGELOG_v0.3.md                ← Historique détaillé
QUICK_START.md                   ← Ce fichier!
```

---

## ✅ CHECKLIST RAPIDE

- [ ] Go 1.22+ installé
- [ ] Axiom cloné et prêt
- [ ] `axiom-ui-components.css` copié en `frontend/`
- [ ] `axiom-ui-components.js` copié en `frontend/`
- [ ] `index.html` mis à jour avec les <link> et <script>
- [ ] `tools/generator.go` copié
- [ ] Testé: `go run tools/generator.go -name test-module -level L0`
- [ ] Lancé: `go run ./main.go`
- [ ] Vu les composants en action

**Tout fait?** Continuez à `AXIOM_v0.3_COMPONENTS_GUIDE.md`! 🚀

---

## 🚨 PROBLÈMES COURANTS

### "Composants ne s'affichent pas"
```html
<!-- Vérifier que tu as mis au bon endroit -->
<link rel="stylesheet" href="./axiom-ui-components.css">
<script src="./axiom-ui-components.js" defer></script>
```

### "axiomButton() n'existe pas"
```javascript
// Attendre le chargement
document.addEventListener('DOMContentLoaded', () => {
    axiomButton({...});
});
```

### "Generator ne fonctionne pas"
```bash
# Vérifier qu'il est au bon endroit
ls tools/generator.go

# Relancer avec chemin correct
cd tools && go run generator.go -name test -level L1
```

---

## 🎯 PROCHAINES ÉTAPES

### Immédiatement (maintenant)
1. Copier les fichiers (5 min)
2. Tester les composants (5 min)
3. Générer un module (2 min)

### Après (20 min)
4. Lire `AXIOM_v0.3_IMPROVEMENTS.md`
5. Regarder `frontend-axiom-v0.3.html`
6. Créer votre propre interface

### Avancé (1-2h)
7. Créer un module custom
8. Intégrer l'IA
9. Déployer en production

---

## 🎉 C'EST PRÊT!

Vous avez maintenant:
- ✅ 8 composants réutilisables
- ✅ Un générateur de modules
- ✅ Un frontend d'exemple
- ✅ Une documentation complète

**Axiom v0.3.0 est 8.5/10 en simplicité!** 🚀

---

## 📞 BESOIN D'AIDE?

```bash
# Voir la documentation complète
cat AXIOM_v0.3_COMPONENTS_GUIDE.md

# Voir la liste complète des améliorations
cat AXIOM_v0.3_IMPROVEMENTS.md

# Lire le changelog complet
cat CHANGELOG_v0.3.md
```

---

**Prêt? Commencez maintenant!** ⚡

*Lancer: `wails dev` ou `go run ./main.go`*

✨ **Enjoy Axiom v0.3.0!** ✨