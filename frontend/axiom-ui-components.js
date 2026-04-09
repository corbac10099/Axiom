/* ═══════════════════════════════════════════════════════════════
   AXIOM UI COMPONENTS LIBRARY — JavaScript
   
   Rend les composants interactifs et réutilisables
   ═══════════════════════════════════════════════════════════════ */

/**
 * AxiomComponents — Classe de gestion des composants
 */
class AxiomComponents {
    constructor() {
        this.components = new Map();
        this.init();
    }

    init() {
        console.log('[Axiom] Initializing UI Components Library');
        this.initTabs();
        this.initPanels();
        this.initButtons();
        this.initAlerts();
        this.initSidebar();
    }

    // ─────────────────────────────────────────────────────────────────
    // TABS
    // ─────────────────────────────────────────────────────────────────

    initTabs() {
        document.querySelectorAll('axiom-tabs').forEach(tabsElement => {
            const tabs = tabsElement.querySelectorAll('.axiom-tab');
            const panels = tabsElement.querySelectorAll('.axiom-tab-panel');

            tabs.forEach((tab, index) => {
                tab.addEventListener('click', () => {
                    // Désactiver tous les onglets
                    tabs.forEach(t => t.classList.remove('active'));
                    panels.forEach(p => p.classList.remove('active'));

                    // Activer le tab cliqué
                    tab.classList.add('active');
                    if (panels[index]) {
                        panels[index].classList.add('active');
                    }

                    // Emit event
                    this.emit('tab:changed', {
                        tabIndex: index,
                        tabLabel: tab.textContent
                    });
                });
            });

            // Activer le premier tab par défaut
            if (tabs.length > 0) {
                tabs[0].classList.add('active');
                if (panels[0]) panels[0].classList.add('active');
            }
        });
    }

    // ─────────────────────────────────────────────────────────────────
    // PANELS
    // ─────────────────────────────────────────────────────────────────

    initPanels() {
        document.querySelectorAll('axiom-panel[collapsible]').forEach(panel => {
            const header = panel.querySelector('.axiom-panel-header');
            const body = panel.querySelector('.axiom-panel-body');

            if (header && body) {
                header.addEventListener('click', () => {
                    body.classList.toggle('axiom-hidden');
                    panel.classList.toggle('collapsed');

                    this.emit('panel:toggled', {
                        panelId: panel.id,
                        collapsed: panel.classList.contains('collapsed')
                    });
                });
            }
        });
    }

    // ─────────────────────────────────────────────────────────────────
    // BUTTONS
    // ─────────────────────────────────────────────────────────────────

    initButtons() {
        document.querySelectorAll('axiom-button, .axiom-button').forEach(btn => {
            btn.addEventListener('click', (e) => {
                if (btn.disabled) {
                    e.preventDefault();
                    return;
                }

                const action = btn.getAttribute('data-action');
                if (action) {
                    this.emit('button:clicked', {
                        action,
                        label: btn.textContent
                    });
                }
            });
        });
    }

    // ─────────────────────────────────────────────────────────────────
    // ALERTS
    // ─────────────────────────────────────────────────────────────────

    initAlerts() {
        document.querySelectorAll('axiom-alert').forEach(alert => {
            const closeBtn = alert.querySelector('.axiom-alert-close');
            if (closeBtn) {
                closeBtn.addEventListener('click', () => {
                    alert.classList.add('axiom-animate-in');
                    setTimeout(() => alert.remove(), 300);
                });
            }
        });
    }

    // ─────────────────────────────────────────────────────────────────
    // SIDEBAR
    // ─────────────────────────────────────────────────────────────────

    initSidebar() {
        document.querySelectorAll('axiom-sidebar .axiom-sidebar-item').forEach(item => {
            item.addEventListener('click', () => {
                // Désactiver autres items
                document.querySelectorAll('axiom-sidebar .axiom-sidebar-item').forEach(i => {
                    i.classList.remove('active');
                });

                // Activer celui-ci
                item.classList.add('active');

                const action = item.getAttribute('data-action');
                this.emit('sidebar:itemClicked', {
                    action,
                    label: item.textContent
                });
            });
        });
    }

