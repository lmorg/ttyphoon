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
        if (style?.fontFamily) {
            statusBar.style.fontFamily = style.fontFamily;
        }
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
let terminalPane;
let statusBar;
let notesStatusWrap;
let terminalStatus;
let terminalJupyterHost;
let notesOriginalParent;
let notesOriginalNextSibling;
let notesOriginalStyle;

let isDraggingSplit = false;
let notesCollapsed = false;
let lastNotesWidthPercent = 50;
let terminalFocusState = true;
let terminalKeyboardFocusVisible = false;
let lastInputWasKeyboard = false;
const MIN_NOTES_PX = 240;
const MIN_NOTES_EMBED_PX = 96;

function updateTerminalFocusChrome() {
    if (!terminalPane) {
        return;
    }

    terminalPane.setAttribute('data-terminal-focused', terminalFocusState ? 'true' : 'false');
    terminalPane.setAttribute('data-terminal-focus-visible', terminalKeyboardFocusVisible ? 'true' : 'false');
}

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
splitHandle.title = 'Drag to resize notes';

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
    if (event.button !== 0) {
        return;
    }

    const embeddedInTerminal = notesPane?.parentElement?.id === 'terminal-jupyter-host';
    if (notesCollapsed || embeddedInTerminal) {
        return;
    }

    isDraggingSplit = true;
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
    event.preventDefault();
});

splitHandle.addEventListener('dblclick', (event) => {
    event.preventDefault();
    event.stopPropagation();
    void toggleNotesPaneCollapsed();
});

function shouldKeepTerminalFocusedForNotesTarget(target) {
    const mode = notesPane.dataset.viewMode;
    const targetEl = target instanceof Element ? target : null;

    if (mode === 'editor') {
        return false;
    }

    if (mode === 'viewer') {
        const editingJsonViewer = Boolean(targetEl && targetEl.closest('.json-inline-editor'));
        return !editingJsonViewer;
    }

    // In jupyter mode, code blocks are editable and should keep keyboard ownership in notes.
    const inJupyterCodeBlock = Boolean(targetEl && targetEl.closest('.jupyter-code-block'));
    return !inJupyterCodeBlock;
}

notesPane.addEventListener('focusin', (event) => {
    // Mode-aware focus bridge: viewer -> terminal, editor -> notes, jupyter -> depends on target.
    setTerminalFocusState(shouldKeepTerminalFocusedForNotesTarget(event.target));
});

notesPane.addEventListener('mousedown', (event) => {
    // Handle repeat clicks where focusin might not fire again.
    setTerminalFocusState(shouldKeepTerminalFocusedForNotesTarget(event.target));
});

terminalPane.addEventListener('focusin', () => {
    setTerminalFocusState(true, { focusVisible: lastInputWasKeyboard });
});

terminalPane.addEventListener('mousedown', () => {
    setTerminalFocusState(true, { focusVisible: false });
});

refreshStatusBarLayout();
updateSplitHandleTooltip();
updateTerminalFocusChrome();

    // Update titlebar text and colors asynchronously after shell render.
    void hydrateTitlebarAndBorders();
})();

function updateSplitHandleTooltip() {
    if (!splitHandle) {
        return;
    }

    const embeddedInTerminal = notesPane?.parentElement?.id === 'terminal-jupyter-host';
    splitHandle.title = (notesCollapsed || embeddedInTerminal)
        ? 'Double-click to expand notes'
        : 'Drag to resize notes. Double-click to collapse notes';
}

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

function ensureTerminalJupyterHost() {
    if (terminalJupyterHost) {
        return terminalJupyterHost;
    }

    const terminalViewport = document.getElementById('terminal-viewport');
    if (!terminalViewport) {
        return null;
    }

    terminalJupyterHost = document.createElement('div');
    terminalJupyterHost.id = 'terminal-jupyter-host';
    terminalJupyterHost.style.cssText = [
        'display:none',
        'position:absolute',
        'inset:0',
        'min-height:0',
        'overflow:auto',
        'z-index:2',
    ].join(';');
    terminalViewport.appendChild(terminalJupyterHost);
    return terminalJupyterHost;
}

function getCurrentNoteFileName() {
    const fileName = notesPane?.dataset?.currentFileName;
    return fileName && fileName.length > 0 ? fileName : 'Notes';
}

