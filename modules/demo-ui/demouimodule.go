// Exemple : modules/demo-ui/demouimodule.go
// Démontre toutes les capacités du Module System UI d'Axiom.
//
// Ce module :
//   - Enregistre une vue complète (replace) avec HTML/CSS/JS propre
//   - Injecte un bouton dans la statusbar droite
//   - Injecte un badge dans l'activity bar
//   - Change le branding de l'app (logo + couleurs)
//
// Clearance L2 requis (ui.module.register, ui.slot.inject, ui.app.branding = L3 → utilise L3 ici).

package demouimodule

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/axiom-ide/axiom/api"
	"github.com/axiom-ide/axiom/core/module"
	"github.com/axiom-ide/axiom/core/security"
)

type DemoUIModule struct {
	module.BaseModule
}

func New(logger *slog.Logger) *DemoUIModule {
	return &DemoUIModule{
		BaseModule: module.NewBase(
			"demo-ui",
			"Demo UI Module",
			security.L2,
			logger,
		),
	}
}

func (m *DemoUIModule) Init(ctx context.Context, d module.Dispatcher, s module.Subscriber) error {
	m.BaseInit(ctx, d, s)

	m.On(api.TopicSystemReady, func(ev api.Event) {
		m.Logger().Info("demo-ui: system ready, registering UI…")
		m.registerUI()
	})

	return nil
}

func (m *DemoUIModule) Stop() error {
	return m.BaseStop()
}

func (m *DemoUIModule) registerUI() {
	// ── 1. Enregistrer la vue principale ──────────────────────────
	reg := api.PayloadUIModuleRegister{
		ModuleID: "demo-ui",
		Icon:     "◈",
		Title:    "Demo Module",
		ViewType: api.ViewTypeReplace,
		AutoActivate: false,
		Position: "top",
		HTML:     buildHTML(),
		CSS:      buildCSS(),
		JS:       buildJS(),
	}

	regJSON, _ := json.Marshal(reg)
	// On sérialise le PayloadUIModuleRegister dans le Content de PayloadUIPanel
	// pour utiliser le topic standard ui.panel.open (L2)
	if err := m.Emit(api.TopicUIOpenPanel, api.PayloadUIPanel{
		PanelID: "demo-ui",
		Title:   "Demo Module",
		Content: string(regJSON),
	}); err != nil {
		m.Logger().Error("demo-ui: cannot register view", slog.String("error", err.Error()))
	}

	// ── 2. Injecter un bouton dans la statusbar droite ─────────────
	if err := m.Emit(api.TopicUISlotInject, api.PayloadUISlotInject{
		Slot:     api.SlotStatusbarRight,
		ModuleID: "demo-ui",
		ID:       "demo-ui-statusbar-btn",
		HTML: `<div class="status-item" style="cursor:pointer;background:rgba(147,51,234,0.2);border-radius:3px;padding:0 8px;"
		             onclick="window.AxiomModules.switchView('demo-ui')">
		         ◈ Demo
		       </div>`,
		Replace: true,
	}); err != nil {
		m.Logger().Warn("demo-ui: cannot inject statusbar slot", slog.String("error", err.Error()))
	}

	// ── 3. Badge sur l'icône ───────────────────────────────────────
	if err := m.Emit(api.TopicUIIconBadge, api.PayloadUIIconBadge{
		ModuleID: "demo-ui",
		Count:    "new",
	}); err != nil {
		m.Logger().Warn("demo-ui: cannot set badge", slog.String("error", err.Error()))
	}
}

// ── HTML de la vue ─────────────────────────────────────────────────

