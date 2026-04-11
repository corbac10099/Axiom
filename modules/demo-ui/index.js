// modules/demo-ui/index.js
// Exemple de module JS avec hot-reload.
// Modifiez ce fichier et sauvegardez : rechargement automatique en ~50ms.

const MODULE_ID = axiom.moduleId; // injecté par le runtime Go

function init() {
  axiom.log.info("demo-ui JS: démarrage...");

  // S'abonner au démarrage du système
  axiom.on(axiom.topics.SystemReady, function(ev) {
    axiom.log.info("demo-ui JS: système prêt, enregistrement de l'UI");
    registerUI();
  });

  // Écouter les fichiers ouverts (exemple)
  axiom.on(axiom.topics.FileOpened, function(ev) {
    if (ev.payload && ev.payload.path) {
      axiom.log.debug("demo-ui JS: fichier ouvert → " + ev.payload.path);
    }
  });
}

function stop() {
  axiom.log.info("demo-ui JS: arrêt propre");
  // Nettoyage si nécessaire (les abonnements sont automatiquement nettoyés)
}

function registerUI() {
  // Enregistrer la vue via le bus — même API qu'en Go
  var err = axiom.emit(axiom.topics.UIOpenPanel, {
    panel_id: MODULE_ID,
    title:    "Demo Module (JS)",
    content: JSON.stringify({
      moduleId:     MODULE_ID,
      icon:         "◈",
      title:        "Demo Module (JS)",
      viewType:     "replace",
      autoActivate: false,
      position:     "top",
      html:         buildHTML(),
      css:          buildCSS(),
      js:           buildClientJS(),
    }),
  });

  if (err) {
    axiom.log.error("demo-ui JS: cannot register UI: " + err.message);
    return;
  }

  // Injecter un bouton dans la statusbar droite
  axiom.emit(axiom.topics.UISlotInject, {
    slot:     "statusbar-right",
    moduleId: MODULE_ID,
    id:       MODULE_ID + "-statusbar-btn",
    html:     '<div class="status-item" style="cursor:pointer;background:rgba(147,51,234,0.2);border-radius:3px;padding:0 8px;" onclick="window.AxiomModules.switchView(\'' + MODULE_ID + '\')">◈ Demo JS</div>',
    replace:  true,
  });

  // Badge sur l'icône
  axiom.emit(axiom.topics.UIIconBadge, {
    module_id: MODULE_ID,
    count:     "js",
  });

  axiom.log.info("demo-ui JS: UI enregistrée ✓");
}

function buildHTML() {
  return `
<div class="demo-container">
  <header class="demo-header">
    <div class="demo-logo">◈</div>
    <div class="demo-header-text">
      <h1>Demo Module — JavaScript</h1>
      <p>Ce module est un fichier <code>index.js</code> — modifiez-le et sauvegardez pour le voir se recharger à chaud !</p>
    </div>
    <div class="demo-header-actions">
      <button class="demo-btn demo-btn-secondary" onclick="window.AxiomModules.switchView('editor')">← Éditeur</button>
    </div>
  </header>
  <div class="demo-grid">
    <div class="demo-card">
      <div class="demo-card-icon">⚡</div>
      <h3>Hot-reload actif</h3>
      <p>Modifiez <code>modules/demo-ui/index.js</code>, sauvegardez, et ce module se recharge automatiquement en ~50ms.</p>
    </div>
    <div class="demo-card">
      <div class="demo-card-icon">🔌</div>
      <h3>Même API Go</h3>
      <p><code>axiom.emit(topic, payload)</code> et <code>axiom.on(topic, handler)</code> fonctionnent exactement comme en Go.</p>
    </div>
    <div class="demo-card">
      <div class="demo-card-icon">🔒</div>
      <h3>Sovereign API</h3>
      <p>Le niveau de clearance du manifest.json est toujours respecté. Un module L1 ne peut pas envoyer sur <code>ui.panel.open</code>.</p>
    </div>
  </div>
  <div class="demo-console" id="demo-console">
    <div class="demo-console-header">
      <span>Console JS</span>
      <button class="demo-btn demo-btn-secondary" style="padding:2px 8px;font-size:11px" onclick="document.getElementById('demo-console-body').innerHTML=''">Effacer</button>
    </div>
    <div class="demo-console-body" id="demo-console-body">
      <div class="demo-log success">Module JS initialisé ✓ (hot-reload actif)</div>
    </div>
  </div>
</div>`;
}