    // ─────────────────────────────────────────────────────────────────
    // CRÉATION DYNAMIQUE DE COMPOSANTS
    // ─────────────────────────────────────────────────────────────────

    /**
     * Crée un panel dynamiquement
     * @param {object} config - {id, title, content, closable, collapsible}
     */
    createPanel(config = {}) {
        const {
            id = 'panel-' + Date.now(),
            title = 'Panel',
            content = '',
            closable = true,
            collapsible = false,
            footer = ''
        } = config;

        const panel = document.createElement('axiom-panel');
        panel.id = id;
        if (collapsible) panel.setAttribute('collapsible', '');

        panel.innerHTML = `
            <div class="axiom-panel-header">
                <span>${title}</span>
                ${closable ? '<span style="cursor: pointer; opacity: 0.7;">✕</span>' : ''}
            </div>
            <div class="axiom-panel-body">
                ${content}
            </div>
            ${footer ? `<div class="axiom-panel-footer">${footer}</div>` : ''}
        `;

        // Close handler
        if (closable) {
            const closeBtn = panel.querySelector('.axiom-panel-header span:last-child');
            if (closeBtn) {
                closeBtn.addEventListener('click', () => panel.remove());
            }
        }

        // Re-init si déjà dans le DOM
        if (collapsible) {
            const header = panel.querySelector('.axiom-panel-header');
            const body = panel.querySelector('.axiom-panel-body');
            header?.addEventListener('click', () => {
                body.classList.toggle('axiom-hidden');
                panel.classList.toggle('collapsed');
            });
        }

        return panel;
    }

    /**
     * Crée un button dynamiquement
     */
    createButton(config = {}) {
        const {
            label = 'Button',
            action = '',
            variant = 'primary',
            size = 'md',
            disabled = false
        } = config;

        const btn = document.createElement('axiom-button');
        btn.textContent = label;
        btn.className = `${variant} ${size}`;
        btn.setAttribute('data-action', action);
        if (disabled) btn.disabled = true;

        btn.addEventListener('click', () => {
            if (!disabled && action) {
                this.emit('button:clicked', { action, label });
            }
        });

        return btn;
    }

    /**
     * Crée un alert
     */
    createAlert(config = {}) {
        const {
            message = 'Alert',
            type = 'info',
            closable = true,
            duration = 0
        } = config;

        const alert = document.createElement('axiom-alert');
        alert.className = type;
        alert.innerHTML = `
            <span>${message}</span>
            ${closable ? '<span class="axiom-alert-close">✕</span>' : ''}
        `;

        if (closable) {
            const closeBtn = alert.querySelector('.axiom-alert-close');
            if (closeBtn) {
                closeBtn.addEventListener('click', () => alert.remove());
            }
        }

        if (duration > 0) {
            setTimeout(() => alert.remove(), duration);
        }

        return alert;
    }

    /**
     * Crée des tabs
     */
    createTabs(config = {}) {
        const {
            tabs = [{ label: 'Tab 1', content: 'Content 1' }]
        } = config;

        const tabsElement = document.createElement('axiom-tabs');
        const tabsList = document.createElement('div');
        tabsList.className = 'axiom-tabs-list';

        const content = document.createElement('div');
        content.className = 'axiom-tabs-content';

        tabs.forEach((tab, index) => {
            const tabBtn = document.createElement('button');
            tabBtn.className = `axiom-tab ${index === 0 ? 'active' : ''}`;
            tabBtn.textContent = tab.label;

            const panel = document.createElement('div');
            panel.className = `axiom-tab-panel ${index === 0 ? 'active' : ''}`;
            panel.innerHTML = tab.content;

            tabBtn.addEventListener('click', () => {
                tabsList.querySelectorAll('.axiom-tab').forEach(t => t.classList.remove('active'));
                content.querySelectorAll('.axiom-tab-panel').forEach(p => p.classList.remove('active'));

                tabBtn.classList.add('active');
                panel.classList.add('active');
            });

            tabsList.appendChild(tabBtn);
            content.appendChild(panel);
        });

        tabsElement.appendChild(tabsList);
        tabsElement.appendChild(content);

        return tabsElement;
    }

