import './style.css';
import './app.css';
import {
    ScreenGetAll,
    WindowGetPosition,
    WindowGetSize,
    WindowIsMaximised,
    WindowMaximise,
    WindowUnmaximise,
    WindowSetPosition,
    WindowSetSize,
} from '../wailsjs/runtime/runtime';
import { GetWindowStyle, GetAppTitle, TerminalSetFocus } from '../wailsjs/go/main/WApp';
import { EventsOn } from '../wailsjs/runtime/runtime';

// Global terminal focus state for canvas dimming
window.terminalFocusedState = true;

// Remove any body margin/padding immediately so there is no layout flash.
document.body.style.margin = '0';
document.body.style.padding = '0';
document.body.style.overflow = 'hidden';

const app = document.getElementById('app') || document.body;
const INACTIVE_PANE_OVERLAY_ALPHA = 51 / 255;

// Setup titlebar shell synchronously to avoid startup race conditions.
function setupTitlebar() {
    const titlebar = document.createElement('div');
    titlebar.id = 'custom-titlebar';
    titlebar.style.cssText = [
        'width:100%',
        'height:32px',
        'display:flex',
        'align-items:center',
        'justify-content:center',
        'background:rgba(30,30,30,1)',
        'border-bottom:1px solid rgba(0,0,0,0.5)',
        'user-select:none',
        '-webkit-user-select:none',
        'cursor:default',
        '-webkit-app-region:drag',
        'flex-shrink:0',
        'font-family:system-ui, -apple-system, sans-serif',
        'font-size:13px',
        'font-weight:500',
        'color:rgba(255,255,255,0.87)',
        'letter-spacing:0.3px',
        '--wails-draggable:drag',
    ].join(';');
    titlebar.textContent = 'loading...';

    titlebar.addEventListener('dblclick', () => {
        void maximizeWindowFromTitlebar();
    });
    
    return titlebar;
}

async function maximizeWindowFromTitlebar() {
    try {
        const isMaximised = await WindowIsMaximised();
        if (isMaximised) {
            WindowUnmaximise();
            return;
        }

        WindowMaximise();
    } catch {
        // Ignore runtime errors from window manager integration.
    }
}

async function hydrateTitlebarAndBorders() {
    let appName = 'TTYphoon';
    /*let bgColor = 'rgba(30,30,30,1)';
    let fgColor = 'rgba(255,255,255,0.87)';
    let borderColor = 'rgba(0,0,0,0.2)';*/

    try {
        appName = await GetAppTitle();
    } catch (err) {
        console.warn('Failed to fetch app name:', err);
    }

    if (titlebar) {
        titlebar.textContent = appName;
    }

    try {
        const style = await GetWindowStyle();
        applyChromePalette(style);
    } catch (err) {
        console.warn('Failed to fetch window style:', err);
    }
}