function buildCSS() {
  return `
.demo-container { display:flex;flex-direction:column;height:100%;background:#1a1a2e;overflow:hidden;font-family:'Segoe UI',system-ui,sans-serif; }
.demo-header { display:flex;align-items:center;gap:16px;padding:16px 24px;background:linear-gradient(135deg,#16213e 0%,#0f3460 100%);border-bottom:1px solid #533483;flex-shrink:0; }
.demo-logo { font-size:32px;color:#a78bfa; }
.demo-header-text h1 { font-size:18px;font-weight:600;color:#e2e8f0;margin-bottom:2px; }
.demo-header-text p { font-size:12px;color:#94a3b8; }
.demo-header-text code { font-size:11px;background:rgba(0,0,0,.3);color:#7dd3fc;padding:1px 5px;border-radius:3px; }
.demo-header-actions { margin-left:auto; }
.demo-btn { padding:6px 14px;border-radius:6px;border:none;cursor:pointer;font-size:12px;font-weight:500;transition:all .15s; }
.demo-btn-secondary { background:rgba(255,255,255,.08);color:#cbd5e1;border:1px solid rgba(255,255,255,.12); }
.demo-btn-secondary:hover { background:rgba(255,255,255,.14); }
.demo-grid { display:grid;grid-template-columns:repeat(3,1fr);gap:16px;padding:20px 24px;flex:1;overflow-y:auto; }
.demo-card { background:rgba(255,255,255,.04);border:1px solid rgba(255,255,255,.08);border-radius:10px;padding:18px;display:flex;flex-direction:column;gap:8px;transition:all .2s; }
.demo-card:hover { background:rgba(255,255,255,.07);border-color:rgba(167,139,250,.3);transform:translateY(-2px); }
.demo-card-icon { font-size:24px; }
.demo-card h3 { font-size:14px;font-weight:600;color:#e2e8f0; }
.demo-card p { font-size:12px;color:#94a3b8;line-height:1.5; }
.demo-card code { font-size:11px;background:rgba(0,0,0,.3);color:#7dd3fc;padding:2px 6px;border-radius:3px;font-family:monospace; }
.demo-console { flex-shrink:0;height:140px;border-top:1px solid rgba(255,255,255,.08);display:flex;flex-direction:column; }
.demo-console-header { display:flex;justify-content:space-between;align-items:center;padding:6px 16px;background:rgba(0,0,0,.2);font-size:11px;color:#64748b;font-weight:600;text-transform:uppercase;letter-spacing:.5px; }
.demo-console-body { flex:1;overflow-y:auto;padding:6px 16px;font-family:monospace;font-size:12px; }
.demo-log { padding:2px 0; }
.demo-log.info { color:#7dd3fc; }
.demo-log.success { color:#4ade80; }
.demo-log.error { color:#f87171; }
.demo-log.warn { color:#fbbf24; }`;
}

function buildClientJS() {
  // Ce JS s'exécute côté navigateur (dans la WebView Wails)
  return `
document.addEventListener('axiom:bus-event', function(e) {
  var ev = e.detail;
  var body = document.getElementById('demo-console-body');
  if (!body || !ev) return;
  var el = document.createElement('div');
  el.className = 'demo-log info';
  el.textContent = new Date().toLocaleTimeString() + ' — bus: ' + ev.topic + (ev.source ? ' (from: ' + ev.source + ')' : '');
  body.appendChild(el);
  body.scrollTop = body.scrollHeight;
});
console.log('[demo-ui JS] client JS initialisé');`;
}