    // ─────────────────────────────────────────────────────────────────
    // UTILITY
    // ─────────────────────────────────────────────────────────────────

    emit(eventName, data = {}) {
        const event = new CustomEvent(eventName, { detail: data });
        document.dispatchEvent(event);
    }

    on(eventName, callback) {
        document.addEventListener(eventName, (e) => {
            callback(e.detail);
        });
    }

    /**
     * Change le thème
     */
    setTheme(themeName) {
        document.documentElement.className = `theme-${themeName}`;
        localStorage.setItem('axiom-theme', themeName);
        this.emit('theme:changed', { theme: themeName });
    }

    /**
     * Retourne le thème actuel
     */
    getTheme() {
        const current = document.documentElement.className.replace('theme-', '');
        return current || 'dark';
    }

    /**
     * Show a loading spinner
     */
    showLoading(message = 'Loading...') {
        const spinner = document.createElement('div');
        spinner.className = 'axiom-animate-spin';
        spinner.style.cssText = `
            display: inline-block;
            width: 20px;
            height: 20px;
            border: 3px solid var(--axiom-border);
            border-top-color: var(--axiom-accent);
            border-radius: 50%;
        `;

        return spinner;
    }

    /**
     * Affiche une modale simple
     */
    modal(config = {}) {
        const {
            title = 'Modal',
            content = '',
            buttons = [{ label: 'Close', action: 'close' }]
        } = config;

        const overlay = document.createElement('div');
        overlay.style.cssText = `
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(0, 0, 0, 0.5);
            display: flex;
            align-items: center;
            justify-content: center;
            z-index: 1000;
        `;

        const modalPanel = this.createPanel({
            title,
            content,
            closable: true
        });

        modalPanel.style.cssText = `
            max-width: 500px;
            max-height: 80vh;
            z-index: 1001;
        `;

        overlay.appendChild(modalPanel);
        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) {
                overlay.remove();
            }
        });

        document.body.appendChild(overlay);

        return overlay;
    }
}

// ─────────────────────────────────────────────────────────────────
// INITIALISATION GLOBALE
// ─────────────────────────────────────────────────────────────────

// Exposer globalement
window.Axiom = window.Axiom || {};
window.Axiom.UI = new AxiomComponents();

// Initialiser au chargement du DOM
document.addEventListener('DOMContentLoaded', () => {
    console.log('[Axiom] UI Components ready');
});

// ─────────────────────────────────────────────────────────────────
// HELPERS GLOBAUX
// ─────────────────────────────────────────────────────────────────

/**
 * Fonctions globales pour utilisation facile
 */

window.axiomPanel = (config) => window.Axiom.UI.createPanel(config);
window.axiomButton = (config) => window.Axiom.UI.createButton(config);
window.axiomAlert = (config) => window.Axiom.UI.createAlert(config);
window.axiomTabs = (config) => window.Axiom.UI.createTabs(config);
window.axiomModal = (config) => window.Axiom.UI.modal(config);
window.axiomSetTheme = (theme) => window.Axiom.UI.setTheme(theme);

/**
 * Raccourci pour émettre un événement personnalisé
 */
window.axiomEmit = (event, data) => window.Axiom.UI.emit(event, data);

/**
 * Écouter les événements Axiom
 */
window.axiomOn = (event, callback) => window.Axiom.UI.on(event, callback);

// ═══════════════════════════════════════════════════════════════
// EXEMPLE D'UTILISATION
// ═══════════════════════════════════════════════════════════════

/*
// Créer un panel
const myPanel = axiomPanel({
    id: 'my-panel',
    title: 'My Panel',
    content: '<p>Hello World!</p>',
    collapsible: true,
    footer: '<axiom-button data-action="save">Save</axiom-button>'
});
document.getElementById('container').appendChild(myPanel);

// Créer un alert
const alert = axiomAlert({
    message: 'Successfully saved!',
    type: 'success',
    duration: 3000
});
document.body.appendChild(alert);

// Écouter les événements
axiomOn('button:clicked', (data) => {
    console.log('Button clicked:', data.action);
});

// Changer le thème
axiomSetTheme('light');
*/

// ═══════════════════════════════════════════════════════════════