func buildHTML() string {
	return `
<div class="demo-container">
  <header class="demo-header">
    <div class="demo-logo">◈</div>
    <div class="demo-header-text">
      <h1>Demo Module</h1>
      <p>Vue complète gérée par le module — HTML/CSS/JS autonomes</p>
    </div>
    <div class="demo-header-actions">
      <button class="demo-btn demo-btn-primary" onclick="DemoModule.refresh()">↻ Refresh</button>
      <button class="demo-btn demo-btn-secondary" onclick="window.AxiomModules.switchView('editor')">← Éditeur</button>
    </div>
  </header>

  <div class="demo-grid">
    <div class="demo-card">
      <div class="demo-card-icon">🚀</div>
      <h3>Vues complètes</h3>
      <p>Un module peut remplacer entièrement la zone principale avec sa propre interface HTML/CSS/JS.</p>
      <code>viewType: "replace"</code>
    </div>
    <div class="demo-card">
      <div class="demo-card-icon">⬛</div>
      <h3>Takeover</h3>
      <p>Mode position:absolute qui recouvre tout le content area sans toucher la sidebar.</p>
      <code>viewType: "takeover"</code>
    </div>
    <div class="demo-card">
      <div class="demo-card-icon">🌫</div>
      <h3>Overlay</h3>
      <p>Modale semi-transparente par-dessus toute l'application. Fermable avec Escape.</p>
      <code>viewType: "overlay"</code>
      <button class="demo-btn demo-btn-primary" style="margin-top:12px" onclick="DemoModule.showOverlay()">Ouvrir un overlay</button>
    </div>
    <div class="demo-card">
      <div class="demo-card-icon">🧩</div>
      <h3>Slots</h3>
      <p>Injection dans sidebar, statusbar (gauche/droite), tab bar, panel tabs, activity bar bas.</p>
      <code>TopicUISlotInject</code>
    </div>
    <div class="demo-card">
      <div class="demo-card-icon">🎨</div>
      <h3>Branding</h3>
      <p>Un module L3 peut modifier le logo, le nom de l'app et les couleurs de titlebar/statusbar.</p>
      <code>TopicUIAppBranding</code>
      <button class="demo-btn demo-btn-secondary" style="margin-top:12px" onclick="DemoModule.testBranding()">Tester</button>
    </div>
    <div class="demo-card">
      <div class="demo-card-icon">🔴</div>
      <h3>Badges</h3>
      <p>Afficher un compteur ou un label sur l'icône du module dans l'activity bar.</p>
      <code>TopicUIIconBadge</code>
      <button class="demo-btn demo-btn-secondary" style="margin-top:12px" onclick="DemoModule.toggleBadge()">Toggle badge</button>
    </div>
  </div>

  <div class="demo-console" id="demo-console">
    <div class="demo-console-header">
      <span>Console</span>
      <button class="demo-btn demo-btn-secondary" style="padding:2px 8px;font-size:11px;" onclick="DemoModule.clearConsole()">Effacer</button>
    </div>
    <div class="demo-console-body" id="demo-console-body">
      <div class="demo-log info">Module initialisé ✓</div>
    </div>
  </div>
</div>`
}

// ── CSS de la vue ──────────────────────────────────────────────────

func buildCSS() string {
	return `
/* Styles scopés au module demo-ui */
.demo-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1a1a2e;
  overflow: hidden;
  font-family: 'Segoe UI', system-ui, sans-serif;
}

.demo-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px 24px;
  background: linear-gradient(135deg, #16213e 0%, #0f3460 100%);
  border-bottom: 1px solid #533483;
  flex-shrink: 0;
}

.demo-logo {
  font-size: 32px;
  color: #e94560;
  flex-shrink: 0;
}

.demo-header-text h1 {
  font-size: 18px;
  font-weight: 600;
  color: #e2e8f0;
  margin-bottom: 2px;
}
.demo-header-text p {
  font-size: 12px;
  color: #94a3b8;
}

.demo-header-actions {
  margin-left: auto;
  display: flex;
  gap: 8px;
}

.demo-btn {
  padding: 6px 14px;
  border-radius: 6px;
  border: none;
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  transition: all 0.15s;
}
.demo-btn-primary {
  background: #e94560;
  color: white;
}
.demo-btn-primary:hover { background: #c73652; transform: translateY(-1px); }
.demo-btn-secondary {
  background: rgba(255,255,255,0.08);
  color: #cbd5e1;
  border: 1px solid rgba(255,255,255,0.12);
}
.demo-btn-secondary:hover { background: rgba(255,255,255,0.14); }

.demo-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  padding: 20px 24px;
  flex: 1;
  overflow-y: auto;
}

.demo-card {
  background: rgba(255,255,255,0.04);
  border: 1px solid rgba(255,255,255,0.08);
  border-radius: 10px;
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  transition: all 0.2s;
}
.demo-card:hover {
  background: rgba(255,255,255,0.07);
  border-color: rgba(233,69,96,0.3);
  transform: translateY(-2px);
}

.demo-card-icon { font-size: 24px; }
.demo-card h3 { font-size: 14px; font-weight: 600; color: #e2e8f0; }
.demo-card p { font-size: 12px; color: #94a3b8; line-height: 1.5; flex: 1; }
.demo-card code {
  font-size: 11px;
  background: rgba(0,0,0,0.3);
  color: #7dd3fc;
  padding: 3px 8px;
  border-radius: 4px;
  font-family: 'Fira Code', monospace;
  align-self: flex-start;
}

.demo-console {
  flex-shrink: 0;
  height: 140px;
  border-top: 1px solid rgba(255,255,255,0.08);
  display: flex;
  flex-direction: column;
}
.demo-console-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 16px;
  background: rgba(0,0,0,0.2);
  font-size: 11px;
  color: #64748b;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  flex-shrink: 0;
}
.demo-console-body {
  flex: 1;
  overflow-y: auto;
  padding: 6px 16px;
  font-family: 'Fira Code', monospace;
  font-size: 12px;
}
.demo-log { padding: 2px 0; }
.demo-log.info { color: #7dd3fc; }
.demo-log.success { color: #4ade80; }
.demo-log.error { color: #f87171; }
.demo-log.warn { color: #fbbf24; }
.demo-log::before { margin-right: 6px; opacity: 0.5; }
.demo-log.info::before { content: 'ℹ'; }
.demo-log.success::before { content: '✓'; }
.demo-log.error::before { content: '✕'; }
.demo-log.warn::before { content: '⚠'; }`
}