function setTerminalJupyterMode(enabled) {
    const host = ensureTerminalJupyterHost();
    if (!notesPane) {
        return;
    }

    if (enabled) {
        if (!host) {
            return;
        }

        if (!notesOriginalParent) {
            notesOriginalParent = notesPane.parentElement;
            notesOriginalNextSibling = notesPane.nextElementSibling;
            notesOriginalStyle = {
                width: notesPane.style.width,
                height: notesPane.style.height,
                borderRight: notesPane.style.borderRight,
                overflow: notesPane.style.overflow,
                position: notesPane.style.position,
                flexShrink: notesPane.style.flexShrink,
            };
        }

        if (notesPane.parentElement !== host) {
            host.appendChild(notesPane);
        }

        notesPane.style.width = '100%';
        notesPane.style.height = '100%';
        notesPane.style.borderRight = '0';
        notesPane.style.overflow = 'hidden';
        notesPane.style.position = 'relative';
        notesPane.style.flexShrink = '1';
        host.style.display = 'none';
        window.dispatchEvent(new CustomEvent('ttyphoon-jupyter-tab-mode', {
            detail: {
                enabled: true,
                active: true,
                title: getCurrentNoteFileName(),
            }
        }));
        return;
    }

    if (host) {
        host.style.display = 'none';
    }

    if (notesOriginalParent && notesPane.parentElement !== notesOriginalParent) {
        if (notesOriginalNextSibling && notesOriginalNextSibling.parentElement === notesOriginalParent) {
            notesOriginalParent.insertBefore(notesPane, notesOriginalNextSibling);
        } else {
            notesOriginalParent.appendChild(notesPane);
        }
    } else if (contentWrapper && splitHandle && notesPane.parentElement !== contentWrapper) {
        // Fallback: always dock notes back to the left of terminal split.
        contentWrapper.insertBefore(notesPane, splitHandle);
    }

    if (notesOriginalStyle) {
        notesPane.style.width = notesOriginalStyle.width;
        notesPane.style.height = notesOriginalStyle.height;
        notesPane.style.borderRight = notesOriginalStyle.borderRight;
        notesPane.style.overflow = notesOriginalStyle.overflow;
        notesPane.style.position = notesOriginalStyle.position;
        notesPane.style.flexShrink = notesOriginalStyle.flexShrink;
    }

    window.dispatchEvent(new CustomEvent('ttyphoon-jupyter-tab-mode', {
        detail: {
            enabled: false,
            active: false,
            title: getCurrentNoteFileName(),
        }
    }));
}

function setSplitFromClientX(clientX) {
    const rect = app.getBoundingClientRect();
    if (rect.width <= 0) {
        return;
    }

    const minPanePx = Math.max(MIN_NOTES_PX, Math.round(rect.width * 0.15));
    const maxNotesPx = rect.width - minPanePx - splitHandle.offsetWidth;
    const rawNotesPx = clientX - rect.left;
    const notesPx = Math.min(Math.max(rawNotesPx, minPanePx), maxNotesPx);

    if (rawNotesPx <= MIN_NOTES_EMBED_PX) {
        collapseNotesIntoTerminal();
        return;
    }

    const notesPercent = (notesPx / rect.width) * 100;

    // Ensure notes pane is docked back into the split layout before sizing it.
    setTerminalJupyterMode(false);

    notesCollapsed = false;
    lastNotesWidthPercent = notesPercent;
    notesPane.style.width = `${notesPercent}%`;
    notesPane.style.borderRight = '1px solid rgba(255,255,255,0.12)';
    updateSplitHandleTooltip();

    refreshStatusBarLayout();

    // Terminal renderer listens for window resize to recompute canvas/rows.
    //window.dispatchEvent(new Event('resize'));
}

function collapseNotesIntoTerminal() {
    const rect = app.getBoundingClientRect();
    if (rect.width > 0) {
        const currentPx = notesPane.getBoundingClientRect().width;
        if (currentPx > 0) {
            lastNotesWidthPercent = (currentPx / rect.width) * 100;
        }
    }

    // Move notes into terminal host first so original split styles are preserved.
    setTerminalJupyterMode(true);
    notesCollapsed = true;
    updateSplitHandleTooltip();
    refreshStatusBarLayout();
}