function applyChromePalette(style) {
    let bgColor = 'rgba(30,30,30,1)';
    let fgColor = 'rgba(255,255,255,0.87)';
    let borderColor = 'rgba(0,0,0,0.2)';
    const showStatusBar = style?.statusBar !== false;
    const statusFontSize = Math.max(8, Number(style?.fontSize || 14) - 2);

    if (style?.colors?.bg) {
        const bg = style.colors.bg;
        bgColor = `rgb(${bg.Red}, ${bg.Green}, ${bg.Blue})`;
        borderColor = `rgba(${bg.Red}, ${bg.Green}, ${bg.Blue}, 0.2)`;
    }
    if (style?.colors?.fg) {
        fgColor = `rgb(${style.colors.fg.Red}, ${style.colors.fg.Green}, ${style.colors.fg.Blue})`;
    }

    if (titlebar) {
        titlebar.style.background = bgColor;
        titlebar.style.color = fgColor;
    }

    if (contentWrapper) {
        contentWrapper.style.borderLeft = `3px solid ${borderColor}`;
        contentWrapper.style.borderRight = `3px solid ${borderColor}`;
        contentWrapper.style.borderBottom = showStatusBar ? '0' : `3px solid ${borderColor}`;
    }

    if (statusBar) {
        statusBar.style.display = showStatusBar ? 'flex' : 'none';
        statusBar.style.fontSize = `${statusFontSize}px`;
        statusBar.style.background = `linear-gradient(rgba(0, 0, 0, ${INACTIVE_PANE_OVERLAY_ALPHA}), rgba(0, 0, 0, ${INACTIVE_PANE_OVERLAY_ALPHA})), ${bgColor}`;
        statusBar.style.color = fgColor;
        statusBar.style.borderLeft = `3px solid ${borderColor}`;
        statusBar.style.borderRight = `3px solid ${borderColor}`;
        statusBar.style.borderBottom = `3px solid ${borderColor}`;
            statusBar.style.borderTop = '1px solid rgba(255, 255, 255, 0.18)';
            statusBar.style.boxShadow = 'inset 0 1px 0 rgba(255, 255, 255, 0.12)';
    }

    if (notesStatusWrap) {
        notesStatusWrap.style.fontSize = `${statusFontSize}px`;
    }

    if (terminalStatus) {
        terminalStatus.style.fontSize = `${statusFontSize}px`;
    }
}

app.style.cssText = [
    'display:flex',
    'flex-direction:column',
    'width:100vw',
    'height:100vh',
    'margin:0',
    'padding:0',
    'overflow:hidden',
    'box-sizing:border-box',
].join(';');

// Initialize titlebar and continue setup
let titlebar;
let contentWrapper;
let notesPane;
let splitHandle;
let splitToggle;
let terminalPane;
let statusBar;
let notesStatusWrap;
let terminalStatus;

let isDraggingSplit = false;
let notesCollapsed = false;
let lastNotesWidthPercent = 50;
let lastCollapseDeltaPx = 0;
let terminalFocusState = true;