// ── JS de la vue ───────────────────────────────────────────────────

func buildJS() string {
	return `
// JS scopé — moduleId est disponible depuis l'IIFE wrapper
window.DemoModule = {
  _badgeState: true,

  log: function(msg, type) {
    var body = document.getElementById('demo-console-body');
    if (!body) return;
    var el = document.createElement('div');
    el.className = 'demo-log ' + (type || 'info');
    el.textContent = new Date().toLocaleTimeString() + ' — ' + msg;
    body.appendChild(el);
    body.scrollTop = body.scrollHeight;
  },

  clearConsole: function() {
    var body = document.getElementById('demo-console-body');
    if (body) body.innerHTML = '';
  },

  refresh: function() {
    this.log('Refresh demandé depuis la vue module', 'success');
  },

  showOverlay: function() {
    // Enregistrer une vue overlay dynamiquement via AxiomModuleSystem
    window.AxiomModules.registerModule({
      moduleId: 'demo-overlay',
      title: 'Overlay Demo',
      viewType: 'overlay',
      showInActivityBar: false,
      closeable: true,
      autoActivate: true,
      html: ` + "`" + `
        <div style="display:flex;flex-direction:column;align-items:center;justify-content:center;height:100%;gap:24px;color:#e2e8f0;font-family:sans-serif;">
          <div style="font-size:48px;">🌫</div>
          <h2 style="font-size:24px;font-weight:300;">Overlay Modal</h2>
          <p style="color:#94a3b8;max-width:400px;text-align:center;">
            Cette vue flotte par-dessus toute l'application. Fermez-la avec le bouton ✕ ou la touche Escape.
          </p>
          <button onclick="window.AxiomModules.closeView('demo-overlay')"
                  style="padding:10px 24px;background:#e94560;color:white;border:none;border-radius:8px;cursor:pointer;font-size:14px;">
            Fermer l'overlay
          </button>
        </div>` + "`" + `,
    });
    window.DemoModule.log('Overlay ouvert', 'info');
  },

  testBranding: function() {
    // Modifier le branding via EventsEmit → axiom:input → engine
    // (nécessite L3 côté Go — ici on émet juste un event custom pour démo)
    if (window.axiomEmit) {
      window.axiomEmit('branding-request', { appName: 'My Custom IDE' });
    }
    this.log('Branding request envoyé (nécessite L3 côté Go)', 'warn');
  },

  toggleBadge: function() {
    this._badgeState = !this._badgeState;
    window.AxiomModules.setIconBadge('demo-ui', this._badgeState ? '!' : null);
    this.log('Badge ' + (this._badgeState ? 'activé' : 'retiré'), 'info');
  },
};

// Écouter les événements du bus Axiom depuis la vue module
document.addEventListener('axiom:bus-event', function(e) {
  var ev = e.detail;
  if (ev && ev.source === 'demo-ui') return; // ignorer nos propres events
  if (window.DemoModule) {
    window.DemoModule.log('Bus event reçu: ' + ev.topic, 'info');
  }
});

// Écouter les clics sur l'icône du module
document.addEventListener('axiom:module-icon-click', function(e) {
  if (e.detail && e.detail.moduleId === moduleId) {
    if (window.DemoModule) window.DemoModule.log('Icône cliquée', 'success');
  }
});

console.log('[DemoModule] JS initialisé, moduleId=' + moduleId);
`
}