function getScreenBounds(screen) {
    const x = Number.isFinite(screen?.x) ? screen.x : 0;
    const y = Number.isFinite(screen?.y) ? screen.y : 0;
    const width = Number.isFinite(screen?.width) ? screen.width : window.screen.availWidth;
    const height = Number.isFinite(screen?.height) ? screen.height : window.screen.availHeight;
    return { x, y, width, height };
}

function clampToScreenX(x, width, bounds) {
    const minX = bounds.x;
    const maxX = bounds.x + Math.max(0, bounds.width - width);
    return Math.min(Math.max(x, minX), maxX);
}

async function getCurrentWindowAndScreen() {
    const [size, pos, screens] = await Promise.all([
        WindowGetSize(),
        WindowGetPosition(),
        ScreenGetAll().catch(() => []),
    ]);

    const width = Number.isFinite(size?.w) ? size.w : window.innerWidth;
    const height = Number.isFinite(size?.h) ? size.h : window.innerHeight;
    const x = Number.isFinite(pos?.x) ? pos.x : window.screenX;
    const y = Number.isFinite(pos?.y) ? pos.y : window.screenY;

    const allScreens = Array.isArray(screens) ? screens : [];
    const currentScreen = allScreens.find((s) => s?.isCurrent) || allScreens.find((s) => s?.isPrimary) || allScreens[0];
    const bounds = getScreenBounds(currentScreen);
    return { width, height, x, y, bounds };
}

async function toggleNotesPaneCollapsed() {
    const rect = app.getBoundingClientRect();
    if (rect.width <= 0) {
        return;
    }

    const embeddedInTerminal = notesPane?.parentElement?.id === 'terminal-jupyter-host';

    if (!notesCollapsed && !embeddedInTerminal) {
        const currentPx = notesPane.getBoundingClientRect().width;
        if (currentPx > 0) {
            lastNotesWidthPercent = (currentPx / rect.width) * 100;
        }

        const frame = await getCurrentWindowAndScreen();
        const collapsedWidth = Math.max(480, Math.round(frame.width / 2));
        const rightEdge = frame.x + frame.width;
        const collapsedX = clampToScreenX(rightEdge - collapsedWidth, collapsedWidth, frame.bounds);

        collapseNotesIntoTerminal();
        WindowSetSize(collapsedWidth, frame.height);
        WindowSetPosition(collapsedX, frame.y);
        requestAnimationFrame(() => {
            refreshStatusBarLayout();
        });
    } else {
        const frame = await getCurrentWindowAndScreen();
        const expandedWidth = Math.min(frame.bounds.width, Math.max(480, Math.round(frame.width * 2)));
        const expandedX = clampToScreenX(frame.x, expandedWidth, frame.bounds);

        setTerminalJupyterMode(false);
        notesCollapsed = false;
        updateSplitHandleTooltip();
        WindowSetSize(expandedWidth, frame.height);
        WindowSetPosition(expandedX, frame.y);

        requestAnimationFrame(() => {
            const nextRect = app.getBoundingClientRect();
            // Expanded state should restore with divider at exact midpoint.
            setSplitFromClientX(nextRect.left + (nextRect.width / 2));
            refreshStatusBarLayout();
        });
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

function setTerminalFocusState(focused, options = {}) {
    const nextFocusVisible = Boolean(options.focusVisible);

    if (terminalFocusState === focused && terminalKeyboardFocusVisible === nextFocusVisible) {
        return;
    }

    terminalFocusState = focused;
    terminalKeyboardFocusVisible = focused ? nextFocusVisible : false;
    window.terminalFocusedState = focused;
    updateTerminalFocusChrome();
    TerminalSetFocus(focused).catch(() => {});
}

window.addEventListener('keydown', (event) => {
    // Treat plain-key navigation as keyboard modality for focus ring display.
    if (event.metaKey || event.ctrlKey || event.altKey) {
        return;
    }

    lastInputWasKeyboard = true;
}, true);

window.addEventListener('mousedown', () => {
    lastInputWasKeyboard = false;
}, true);

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
    void toggleNotesPaneCollapsed();
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
    setTerminalJupyterMode(notesCollapsed);

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