(async () => {
    titlebar = setupTitlebar();
    app.appendChild(titlebar);

// Content wrapper for borders and split layout
contentWrapper = document.createElement('div');
contentWrapper.id = 'content-wrapper';
contentWrapper.style.cssText = [
    'flex:1',
    'display:flex',
    'width:100%',
    'height:100%',
    'border-left:3px solid rgba(0,0,0,0.2)',
    'border-right:3px solid rgba(0,0,0,0.2)',
    'box-sizing:border-box',
    'overflow:hidden',
].join(';');

// The split layout: notes on the left half, terminal on the right half.
// Both panes are created synchronously here.  notes.js and terminal.js are
// loaded as dynamic imports below, so their module bodies run *after* this
// synchronous code — they will find #notes-pane and #terminal-pane in the DOM.
notesPane = document.createElement('div');
notesPane.id = 'notes-pane';
notesPane.style.cssText = [
    'width:50%',
    'height:100%',
    'overflow:hidden',
    'position:relative',
    // Subtle separator between the two halves.  The notes module injects CSS
    // variables for the theme colours; until those land we use a semi-transparent
    // white border that is unobtrusive on any background.
    'border-right:1px solid rgba(255,255,255,0.12)',
    'box-sizing:border-box',
    'flex-shrink:0',
].join(';');

splitHandle = document.createElement('div');
splitHandle.id = 'notes-terminal-split';
splitHandle.style.cssText = [
    'width:8px',
    'height:100%',
    'cursor:col-resize',
    'background:transparent',
    'position:relative',
    'flex-shrink:0',
    'user-select:none',
    'touch-action:none',
].join(';');

const splitHandleLine = document.createElement('div');
splitHandleLine.style.cssText = [
    'position:absolute',
    'left:50%',
    'top:0',
    'transform:translateX(-50%)',
    'width:1px',
    'height:100%',
    'background:rgba(255,255,255,0.16)',
].join(';');
splitHandle.appendChild(splitHandleLine);

splitToggle = document.createElement('button');
splitToggle.type = 'button';
splitToggle.id = 'notes-terminal-toggle';
splitToggle.setAttribute('aria-label', 'Collapse notes pane');
splitToggle.title = 'Collapse notes pane';
splitToggle.textContent = '▶';
splitToggle.style.cssText = [
    'position:absolute',
    'left:50%',
    'top:50%',
    'transform:translate(-50%, -50%)',
    'width:16px',
    'height:40px',
    'padding:0',
    'border:1px solid rgba(255,255,255,0.25)',
    'border-radius:8px',
    'background:rgba(0,0,0,0.35)',
    'color:rgba(255,255,255,0.8)',
    'font-size:11px',
    'line-height:1',
    'cursor:pointer',
    'z-index:2',
    '-webkit-app-region:no-drag',
].join(';');
splitHandle.appendChild(splitToggle);

terminalPane = document.createElement('div');
terminalPane.id = 'terminal-pane';
terminalPane.style.cssText = [
    'flex:1',
    'height:100%',
    'overflow:hidden',
    'position:relative',
    'min-width:0',
].join(';');

contentWrapper.appendChild(notesPane);
contentWrapper.appendChild(splitHandle);
contentWrapper.appendChild(terminalPane);

app.appendChild(contentWrapper);

statusBar = document.createElement('div');
statusBar.id = 'app-statusbar';
statusBar.style.cssText = [
    'height:24px',
    'display:flex',
    'align-items:center',
    'justify-content:space-between',
    'width:100%',
    `background:linear-gradient(rgba(0, 0, 0, ${INACTIVE_PANE_OVERLAY_ALPHA}), rgba(0, 0, 0, ${INACTIVE_PANE_OVERLAY_ALPHA})), rgba(30,30,30,1)`,
    'border-left:3px solid rgba(0,0,0,0.2)',
    'border-right:3px solid rgba(0,0,0,0.2)',
    'border-bottom:3px solid rgba(0,0,0,0.2)',
    'border-top:1px solid rgba(255,255,255,0.18)',
    'box-shadow:inset 0 1px 0 rgba(255,255,255,0.12)',
    'box-sizing:border-box',
    'overflow:hidden',
    'font-size:12px',
    'line-height:1',
    'padding:0',
    'padding-top:4px',
    '--wails-draggable:drag',
    'cursor:arrow'
].join(';');

notesStatusWrap = document.createElement('div');
notesStatusWrap.id = 'notes-status-wrap';
notesStatusWrap.style.cssText = [
    'width:50%',
    'height:100%',
    'display:flex',
    'align-items:center',
    'padding:0 10px',
    'min-width:0',
    'box-sizing:border-box',
].join(';');

const notesStatus = document.createElement('div');
notesStatus.id = 'notes-status';
notesStatus.setAttribute('role', 'status');
notesStatus.style.cssText = [
    'height:100%',
    'display:flex',
    'align-items:center',
    'width:100%',
    'white-space:nowrap',
    'overflow:hidden',
    'text-overflow:ellipsis',
    'opacity:0.85',
].join(';');
notesStatusWrap.appendChild(notesStatus);

terminalStatus = document.createElement('div');
terminalStatus.id = 'terminal-status';
terminalStatus.style.cssText = [
    'flex:1',
    'height:100%',
    'display:flex',
    'align-items:center',
    'line-height:1',
    'padding:0 10px',
    'min-width:0',
    'white-space:nowrap',
    'overflow:hidden',
    'text-overflow:ellipsis',
    'opacity:0.85',
    'box-sizing:border-box',
].join(';');

statusBar.appendChild(notesStatusWrap);
statusBar.appendChild(terminalStatus);
app.appendChild(statusBar);

// Setup event listeners after DOM elements are created
splitHandle.addEventListener('mousedown', (event) => {
    if (event.target === splitToggle) {
        return;
    }

    if (event.button !== 0) {
        return;
    }

    isDraggingSplit = true;
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
    event.preventDefault();
});

splitToggle.addEventListener('click', (event) => {
    event.preventDefault();
    event.stopPropagation();
    toggleNotesPaneCollapsed();
});

notesPane.addEventListener('focusin', () => {
    setTerminalFocusState(false);
});

notesPane.addEventListener('mousedown', () => {
    setTerminalFocusState(true);
});

terminalPane.addEventListener('focusin', () => {
    setTerminalFocusState(true);
});

terminalPane.addEventListener('mousedown', () => {
    setTerminalFocusState(true);
});

refreshStatusBarLayout();

    // Update titlebar text and colors asynchronously after shell render.
    void hydrateTitlebarAndBorders();
})();

function refreshStatusBarLayout() {
    if (!notesPane || !notesStatusWrap || !terminalStatus) {
        return;
    }

    const notesWidthPx = notesCollapsed
        ? 0
        : Math.max(0, Math.round(notesPane.getBoundingClientRect().width));
    const collapsed = notesWidthPx <= 1;

    notesStatusWrap.style.width = `${notesWidthPx}px`;
    notesStatusWrap.style.padding = collapsed ? '0' : '0 10px';

    // Keep terminal status text closer to the left edge when notes are collapsed.
    terminalStatus.style.paddingLeft = collapsed ? '4px' : '10px';
}

function clamp(value, min, max) {
    return Math.min(Math.max(value, min), max);
}

function getScreenBounds(screen) {
    const x = Number.isFinite(screen?.x) ? screen.x : 0;
    const y = Number.isFinite(screen?.y) ? screen.y : 0;
    const width = Number.isFinite(screen?.width) ? screen.width : window.screen.availWidth;
    const height = Number.isFinite(screen?.height) ? screen.height : window.screen.availHeight;
    return { x, y, width, height };
}

async function adjustWindowFrameBy(deltaPx) {
    if (!Number.isFinite(deltaPx) || deltaPx === 0) {
        return;
    }

    try {
        const [size, pos, screens] = await Promise.all([
            WindowGetSize(),
            WindowGetPosition(),
            ScreenGetAll().catch(() => [])
        ]);

        const width = Number.isFinite(size?.w) ? size.w : window.innerWidth;
        const height = Number.isFinite(size?.h) ? size.h : window.innerHeight;
        const x = Number.isFinite(pos?.x) ? pos.x : 0;
        const y = Number.isFinite(pos?.y) ? pos.y : 0;

        const targetWidth = Math.max(480, Math.round(width + deltaPx));
        const appliedDelta = targetWidth - width;
        let targetX = Math.round(x - appliedDelta);
        let targetY = y;

        const allScreens = Array.isArray(screens) ? screens : [];
        const currentScreen = allScreens.find((s) => s?.isCurrent) || allScreens.find((s) => s?.isPrimary) || allScreens[0];
        const bounds = getScreenBounds(currentScreen);

        const minX = bounds.x;
        const maxX = bounds.x + Math.max(0, bounds.width-targetWidth);
        targetX = clamp(targetX, minX, maxX);

        const minY = bounds.y;
        const maxY = bounds.y + Math.max(0, bounds.height-height);
        targetY = clamp(targetY, minY, maxY);

        if (targetWidth !== width) {
            WindowSetSize(targetWidth, height);
        }

        if (targetX !== x || targetY !== y) {
            WindowSetPosition(targetX, targetY);
        }
    } catch {
        // Ignore runtime errors; pane collapse/expand still works without frame resize.
    }
}

function setSplitFromClientX(clientX) {
    const rect = app.getBoundingClientRect();
    if (rect.width <= 0) {
        return;
    }

    const minPanePx = Math.max(240, Math.round(rect.width * 0.15));
    const maxNotesPx = rect.width - minPanePx - splitHandle.offsetWidth;
    const rawNotesPx = clientX - rect.left;
    const notesPx = Math.min(Math.max(rawNotesPx, minPanePx), maxNotesPx);
    const notesPercent = (notesPx / rect.width) * 100;

    notesCollapsed = false;
    lastNotesWidthPercent = notesPercent;
    notesPane.style.width = `${notesPercent}%`;
    notesPane.style.borderRight = '1px solid rgba(255,255,255,0.12)';
    splitToggle.textContent = '▶';
    splitToggle.setAttribute('aria-label', 'Collapse notes pane');
    splitToggle.title = 'Collapse notes pane';

    refreshStatusBarLayout();

    // Terminal renderer listens for window resize to recompute canvas/rows.
    //window.dispatchEvent(new Event('resize'));
}

function toggleNotesPaneCollapsed() {
    const rect = app.getBoundingClientRect();
    if (rect.width <= 0) {
        return;
    }

    if (!notesCollapsed) {
        const currentPx = notesPane.getBoundingClientRect().width;
        if (currentPx > 0) {
            lastNotesWidthPercent = (currentPx / rect.width) * 100;
            lastCollapseDeltaPx = Math.round(currentPx);
        }

        notesCollapsed = true;
        notesPane.style.width = '0';
        notesPane.style.borderRight = '0';
        splitToggle.textContent = '◀';
        splitToggle.setAttribute('aria-label', 'Expand notes pane');
        splitToggle.title = 'Expand notes pane';

        void adjustWindowFrameBy(-lastCollapseDeltaPx).finally(() => {
            requestAnimationFrame(() => {
                refreshStatusBarLayout();
            });
        });
    } else {
        const minPercent = (Math.max(240, Math.round(rect.width * 0.15)) / rect.width) * 100;
        const maxPercent = ((rect.width - Math.max(240, Math.round(rect.width * 0.15)) - splitHandle.offsetWidth) / rect.width) * 100;
        const restored = Math.min(Math.max(lastNotesWidthPercent, minPercent), maxPercent);

        // Reuse divider logic so pane and status layout stay in sync.
        const restoredPx = (restored / 100) * rect.width;
        setSplitFromClientX(rect.left + restoredPx);

        if (lastCollapseDeltaPx > 0) {
            void adjustWindowFrameBy(lastCollapseDeltaPx).finally(() => {
                requestAnimationFrame(() => {
                    refreshStatusBarLayout();
                });
            });
        }
    }

    refreshStatusBarLayout();
    requestAnimationFrame(() => {
        refreshStatusBarLayout();
    });

    requestTerminalResizeAfterLayout();
}

function requestTerminalResizeAfterLayout() {
    // Wait for layout to settle before notifying terminal.js to recompute rows/cols.
    requestAnimationFrame(() => {
        window.dispatchEvent(new Event('resize'));
    });
}

function setTerminalFocusState(focused) {
    if (terminalFocusState === focused) {
        return;
    }

    terminalFocusState = focused;
    window.terminalFocusedState = focused;
    TerminalSetFocus(focused).catch(() => {});
}

window.addEventListener('mousemove', (event) => {
    if (!isDraggingSplit) {
        return;
    }

    setSplitFromClientX(event.clientX);
});

window.addEventListener('mouseup', () => {
    if (!isDraggingSplit) {
        return;
    }

    isDraggingSplit = false;
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
    requestTerminalResizeAfterLayout();
});

EventsOn('toggleNotesPane', () => {
    toggleNotesPaneCollapsed();
});

window.addEventListener('resize', () => {
    refreshStatusBarLayout();
});

// Dynamic imports — the promises resolve asynchronously, but the resolution
// microtask queue starts only after this synchronous module body finishes.
// By then #notes-pane and #terminal-pane exist, so each module finds its root.
Promise.all([
    import('./notes.js'),
    import('./terminal.js')
]).then(() => {
    // After all modules have loaded, trigger a resize event to ensure
    // proper sizing of all components (file list, terminal tabs, tmux, etc.)
    requestAnimationFrame(() => {
        window.dispatchEvent(new Event('resize'));
    });
}).catch((err) => {
    console.error('Failed to load modules:', err);
});

EventsOn('terminalStyleUpdate', payload => {
    const result = Array.isArray(payload?.[0]) ? payload[0] : payload;
    if (result && result.colors) {
        applyChromePalette(result);
    }
});
