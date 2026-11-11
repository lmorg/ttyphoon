import { CloseNotification, GetWindowStyle, SendIpc, TerminalCancelPaneAutoClose, TerminalClonePane, TerminalClosePane, TerminalConfirmKillPane, TerminalCopyImageDataURL, TerminalGetPanes, TerminalGetTabs, TerminalMouseLeave, TerminalPaneZoom, TerminalRequestRedraw, TerminalResize, TerminalSelectWindow, TerminalSetGlyphSize } from './wails-bindings.js';
import { EventsEmit, EventsOn } from './wails-runtime.js';
import { wireKeyboardEvents, wireMouseEvents } from './events';
import { createFontController } from './font';
import { drawGauge } from './gauge';
import { drawBlockChrome } from './block_chrome';
import { initTerminalPopupMenu } from './popup_menu';
import { initInputBox } from './inputbox';
import { showFullscreenImageOverlay } from './fullscreen-image-overlay';
import { bindSharedTooltipEvents, bindSharedTooltipMouseTracking, closeSharedTooltip, showSharedTooltip } from './shared_tooltip';
import { DARKEN_BACKGROUND_OVERLAY } from './style-utils';
import dingSound from './assets/sound/ding.mp3';

(document.getElementById('terminal-pane') || document.querySelector('#app')).innerHTML = `
    <div id="terminal-app">
        <div id="terminal-tabs" role="tablist" aria-label="tmux windows"></div>
        <div id="terminal-viewport">
            <canvas id="ttyphoon-terminal"></canvas>
            <div id="terminal-pane-controls"></div>
        </div>
        <div id="terminal-notifications"></div>
    </div>
`;

const tabsEl = document.getElementById('terminal-tabs');
const canvas = document.getElementById('ttyphoon-terminal');
const paneControlsEl = document.getElementById('terminal-pane-controls');
const ctx = canvas.getContext('2d');
const offscreen = document.createElement('canvas');
const offCtx = offscreen.getContext('2d');
const font = createFontController(offCtx);
let windowStyle;
let rafPending = false;
let rafHandle = 0;
let rafWatchdog = 0;
let terminalPanes = [];
let paneLayoutVersion = 0;
let paneCountdownInterval = 0;
let hoveredPaneId = '';
let lastPanePointer = null;
let activeTmuxWindowId = '';

function paneIsFinished(pane) {
    return pane?.finished === true || (pane?.finished == null && pane?.canClose === true);
}

function updateHoveredPane(event) {
    if (event) {
        lastPanePointer = { x: event.clientX, y: event.clientY };
    }
    const rect = canvas.getBoundingClientRect();
    const { cellWidth, cellHeight } = font.getCellSize();
    if (!lastPanePointer || rect.width <= 0 || rect.height <= 0 || cellWidth <= 0 || cellHeight <= 0) {
        hoveredPaneId = '';
    } else {
        const cellX = (lastPanePointer.x - rect.left) * canvas.width / rect.width / cellWidth;
        const cellY = (lastPanePointer.y - rect.top) * canvas.height / rect.height / cellHeight;
        const pane = terminalPanes.find((item) => item?.canClose
            && cellX >= item.x && cellX < item.x + item.width
            && cellY >= item.y && cellY < item.y + item.height);
        hoveredPaneId = pane?.id || '';
    }
    for (const button of paneControlsEl.querySelectorAll('.terminal-pane-action')) {
        button.dataset.hovered = String(button.dataset.paneId === hoveredPaneId);
    }
}

function updatePaneCountdowns() {
    let active = false;
    for (const timer of paneControlsEl.querySelectorAll('.terminal-pane-timer')) {
        const remaining = Math.max(0, Number(timer.dataset.deadline) - Date.now());
        const duration = Math.max(1, Number(timer.dataset.duration) || 30000);
        const ratio = Math.max(0, Math.min(1, 1 - remaining / duration));
        timer.firstElementChild.style.setProperty('--progress-deg', `${Math.round(ratio * 360)}deg`);
        timer.setAttribute('aria-label', `Closes in ${Math.ceil(remaining / 1000)}s. Click to keep open`);
        if (remaining > 0) {
            active = true;
        } else {
            timer.disabled = true;
        }
    }
    if (!active && paneCountdownInterval) {
        clearInterval(paneCountdownInterval);
        paneCountdownInterval = 0;
    }
}

function renderPaneControls() {
    if (paneCountdownInterval) {
        clearInterval(paneCountdownInterval);
        paneCountdownInterval = 0;
    }
    paneControlsEl.replaceChildren();
    closeSharedTooltip();
    const { cellWidth, cellHeight } = font.getCellSize();
    if (cellWidth <= 0 || cellHeight <= 0) {
        return;
    }
    for (const pane of terminalPanes) {
        if (!pane?.canClose || !pane.id || ![pane.x, pane.y, pane.width, pane.height].every(Number.isFinite)
            || pane.x < 0 || pane.y < 0 || pane.width <= 0 || pane.height <= 0) {
            continue;
        }
        const finished = paneIsFinished(pane);
        const hasTimer = Number.isFinite(pane.autoCloseAt) && pane.autoCloseAt > 0;
        const controlCount = hasTimer ? 4 : 3;
        const size = Math.min(20, (pane.width * cellWidth - (controlCount - 1) * 4) / controlCount, pane.height * cellHeight);
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'notes-tools-clear terminal-pane-action terminal-pane-close';
        button.dataset.paneId = pane.id;
        button.dataset.finished = String(finished);
        button.dataset.hovered = String(pane.id === hoveredPaneId);
        const closeLabel = finished ? 'Close completed pane' : 'Kill pane';
        button.setAttribute('aria-label', closeLabel);
        button.innerHTML = '&#xf00d;';
        button.style.width = `${size}px`;
        button.style.height = `${size}px`;
        const closeLeft = Math.max(pane.x * cellWidth, (pane.x + pane.width) * cellWidth - size - 2);
        button.style.left = `${closeLeft}px`;
        button.style.top = `${Math.min(pane.y * cellHeight + 2, (pane.y + pane.height) * cellHeight - size)}px`;
        button.addEventListener('mouseenter', () => showSharedTooltip(closeLabel));
        button.addEventListener('mouseleave', closeSharedTooltip);
        button.addEventListener('focus', () => showSharedTooltip(closeLabel));
        button.addEventListener('blur', closeSharedTooltip);
        button.addEventListener('pointerdown', (event) => {
            event.preventDefault();
            event.stopPropagation();
        });
        button.addEventListener('click', async (event) => {
            event.stopPropagation();
            if (finished ? typeof TerminalClosePane !== 'function' : typeof TerminalConfirmKillPane !== 'function') {
                return;
            }
            button.disabled = true;
            closeSharedTooltip();
            try {
                if (finished) {
                    await TerminalClosePane(pane.id);
                    terminalPanes = terminalPanes.filter((item) => item.id !== pane.id);
                    renderPaneControls();
                    canvas.focus();
                } else {
                    await TerminalConfirmKillPane(pane.id, pane.name || '');
                    button.disabled = false;
                }
            } catch (error) {
                button.disabled = false;
                console.error('Failed to close pane', error);
            }
        });
        const zoomButton = document.createElement('button');
        zoomButton.type = 'button';
        zoomButton.className = 'terminal-pane-action terminal-pane-zoom';
        zoomButton.dataset.paneId = pane.id;
        zoomButton.dataset.finished = String(finished);
        zoomButton.dataset.hovered = String(pane.id === hoveredPaneId);
        zoomButton.setAttribute('aria-label', pane.zoomed ? 'Restore pane' : 'Maximize pane');
        zoomButton.title = zoomButton.getAttribute('aria-label');
        zoomButton.innerHTML = pane.zoomed ? '&#xf066;' : '&#xf065;';
        zoomButton.style.width = `${size}px`;
        zoomButton.style.height = `${size}px`;
        zoomButton.style.left = `${closeLeft - size - 4 - (hasTimer ? size + 4 : 0)}px`;
        zoomButton.style.top = button.style.top;
        zoomButton.addEventListener('mouseenter', () => showSharedTooltip(zoomButton.title));
        zoomButton.addEventListener('mouseleave', closeSharedTooltip);
        zoomButton.addEventListener('focus', () => showSharedTooltip(zoomButton.title));
        zoomButton.addEventListener('blur', closeSharedTooltip);
        zoomButton.addEventListener('pointerdown', (event) => {
            event.preventDefault();
            event.stopPropagation();
        });
        zoomButton.addEventListener('click', async (event) => {
            event.stopPropagation();
            if (typeof TerminalPaneZoom !== 'function') {
                return;
            }
            zoomButton.disabled = true;
            closeSharedTooltip();
            try {
                await TerminalPaneZoom(pane.id);
                window.dispatchEvent(new CustomEvent('ttyphoon-terminal-fullsize-toggle'));
            } catch (error) {
                console.error('Failed to zoom pane', error);
            } finally {
                zoomButton.disabled = false;
            }
        });
        const cloneButton = document.createElement('button');
        cloneButton.type = 'button';
        cloneButton.className = 'terminal-pane-action terminal-pane-clone';
        cloneButton.dataset.paneId = pane.id;
        cloneButton.dataset.finished = String(finished);
        cloneButton.dataset.hovered = String(pane.id === hoveredPaneId);
        cloneButton.setAttribute('aria-label', 'Clone pane');
        cloneButton.innerHTML = '&#xf24d;';
        cloneButton.style.width = `${size}px`;
        cloneButton.style.height = `${size}px`;
        cloneButton.style.left = `${parseFloat(zoomButton.style.left) - size - 4}px`;
        cloneButton.style.top = button.style.top;
        cloneButton.addEventListener('mouseenter', () => showSharedTooltip('Clone pane'));
        cloneButton.addEventListener('mouseleave', closeSharedTooltip);
        cloneButton.addEventListener('focus', () => showSharedTooltip('Clone pane'));
        cloneButton.addEventListener('blur', closeSharedTooltip);
        cloneButton.addEventListener('pointerdown', (event) => {
            event.preventDefault();
            event.stopPropagation();
        });
        cloneButton.addEventListener('click', async (event) => {
            event.stopPropagation();
            if (typeof TerminalClonePane !== 'function') return;
            cloneButton.disabled = true;
            closeSharedTooltip();
            try {
                await TerminalClonePane();
                canvas.focus();
            } catch (error) {
                console.error('Failed to clone pane', error);
            } finally {
                cloneButton.disabled = false;
            }
        });
        paneControlsEl.appendChild(cloneButton);
        paneControlsEl.appendChild(zoomButton);
        if (hasTimer) {
            const timer = document.createElement('button');
            timer.type = 'button';
            timer.className = 'terminal-pane-timer';
            timer.dataset.paneId = pane.id;
            timer.dataset.deadline = String(pane.autoCloseAt);
            timer.dataset.duration = String(pane.autoCloseDuration || 30000);
            timer.style.width = `${size}px`;
            timer.style.height = `${size}px`;
            timer.style.left = `${closeLeft - size - 4}px`;
            timer.style.top = button.style.top;
            const progress = document.createElement('span');
            progress.className = 'notif-progress';
            progress.setAttribute('aria-hidden', 'true');
            timer.appendChild(progress);
            timer.addEventListener('mouseenter', () => showSharedTooltip(timer.getAttribute('aria-label')));
            timer.addEventListener('mouseleave', closeSharedTooltip);
            timer.addEventListener('focus', () => showSharedTooltip(timer.getAttribute('aria-label')));
            timer.addEventListener('blur', closeSharedTooltip);
            timer.addEventListener('pointerdown', (event) => {
                event.preventDefault();
                event.stopPropagation();
            });
            timer.addEventListener('click', async (event) => {
                event.stopPropagation();
                if (typeof TerminalCancelPaneAutoClose !== 'function') {
                    return;
                }
                timer.disabled = true;
                closeSharedTooltip();
                try {
                    await TerminalCancelPaneAutoClose(pane.id);
                    terminalPanes = terminalPanes.map((item) => item.id === pane.id && item.autoCloseAt === pane.autoCloseAt
                        ? { ...item, autoCloseAt: 0 } : item);
                    renderPaneControls();
                    canvas.focus();
                } catch (error) {
                    timer.disabled = false;
                    console.error('Failed to cancel pane auto-close', error);
                }
            });
            paneControlsEl.appendChild(timer);
        }
        paneControlsEl.appendChild(button);
    }
    const hasCountdowns = !!paneControlsEl.querySelector('.terminal-pane-timer');
    if (hasCountdowns) {
        updatePaneCountdowns();
        paneCountdownInterval = setInterval(updatePaneCountdowns, 50);
    }
    updateHoveredPane();
}

const terminalViewport = document.getElementById('terminal-viewport');
terminalViewport.addEventListener('pointermove', updateHoveredPane);
terminalViewport.addEventListener('pointerleave', () => {
    lastPanePointer = null;
    updateHoveredPane();
});

function clearRedrawLatch() {
    if (rafWatchdog) {
        clearTimeout(rafWatchdog);
        rafWatchdog = 0;
    }
    rafHandle = 0;
    rafPending = false;
}

const REDRAW_OP = {
    CELL: 1,
    FRAME: 2,
    HIGHLIGHT_RECT: 3,
    RECT_COLOUR: 4,
    BLOCK_CHROME: 5,
    GAUGE_H: 6,
    GAUGE_V: 7,
    TILE_OVERLAY: 8,
    IMAGE: 9,
    TABLE: 10,
    GRAPHICS_IMAGE: 11,
};

const REDRAW_FLAG = {
    FOLDED: 1 << 5,
};

const FRAME_FLAG = {
    RESULT_MASK: 0x3,
    TOP_HALF_INSET: 1 << 2,
    BOTTOM_HALF_INSET: 1 << 3,
};

const TILE_OVERLAY_FLAG = {
    TOP_HALF: 1 << 0,
    BOTTOM_HALF: 1 << 1,
};

// SGR_FLAG mirrors the Go types.SgrFlag iota bit layout.
const SGR_FLAG = {
    BOLD: 1 << 1,           // SGR_BOLD
    ITALIC: 1 << 2,         // SGR_ITALIC
    STRIKE: 1 << 3,         // SGR_STRIKETHROUGH
    SEARCH_RESULT: 1 << 8,  // SGR_HIGHLIGHT_SEARCH_RESULT
    UNDERLINE_SHIFT: 10,    // SGR_UNDERLINE bit position
    UNDERLINE_MASK: 0x7,
};

function colourFrom24(value) {
    const c = Number(value);
    if (!Number.isFinite(c) || c <= 0) {
        return null;
    }

    const n = c >>> 0;
    return {
        Red: (n >>> 16) & 255,
        Green: (n >>> 8) & 255,
        Blue: n & 255,
    };
}

function hasFlag(flags, flag) {
    const n = Number(flags);
    if (!Number.isFinite(n)) {
        return false;
    }
    return (n & flag) !== 0;
}

function toCompactCommand(op) {
    if (!Array.isArray(op) || op.length === 0) {
        return null;
    }

    const kind = Number(op[0]);

    switch (kind) {
        case REDRAW_OP.CELL: {
            const flags = Number(op[5]) || 0;
            // Cell flags are transmitted in raw Go SGR bit layout.
            const underlineStyle = (flags >> SGR_FLAG.UNDERLINE_SHIFT) & SGR_FLAG.UNDERLINE_MASK;
            return {
                op: 'cell',
                x: op[1],
                y: op[2],
                width: op[3],
                char: op[4],
                bold: hasFlag(flags, SGR_FLAG.BOLD),
                italic: hasFlag(flags, SGR_FLAG.ITALIC),
                underlineStyle: underlineStyle, // 0=none, 1=single, 2=double, 3=curly, 4=dotted, 5=dashed
                strike: hasFlag(flags, SGR_FLAG.STRIKE),
                searchResult: hasFlag(flags, SGR_FLAG.SEARCH_RESULT),
                fg: colourFrom24(op[6]),
                bg: colourFrom24(op[7]),
                ulc: colourFrom24(op[8]),
            };
        }

        case REDRAW_OP.FRAME:
            return { op: 'frame', x: op[1], y: op[2], width: op[3], height: op[4], result: Number(op[5]) || 0 };

        case REDRAW_OP.HIGHLIGHT_RECT:
            return {
                op: 'highlight_rect',
                x: op[1],
                y: op[2],
                width: op[3],
                height: op[4],
                fg: colourFrom24(op[5]),
                bg: colourFrom24(op[6]),
            };

        case REDRAW_OP.RECT_COLOUR:
            return {
                op: 'rect_colour',
                x: op[1],
                y: op[2],
                width: op[3],
                height: op[4],
                bg: colourFrom24(op[5]),
            };

        case REDRAW_OP.BLOCK_CHROME: {
            const flags = Number(op[6]) || 0;
            return {
                op: 'block_chrome',
                x: op[1],
                y: op[2],
                height: op[3],
                endX: op[4],
                fg: colourFrom24(op[5]),
                folded: hasFlag(flags, REDRAW_FLAG.FOLDED),
            };
        }

        case REDRAW_OP.GAUGE_H:
            return {
                op: 'gauge_h',
                x: op[1],
                y: op[2],
                width: op[3],
                value: op[4],
                max: op[5],
                fg: colourFrom24(op[6]),
            };

        case REDRAW_OP.GAUGE_V:
            return {
                op: 'gauge_v',
                x: op[1],
                y: op[2],
                height: op[3],
                value: op[4],
                max: op[5],
                fg: colourFrom24(op[6]),
            };

        case REDRAW_OP.TILE_OVERLAY:
            return {
                op: 'tile_overlay',
                x: op[1],
                y: op[2],
                width: op[3],
                height: op[4],
                alpha: op[5],
                flags: Number(op[6]) || 0,
            };

        case REDRAW_OP.IMAGE:
            return {
                op: 'image',
                x: op[1],
                y: op[2],
                width: op[3],
                height: op[4],
                imageId: op[5],
                srcWidth: op[6],
                srcHeight: op[7],
                srcScaleX: Number.isFinite(Number(op[8])) ? Number(op[8]) / 1000 : 0,
                srcScaleY: Number.isFinite(Number(op[9])) ? Number(op[9]) / 1000 : 0,
            };

        case REDRAW_OP.GRAPHICS_IMAGE:
            return {
                op: 'graphics_image', x: op[1], y: op[2],
                pixelX: op[3], pixelY: op[4], width: op[5], height: op[6],
                imageId: op[7], srcX: op[8], srcY: op[9], srcWidth: op[10], srcHeight: op[11],
                clipX: op[12], clipY: op[13], clipWidth: op[14], clipHeight: op[15],
            };

        case REDRAW_OP.TABLE:
            return {
                op: 'table',
                x: op[1],
                y: op[2],
                height: op[3],
                width: op[4],
                fg: colourFrom24(op[5]),
                boundaries: Array.isArray(op[6]) ? op[6] : [],
            };

        default:
            return null;
    }
}

function decodeDrawOpsPayload(payload) {
    const compactOps = Array.isArray(payload?.[0]) && Array.isArray(payload[0][0])
        ? payload[0]
        : payload;

    if (!Array.isArray(compactOps)) {
        return [];
    }

    const decoded = [];
    for (let i = 0; i < compactOps.length; i++) {
        const cmd = toCompactCommand(compactOps[i]);
        if (cmd) {
            decoded.push(cmd);
        }
    }
    return decoded;
}
let tabState = [];
const imageCache = new Map();
const terminalStatusEl = document.getElementById('terminal-status');
const notifContainer = document.getElementById('terminal-notifications');
let cursorRects = [];
let cursorPulseRaf = 0;
let jupyterTabEnabled = false;
let jupyterTabActive = false;
let jupyterTabTitle = 'Notes';
const bellAudio = typeof Audio === 'function' ? new Audio(dingSound) : null;

if (bellAudio) {
    bellAudio.preload = 'auto';
}

function syncAuxTerminalTabState() {
    SendIpc('terminal-extra-tab-state', {
        id: 'notes',
        enabled: jupyterTabEnabled ? 'true' : 'false',
        active: (jupyterTabEnabled && jupyterTabActive) ? 'true' : 'false',
        name: jupyterTabTitle || 'Notes',
    }).catch(() => {});
}

function formatNotesTabTitle(fileName) {
    if (typeof fileName !== 'string' || fileName.length === 0) {
        return 'Notes';
    }
    return `${fileName} (Notes)`;
}

function updateNotificationOffset() {
    if (!notifContainer) {
        return;
    }

    const tabsHeight = tabsEl && tabsEl.style.display !== 'none'
        ? Math.ceil(tabsEl.getBoundingClientRect().height)
        : 0;

    notifContainer.style.top = `${tabsHeight + 8}px`;
}

if (tabsEl) {
    // Convert wheel up/down into horizontal scrolling so hidden tabs are reachable.
    tabsEl.addEventListener('wheel', (event) => {
        if (tabsEl.scrollWidth <= tabsEl.clientWidth) {
            return;
        }

        const delta = event.deltaY !== 0 ? event.deltaY : event.deltaX;
        if (delta === 0) {
            return;
        }

        tabsEl.scrollLeft += delta;
        event.preventDefault();
    }, { passive: false });
}

function renderTerminalTabs(tabs) {
    if (!tabsEl) {
        return;
    }

    tabState = Array.isArray(tabs) ? [...tabs].sort((a, b) => (a?.index ?? 0) - (b?.index ?? 0)) : [];
    const nextActiveWindowId = tabState.find((tab) => tab?.active && tab.id && tab.id !== '__jupyter__')?.id || '';
    if (activeTmuxWindowId && activeTmuxWindowId !== nextActiveWindowId) {
        const zoomedPane = terminalPanes.find((pane) => pane?.zoomed);
        if (zoomedPane && typeof TerminalPaneZoom === 'function') {
            TerminalPaneZoom(zoomedPane.id).catch((error) => console.error('Failed to restore outgoing tmux pane', error));
        }
        if (document.getElementById('terminal-pane')?.dataset.fullsize === 'true') {
            window.dispatchEvent(new CustomEvent('ttyphoon-terminal-fullsize-toggle'));
        }
    }
    activeTmuxWindowId = nextActiveWindowId;
    tabsEl.innerHTML = '';

    for (const tab of tabState) {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'tab terminal-tab';
        button.setAttribute('role', 'tab');
        button.setAttribute('aria-selected', (!jupyterTabActive && tab.active) ? 'true' : 'false');
        button.dataset.windowId = tab.id || '';
        button.textContent = tab.name || tab.id || 'window';
        button.title = tab.id || button.textContent;

        button.addEventListener('click', () => {
            jupyterTabActive = false;
            applyEmbeddedJupyterVisibility();
            syncAuxTerminalTabState();
            if (!tab.id) {
                return;
            }
            TerminalSelectWindow(tab.id).catch(() => {});
            renderTerminalTabs(tabState);
        });

        tabsEl.appendChild(button);
    }

    if (jupyterTabEnabled) {
        const jupyterButton = document.createElement('button');
        jupyterButton.type = 'button';
        jupyterButton.className = 'tab terminal-tab';
        jupyterButton.setAttribute('role', 'tab');
        jupyterButton.dataset.windowId = '__jupyter__';
        jupyterButton.textContent = jupyterTabTitle || 'Notes';
        jupyterButton.title = jupyterButton.textContent;
        jupyterButton.setAttribute('aria-selected', jupyterTabActive ? 'true' : 'false');
        jupyterButton.addEventListener('click', () => {
            jupyterTabActive = true;
            applyEmbeddedJupyterVisibility();
            syncAuxTerminalTabState();
            renderTerminalTabs(tabState);
        });
        tabsEl.appendChild(jupyterButton);
    }

    tabsEl.style.display = (tabState.length > 0 || jupyterTabEnabled) ? 'flex' : 'none';
    updateNotificationOffset();

}

function applyEmbeddedJupyterVisibility() {
    const jupyterHost = document.getElementById('terminal-jupyter-host');
    const showJupyter = jupyterTabEnabled && jupyterTabActive;

    if (jupyterHost) {
        jupyterHost.style.display = showJupyter ? 'block' : 'none';
    }

    canvas.style.display = showJupyter ? 'none' : 'block';
    paneControlsEl.hidden = showJupyter;

    if (!showJupyter) {
        TerminalRequestRedraw().catch(() => {});
    }
}

function applyTerminalStyles(result) {
    const existing = document.getElementById('terminal-theme');
    if (existing) {
        existing.remove();
    }

    const style = document.createElement('style');
    style.id = 'terminal-theme';
    style.textContent = `
        :root {
            --terminal-bg: rgb(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue});
            --terminal-bg-50: rgba(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue}, 0.5);
            --terminal-fg: rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue});
            --terminal-accent: rgb(${result.colors.accent.Red}, ${result.colors.accent.Green}, ${result.colors.accent.Blue});
            --terminal-accent-20: rgba(${result.colors.accent.Red}, ${result.colors.accent.Green}, ${result.colors.accent.Blue}, 0.2);
            --terminal-selection: rgb(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue});
            --terminal-selection-20: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.2);
            --terminal-green: rgb(${result.colors.green.Red}, ${result.colors.green.Green}, ${result.colors.green.Blue});
            --terminal-green-20: rgba(${result.colors.green.Red}, ${result.colors.green.Green}, ${result.colors.green.Blue}, 0.2);
            --terminal-red: rgb(${result.colors.red.Red}, ${result.colors.red.Green}, ${result.colors.red.Blue});
            --terminal-red-20: rgba(${result.colors.red.Red}, ${result.colors.red.Green}, ${result.colors.red.Blue}, 0.2);
            --terminal-menu-fg: rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue});
            --terminal-menu-bg: rgb(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue});
            --terminal-menu-border: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.3);
            --terminal-menu-separator: rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.1);
            --terminal-menu-hover: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.4);
            --terminal-menu-font: ${result.fontFamily};
            --terminal-menu-font-size: 12px;
        }

        #terminal-app {
            display: flex;
            flex-direction: column;
            height: 100%;
            width: 100%;
            min-height: 0;
            position: relative;
        }

        #terminal-tabs {
            display: none;
            gap: 8px;
            align-items: center;
            padding: 6px 8px 0 8px;
            background-color: ${DARKEN_BACKGROUND_OVERLAY};
            border-bottom: 1px solid rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.2);
            flex-wrap: nowrap;
            overflow-x: auto;
            overflow-y: hidden;
            scrollbar-width: none;
            -ms-overflow-style: none;
            box-sizing: border-box;
        }

        #terminal-tabs::-webkit-scrollbar {
            display: none;
        }

        #terminal-tabs button {
            border-radius: 0;
            border: 1px solid transparent;
            background: transparent;
            color: rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue});
            padding: 6px 12px;
            cursor: pointer;
            white-space: nowrap;
            transition: color 0.2s ease, border-color 0.2s ease;
        }

        #terminal-tabs button[aria-selected="true"] {
            background-color: rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.1);
            border-color: rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.2) !important;
        }

        .terminal-tab {
            border-top-left-radius: 5px !important;
            border-top-right-radius: 5px !important;
            border: 1px solid !important;
            border-bottom: 0 !important;
            border-color: rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.2) !important;
        }

        .terminal-tab:not([aria-selected="true"]):hover {
            border-color: var(--terminal-accent) !important;
            color: var(--terminal-accent) !important;
        }

        #terminal-viewport {
            position: relative;
            flex: 1;
            min-height: 0;
            overflow: hidden;
            box-sizing: border-box;
            border: 1px solid;
            border-color: ${DARKEN_BACKGROUND_OVERLAY};
        }

        #terminal-pane-controls {
            position: absolute;
            inset: 0;
            z-index: 11;
            pointer-events: none;
            overflow: hidden;
        }

        #terminal-pane-controls .terminal-pane-clone,
        #terminal-pane-controls .terminal-pane-close,
        #terminal-pane-controls .terminal-pane-timer,
        #terminal-pane-controls .terminal-pane-zoom {
            position: absolute;
            pointer-events: auto;
            display: grid;
            place-items: center;
            padding: 0;
            border: 1px solid var(--terminal-menu-border);
            border-radius: 3px;
            background: var(--terminal-bg-50);
            color: var(--terminal-fg);
            font-family: "Font Awesome Regular", "Font Awesome Solid", "Font Awesome", sans-serif;
            font-weight: 400;
            font-size: 12px;
            line-height: 1;
            cursor: pointer;
        }

        #terminal-pane-controls .terminal-pane-clone:hover,
        #terminal-pane-controls .terminal-pane-clone:focus-visible,
        #terminal-pane-controls .terminal-pane-close:hover,
        #terminal-pane-controls .terminal-pane-close:focus-visible,
        #terminal-pane-controls .terminal-pane-zoom:hover,
        #terminal-pane-controls .terminal-pane-zoom:focus-visible,
        #terminal-pane-controls .terminal-pane-timer:hover,
        #terminal-pane-controls .terminal-pane-timer:focus-visible {
            color: var(--terminal-bg);
            background: var(--terminal-accent);
            border-color: var(--terminal-accent);
        }

        #terminal-pane-controls .terminal-pane-action {
            transition: opacity 120ms ease, transform 120ms ease, visibility 0s;
        }

        #terminal-pane-controls .terminal-pane-action[data-finished="false"][data-hovered="false"] {
            opacity: 0;
            transform: scale(0.9);
            visibility: hidden;
            pointer-events: none;
            transition: opacity 120ms ease, transform 120ms ease, visibility 0s linear 120ms;
        }

        #terminal-pane-controls .terminal-pane-timer {
            background: var(--terminal-green);
        }

        #terminal-pane-controls .terminal-pane-timer .notif-progress {
            display: block;
            width: 70%;
            height: 70%;
            min-width: 0;
            margin: 0;
        }

        #terminal-pane-controls .terminal-pane-clone:disabled,
        #terminal-pane-controls .terminal-pane-close:disabled,
        #terminal-pane-controls .terminal-pane-timer:disabled,
        #terminal-pane-controls .terminal-pane-zoom:disabled {
            opacity: 0.5;
            cursor: default;
        }

        #terminal-viewport[data-tile-count]:not([data-tile-count="1"]) {
            background-color: ${DARKEN_BACKGROUND_OVERLAY} !important;
        }

        #terminal-pane[data-terminal-focused="true"] #terminal-viewport {
            border: 1px solid !important;
            border-color: var(--terminal-accent) !important;
        }

        /* Match the maximized pane's 8px radius so the accent border curves with
           it rather than being clipped square at the corners. */
        #terminal-pane[data-fullsize="true"] #terminal-viewport {
            border-bottom-left-radius: 8px;
            border-bottom-right-radius: 8px;
        }

        #ttyphoon-terminal {
            display: block;
            width: 100%;
            height: 100%;
            outline: none;
        }

    `;
    document.head.appendChild(style);
}

function fitCanvasToWindow() {
    const pane = canvas.parentElement;
    // Ensure we always get valid dimensions; use window size if parent reports 0
    // (which can happen before layout is complete on startup).
    let width = pane?.clientWidth ?? 0;
    let height = pane?.clientHeight ?? 0;
    if (width <= 0) {
        width = window.innerWidth;
    }
    if (height <= 0) {
        height = window.innerHeight;
    }
    canvas.width = width;
    canvas.height = height;
    offscreen.width = canvas.width;
    offscreen.height = canvas.height;
    renderPaneControls();
}

function syncTerminalGridSize() {
    const { cellWidth, cellHeight } = font.getCellSize();
    if (cellWidth <= 0 || cellHeight <= 0) {
        return;
    }

    const cols = Math.floor(canvas.width / cellWidth) - 1;
    const rows = Math.floor(canvas.height / cellHeight);
    if (cols > 0 && rows > 0) {
        TerminalResize(cols, rows).catch(() => {});
    }
}

function drawCursorPulseOverlay(targetCtx) {
    if (!Array.isArray(cursorRects) || cursorRects.length === 0) {
        return;
    }

    const fg = windowStyle?.colors?.fg;
    const colour = fg ? `rgb(${fg.Red}, ${fg.Green}, ${fg.Blue})` : 'rgb(255, 255, 255)';

    // Smooth pulse between 30% and 100% alpha over 1.2s.
    const phase = (performance.now() % 1000) / 1000;
    const animatedAlpha = 0.1 + (0.7 * (0.5 + 0.5 * Math.sin(phase * Math.PI * 2)));

    for (const cursorRect of cursorRects) {
        if (!cursorRect) {
            continue;
        }

        targetCtx.save();
        if (cursorRect.animated) {
            targetCtx.globalAlpha = animatedAlpha;
            // Invert underlying pixels so glyph/background always contrast.
            targetCtx.globalCompositeOperation = 'difference';
            targetCtx.fillStyle = colour;
            targetCtx.fillRect(cursorRect.x, cursorRect.y, cursorRect.width, cursorRect.height);
        } else {
            // Inactive panes: static hollow box cursor.
            targetCtx.globalAlpha = 1;
            targetCtx.strokeStyle = colour;
            targetCtx.lineWidth = 1;
            targetCtx.strokeRect(cursorRect.x + 0.5, cursorRect.y + 0.5, Math.max(1, cursorRect.width - 1), Math.max(1, cursorRect.height - 1));
        }
        targetCtx.restore();
    }
}

function paintTerminalCanvas() {
    // Fill canvas with theme background
    const bg = windowStyle?.colors?.bg || { Red: 0, Green: 0, Blue: 0 };
    ctx.fillStyle = `rgb(${bg.Red}, ${bg.Green}, ${bg.Blue})`;
    ctx.fillRect(0, 0, canvas.width, canvas.height);

    // Draw terminal content
    ctx.drawImage(offscreen, 0, 0);

    // Dim when unfocused or when multiple tiles are present.
    //const viewport = document.getElementById('terminal-viewport');
    //const tileCount = Number(viewport?.getAttribute('data-tile-count')) || 1;
    //if (window.terminalFocusedState === false || tileCount > 1) {
    if (window.terminalFocusedState === false) {
        ctx.fillStyle = DARKEN_BACKGROUND_OVERLAY;
        ctx.fillRect(0, 0, canvas.width, canvas.height);
    }

    drawCursorPulseOverlay(ctx);
}

function ensureCursorPulseLoop() {
    if (!Array.isArray(cursorRects) || !cursorRects.some((cursor) => cursor?.animated) || cursorPulseRaf !== 0) {
        return;
    }

    const tick = () => {
        if (!Array.isArray(cursorRects) || cursorRects.length === 0) {
            cursorPulseRaf = 0;
            return;
        }
        if (!cursorRects.some((cursor) => cursor?.animated)) {
            cursorPulseRaf = 0;
            return;
        }

        paintTerminalCanvas();
        cursorPulseRaf = requestAnimationFrame(tick);
    };

    cursorPulseRaf = requestAnimationFrame(tick);
}

function syncCursorLoopState() {
    if (!cursorRects.some((cursor) => cursor?.animated) && cursorPulseRaf !== 0) {
        cancelAnimationFrame(cursorPulseRaf);
        cursorPulseRaf = 0;
    }
    ensureCursorPulseLoop();
}

function sameHighlightRect(a, b) {
    return a && b &&
        a.op === 'highlight_rect' &&
        b.op === 'highlight_rect' &&
        a.x === b.x &&
        a.y === b.y &&
        a.width === b.width &&
        a.height === b.height;
}

function isCursorMarkerRect(cmd) {
    if (!cmd || cmd.op !== 'highlight_rect') {
        return false;
    }

    const width = Number.isFinite(cmd.width) ? cmd.width : 0;
    const height = Number.isFinite(cmd.height) ? cmd.height : 0;
    if (height !== 1 || width < 1 || width > 2) {
        return false;
    }

    // Cursor marker emits same fg/bg colour values.
    const fg = cmd.fg;
    const bg = cmd.bg;
    if (!fg || !bg) {
        return false;
    }

    return fg.Red === bg.Red && fg.Green === bg.Green && fg.Blue === bg.Blue;
}

async function copyCanvasSelectionAsPng(payload) {
    if (!payload || typeof payload !== 'object') {
        return;
    }

    const { cellWidth, cellHeight } = font.getCellSize();
    if (cellWidth <= 0 || cellHeight <= 0) {
        return;
    }

    const xCells = Number.isFinite(payload.x) ? payload.x : 0;
    const yCells = Number.isFinite(payload.y) ? payload.y : 0;
    const widthCells = Number.isFinite(payload.width) ? payload.width : 0;
    const heightCells = Number.isFinite(payload.height) ? payload.height : 0;
    if (widthCells <= 0 || heightCells <= 0) {
        return;
    }

    const sx = Math.max(0, Math.floor(xCells * cellWidth));
    const sy = Math.max(0, Math.floor(yCells * cellHeight));
    const sw = Math.min(canvas.width - sx, Math.ceil(widthCells * cellWidth));
    const sh = Math.min(canvas.height - sy, Math.ceil(heightCells * cellHeight));
    if (sw <= 0 || sh <= 0) {
        return;
    }

    const copyCanvas = document.createElement('canvas');
    copyCanvas.width = sw;
    copyCanvas.height = sh;
    const copyCtx = copyCanvas.getContext('2d');
    if (!copyCtx) {
        return;
    }

    copyCtx.drawImage(canvas, sx, sy, sw, sh, 0, 0, sw, sh);

    const dataURL = copyCanvas.toDataURL('image/png');
    if (typeof dataURL !== 'string' || dataURL.length === 0) {
        return;
    }

    await TerminalCopyImageDataURL(dataURL);
}

function drawCell(cmd) {
    if (!offCtx) {
        return;
    }

    const { cellWidth, cellHeight } = font.getCellSize();
    const { y: glyphOffsetY } = font.getGlyphOffset();

    const xCell = Number.isFinite(cmd.x) ? cmd.x : 0;
    const yCell = Number.isFinite(cmd.y) ? cmd.y : 0;
    const widthCells = Number.isFinite(cmd.width) && cmd.width > 0 ? cmd.width : 1;

    const x = xCell * cellWidth;
    const y = yCell * cellHeight;
    const glyphY = y + glyphOffsetY;
    const width = widthCells * cellWidth;

    if (cmd.bg) {
        offCtx.fillStyle = `rgb(${cmd.bg.Red}, ${cmd.bg.Green}, ${cmd.bg.Blue})`;
        offCtx.fillRect(x, y, width, cellHeight);
    }

    font.applyCellStyle(cmd);

    if (cmd.fg) {
        offCtx.fillStyle = `rgb(${cmd.fg.Red}, ${cmd.fg.Green}, ${cmd.fg.Blue})`;
    } else {
        offCtx.fillStyle = '#ffffff';
    }
    const textColour = offCtx.fillStyle;

    if (cmd.char) {
        offCtx.save();
        offCtx.beginPath();
        offCtx.rect(x, y, width, cellHeight);
        offCtx.clip();

        if (cmd.searchResult) {
            const wsr = windowStyle?.colors?.searchResult;
            const outline = wsr
                ? `rgb(${wsr.Red}, ${wsr.Green}, ${wsr.Blue})`
                : 'rgb(64, 64, 255)';

            const wswb = windowStyle?.colors?.whiteBright;
            const fill = wswb
                ? `rgb(${wswb.Red}, ${wswb.Green}, ${wswb.Blue})`
                : 'rgb(64, 64, 255)';
            
            offCtx.lineWidth = 1;
            offCtx.strokeStyle = outline;
            offCtx.strokeText(cmd.char, x, glyphY);
                
            offCtx.shadowColor = outline;
            offCtx.shadowBlur = 6;
            offCtx.fillStyle = fill;
        }
        offCtx.fillText(cmd.char, x, glyphY);
        if (cmd.searchResult) {
            offCtx.shadowColor = 'transparent';
            offCtx.shadowBlur = 0;
            offCtx.lineWidth = 0;
            offCtx.strokeStyle = 'transparent';
        }
        offCtx.restore();
    }

    if (cmd.underlineStyle && cmd.underlineStyle > 0) {
        const underlineColour = cmd.ulc
            ? `rgb(${cmd.ulc.Red}, ${cmd.ulc.Green}, ${cmd.ulc.Blue})`
            : textColour;
        drawUnderline(cmd.underlineStyle, x, y, width, cellHeight, underlineColour);
    }

    if (cmd.strike) {
        const lineY = y + Math.floor(cellHeight / 2);
        offCtx.fillRect(x, lineY, width, 1);
    }
}

function drawUnderline(style, x, y, width, cellHeight, colour) {
    // Underline styles: 0=none, 1=single, 2=double, 3=curly/wavy, 4=dotted, 5=dashed
    const baseY = y + cellHeight - 2;
    const underlineColour = colour || '#ffffff';

    offCtx.save();
    offCtx.fillStyle = underlineColour;
    offCtx.strokeStyle = underlineColour;
    offCtx.lineWidth = 1;

    switch (style) {
        case 1: // single underline
            offCtx.fillRect(x, baseY, width, 1);
            break;

        case 2: { // double underline
            offCtx.fillRect(x, baseY - 1, width, 1);
            offCtx.fillRect(x, baseY + 1, width, 1);
            break;
        }

        case 3: { // curly/wavy underline
            const amplitude = 1.5;
            const frequency = 0.4;
            offCtx.beginPath();
            for (let i = 0; i <= width; i++) {
                const waveY = baseY + Math.sin((x + i) * frequency) * amplitude;
                if (i === 0) {
                    offCtx.moveTo(x + i, waveY);
                } else {
                    offCtx.lineTo(x + i, waveY);
                }
            }
            offCtx.stroke();
            break;
        }

        case 4: { // dotted underline
            offCtx.setLineDash([2, 2]);
            offCtx.beginPath();
            offCtx.moveTo(x, baseY);
            offCtx.lineTo(x + width, baseY);
            offCtx.stroke();
            offCtx.setLineDash([]);
            break;
        }

        case 5: { // dashed underline
            offCtx.setLineDash([4, 3]);
            offCtx.beginPath();
            offCtx.moveTo(x, baseY);
            offCtx.lineTo(x + width, baseY);
            offCtx.stroke();
            offCtx.setLineDash([]);
            break;
        }
    }

    offCtx.restore();
}

function getOrLoadImageById(imageId) {
    if (!Number.isFinite(imageId)) {
        return null;
    }

    return imageCache.get(imageId) || null;
}

function drawImageCommand(cmd) {
    if (!offCtx) {
        return;
    }

    const imageId = Number.isFinite(cmd.imageId) ? cmd.imageId : Number.NaN;
    const img = getOrLoadImageById(imageId);
    if (!img || !img.complete || img.naturalWidth <= 0 || img.naturalHeight <= 0) {
        return;
    }

    const { cellWidth, cellHeight } = font.getCellSize();
    const xCell = Number.isFinite(cmd.x) ? cmd.x : 0;
    const yCell = Number.isFinite(cmd.y) ? cmd.y : 0;
    const widthCells = Number.isFinite(cmd.width) && cmd.width > 0 ? cmd.width : 0;
    const heightCells = Number.isFinite(cmd.height) && cmd.height > 0 ? cmd.height : 0;

    if (widthCells <= 0 || heightCells <= 0) {
        return;
    }

    const scaleX = Number.isFinite(cmd.srcScaleX) && cmd.srcScaleX > 0
        ? Math.min(1, cmd.srcScaleX)
        : null;
    const scaleY = Number.isFinite(cmd.srcScaleY) && cmd.srcScaleY > 0
        ? Math.min(1, cmd.srcScaleY)
        : null;

    const srcWidth = scaleX !== null
        ? Math.max(1, Math.round(img.naturalWidth * scaleX))
        : (Number.isFinite(cmd.srcWidth) && cmd.srcWidth > 0 ? cmd.srcWidth : img.naturalWidth);
    const srcHeight = scaleY !== null
        ? Math.max(1, Math.round(img.naturalHeight * scaleY))
        : (Number.isFinite(cmd.srcHeight) && cmd.srcHeight > 0 ? cmd.srcHeight : img.naturalHeight);

    offCtx.drawImage(
        img,
        0,
        0,
        Math.min(srcWidth, img.naturalWidth),
        Math.min(srcHeight, img.naturalHeight),
        xCell * cellWidth,
        yCell * cellHeight,
        widthCells * cellWidth,
        heightCells * cellHeight,
    );
}

function drawGraphicsImageCommand(cmd) {
    if (!offCtx) return;
    const img = getOrLoadImageById(cmd.imageId);
    if (!img?.complete || img.naturalWidth <= 0 || img.naturalHeight <= 0) return;
    const { cellWidth, cellHeight } = font.getCellSize();
    offCtx.save();
    offCtx.beginPath();
    offCtx.rect(cmd.clipX * cellWidth, cmd.clipY * cellHeight,
        cmd.clipWidth * cellWidth, cmd.clipHeight * cellHeight);
    offCtx.clip();
    offCtx.drawImage(img, cmd.srcX, cmd.srcY, cmd.srcWidth, cmd.srcHeight,
        cmd.x * cellWidth + cmd.pixelX, cmd.y * cellHeight + cmd.pixelY, cmd.width, cmd.height);
    offCtx.restore();
}

function drawFrame(cmd = null) {
    if (!offCtx) {
        return;
    }

    const { cellWidth, cellHeight } = font.getCellSize();

    const xCell = Number.isFinite(cmd?.x) ? cmd.x : 0;
    const yCell = Number.isFinite(cmd?.y) ? cmd.y : 0;
    const widthCells = Number.isFinite(cmd?.width) && cmd.width > 0 ? cmd.width : null;
    const heightCells = Number.isFinite(cmd?.height) && cmd.height > 0 ? cmd.height : null;
    const frameFlags = Number.isFinite(cmd?.result) ? cmd.result : 0;
    const topInset = (frameFlags & FRAME_FLAG.TOP_HALF_INSET) !== 0 ? cellHeight / 2 : 0;
    const bottomInset = (frameFlags & FRAME_FLAG.BOTTOM_HALF_INSET) !== 0 ? cellHeight / 2 : 0;

    const x = Math.max(0, Math.floor(xCell * cellWidth));
    const frameY = yCell * cellHeight + topInset;
    const y = Math.max(0, frameY);
    const clippedTop = y - frameY;

    let width = offscreen.width;
    let height = offscreen.height;

    if (cellWidth > 0 && cellHeight > 0) {
        const cols = widthCells ?? Math.floor(offscreen.width / cellWidth);
        const rows = heightCells ?? Math.floor(offscreen.height / cellHeight);

        width = Math.max(0, cols * cellWidth);
        height = Math.max(0, rows * cellHeight - topInset + bottomInset - clippedTop);
    }

    width = Math.min(width, Math.max(0, offscreen.width-x)) + 2;
    height = Math.min(height, Math.max(0, offscreen.height-y)) + (bottomInset === 0 ? 2 : 0);

    if (width <= 0 || height <= 0) {
        return;
    }

    const bg = windowStyle?.colors?.bg || { Red: 0, Green: 0, Blue: 0 };
    const result = frameFlags & FRAME_FLAG.RESULT_MASK;
    const tint = result === 1 ? windowStyle?.colors?.green : result === 2 ? windowStyle?.colors?.red : null;
    const background = tint ? {
        Red: Math.round(bg.Red * 0.8 + tint.Red * 0.2),
        Green: Math.round(bg.Green * 0.8 + tint.Green * 0.2),
        Blue: Math.round(bg.Blue * 0.8 + tint.Blue * 0.2),
    } : bg;
    offCtx.fillStyle = `rgb(${background.Red}, ${background.Green}, ${background.Blue})`;
    offCtx.fillRect(x, y, width, height);
}

function drawHighlightRect(cmd) {
    if (!offCtx) {
        return;
    }

    const { cellWidth, cellHeight } = font.getCellSize();

    const xCell = Number.isFinite(cmd.x) ? cmd.x : 0;
    const yCell = Number.isFinite(cmd.y) ? cmd.y : 0;
    const widthCells = Number.isFinite(cmd.width) && cmd.width > 0 ? cmd.width : 0;
    const heightCells = Number.isFinite(cmd.height) && cmd.height > 0 ? cmd.height : 0;

    if (widthCells <= 0 || heightCells <= 0) {
        return;
    }

    const x = xCell * cellWidth;
    const y = yCell * cellHeight;
    const width = widthCells * cellWidth;
    const height = heightCells * cellHeight;

    if (cmd.fg) {
        offCtx.save();
        offCtx.globalAlpha = 190 / 255;
        offCtx.strokeStyle = `rgb(${cmd.fg.Red}, ${cmd.fg.Green}, ${cmd.fg.Blue})`;
        offCtx.strokeRect(x - 1, y - 1, width + 2, height + 2);
        offCtx.strokeRect(x, y, width, height);
        offCtx.restore();
    }

    if (cmd.bg) {
        offCtx.save();
        offCtx.globalAlpha = 128 / 255;
        offCtx.fillStyle = `rgba(${cmd.bg.Red}, ${cmd.bg.Green}, ${cmd.bg.Blue}, 0.2)`;
        offCtx.fillRect(x + 1, y + 1, Math.max(0, width - 2), Math.max(0, height - 2));
        offCtx.restore();
    }
}

// Fill-only rect using the supplied colour — no stroke border.
// Used for selection highlights and hover fills where a border would be intrusive.
function drawRectColour(cmd) {
    if (!offCtx) {
        return;
    }

    const { cellWidth, cellHeight } = font.getCellSize();

    const xCell = Number.isFinite(cmd.x) ? cmd.x : 0;
    const yCell = Number.isFinite(cmd.y) ? cmd.y : 0;
    const widthCells = Number.isFinite(cmd.width) && cmd.width > 0 ? cmd.width : 0;
    const heightCells = Number.isFinite(cmd.height) && cmd.height > 0 ? cmd.height : 0;

    if (widthCells <= 0 || heightCells <= 0) {
        return;
    }

    const colour = cmd.bg || cmd.fg;
    if (!colour) {
        return;
    }

    const x = xCell * cellWidth;
    const y = yCell * cellHeight;
    const width = widthCells * cellWidth;
    const height = heightCells * cellHeight;

    offCtx.save();
    offCtx.globalAlpha = 0.4;
    offCtx.fillStyle = `rgb(${colour.Red}, ${colour.Green}, ${colour.Blue})`;
    offCtx.fillRect(x, y, width, height);
    offCtx.restore();
}

function drawTable(cmd) {
    if (!offCtx) {
        return;
    }

    const { cellWidth, cellHeight } = font.getCellSize();
    const fg = cmd.fg || { Red: 128, Green: 128, Blue: 128 };

    offCtx.save();
    offCtx.strokeStyle = `rgb(${fg.Red}, ${fg.Green}, ${fg.Blue})`;
    offCtx.globalAlpha = 0.2;
    offCtx.lineWidth = 1;

    const xCell = Number.isFinite(cmd.x) ? cmd.x : 0;
    const yCell = Number.isFinite(cmd.y) ? cmd.y : 0;
    const heightCells = Number.isFinite(cmd.height) ? cmd.height : 0;
    const maxWidthCells = Number.isFinite(cmd.width) ? cmd.width : 0;

    const x = xCell * cellWidth;
    const y = yCell * cellHeight;
    const h = y + (heightCells * cellHeight);
    const endX = x + (maxWidthCells * cellWidth);

    // Draw vertical line at left border
    offCtx.beginPath();
    offCtx.moveTo(x, y);
    offCtx.lineTo(x, h);
    offCtx.stroke();

    // Draw vertical lines at each boundary
    if (Array.isArray(cmd.boundaries)) {
        for (let i = 0; i < cmd.boundaries.length; i++) {
            const boundaryX = x + (cmd.boundaries[i] * cellWidth);
            offCtx.beginPath();
            offCtx.moveTo(boundaryX, y);
            offCtx.lineTo(boundaryX, h);
            offCtx.stroke();
        }
    }

    // Draw horizontal lines (top and for each row)
    for (let i = 0; i <= heightCells; i++) {
        const lineY = y + (i * cellHeight);
        offCtx.beginPath();
        offCtx.moveTo(x, lineY);
        offCtx.lineTo(endX, lineY);
        offCtx.stroke();
    }

    offCtx.restore();
}

function drawTileOverlay(cmd) {
    if (!offCtx) {
        return;
    }

    const { cellWidth, cellHeight } = font.getCellSize();

    const xCell = Number.isFinite(cmd.x) ? cmd.x : 0;
    const yCell = Number.isFinite(cmd.y) ? cmd.y : 0;
    const widthCells = Number.isFinite(cmd.width) && cmd.width > 0 ? cmd.width : 0;
    const heightCells = Number.isFinite(cmd.height) && cmd.height > 0 ? cmd.height : 0;

    if (widthCells <= 0 || heightCells <= 0) {
        return;
    }

    const x = xCell * cellWidth;
    const flags = Number(cmd.flags) || 0;
    const topHalf = (flags & TILE_OVERLAY_FLAG.TOP_HALF) !== 0;
    const bottomHalf = (flags & TILE_OVERLAY_FLAG.BOTTOM_HALF) !== 0;
    const y = yCell * cellHeight + (bottomHalf ? cellHeight / 2 : 0);
    const width = widthCells * cellWidth;
    const height = topHalf || bottomHalf ? cellHeight / 2 : heightCells * cellHeight;

    const alpha = Number.isFinite(cmd.alpha) ? Math.max(0, Math.min(255, cmd.alpha)) / 255 : 0.5;
    const bg = cmd.bg;

    offCtx.save();
    offCtx.globalAlpha = alpha;
    offCtx.fillStyle = bg ? `rgb(${bg.Red}, ${bg.Green}, ${bg.Blue})` : 'rgb(0, 0, 0)';
    offCtx.fillRect(x, y, width, height);
    offCtx.restore();
}

EventsOn("setCursor", css => {
    canvas.style.cursor = css;
});

canvas.addEventListener('mousemove', (event) => {
    void event;
});

canvas.addEventListener('mouseleave', () => {
    closeSharedTooltip();
    if (typeof TerminalMouseLeave === 'function') {
        TerminalMouseLeave().catch(() => {});
    }
});

EventsOn("terminalImageCachePut", payload => {
    const data = Array.isArray(payload?.[0]) ? payload[0] : payload;
    const imageId = Number(data?.id);
    const imageData = data?.data;

    if (!Number.isFinite(imageId) || typeof imageData !== 'string' || imageData.length === 0) {
        return;
    }

    const img = new Image();
    imageCache.set(imageId, img);
    img.onload = () => {
        TerminalRequestRedraw().catch(() => {});
    };
    img.src = imageData;
});

EventsOn("terminalImageCacheDelete", payload => {
    const raw = Array.isArray(payload?.[0]) ? payload[0] : payload;
    const imageId = Number(raw);
    if (!Number.isFinite(imageId)) {
        return;
    }

    const img = imageCache.get(imageId);
    if (img) {
        img.src = '';
    }
    imageCache.delete(imageId);
});

EventsOn("terminalRedraw", payload => {
    const batchId = Number.isSafeInteger(payload?.id) && payload.id > 0 ? payload.id : null;
    const ops = batchId === null ? payload : payload.ops;
    let ownsRedrawLatch = false;
    try {
        // Guard against startup race where canvas dimensions are 0.
        // If parent now has layout, re-fit to proper dimensions.
        if (canvas.width === 0 || canvas.height === 0) {
            fitCanvasToWindow();
        }

        const drawOps = decodeDrawOpsPayload(ops);

        if (!Array.isArray(drawOps) || drawOps.length === 0) {
            return;
        }

        // Apply every batch to the offscreen canvas; only coalesce visible paints.
        if (!rafPending) {
            rafPending = true;
            ownsRedrawLatch = true;
        }

        cursorRects = [];

        for (let i = 0; i < drawOps.length; i++) {
            const cmd = drawOps[i];
            if (cmd.op === 'frame') {
                drawFrame(cmd);
                continue;
            }
            if (cmd.op === 'cell') {
                drawCell(cmd);
                continue;
            }
            if (cmd.op === 'image') {
                drawImageCommand(cmd);
                continue;
            }
            if (cmd.op === 'graphics_image') {
                drawGraphicsImageCommand(cmd);
                continue;
            }
            if (cmd.op === 'gauge_h' || cmd.op === 'gauge_v') {
                drawGauge(offCtx, font.getCellSize, cmd, windowStyle?.colors?.fg);
                continue;
            }
            if (cmd.op === 'block_chrome') {
                drawBlockChrome(offCtx, font.getCellSize, cmd);
                continue;
            }
            if (cmd.op === 'tile_overlay') {
                if (window.terminalFocusedState === true) {
                    drawTileOverlay(cmd);
                }
                continue;
            }
            if (cmd.op === 'highlight_rect') {
                if (isCursorMarkerRect(cmd)) {
                    const { cellWidth, cellHeight } = font.getCellSize();
                    const animated = isCursorMarkerRect(drawOps[i + 1]) && sameHighlightRect(cmd, drawOps[i + 1]);
                    cursorRects.push({
                        x: (Number.isFinite(cmd.x) ? cmd.x : 0) * cellWidth,
                        y: (Number.isFinite(cmd.y) ? cmd.y : 0) * cellHeight,
                        width: (Number.isFinite(cmd.width) ? cmd.width : 1) * cellWidth,
                        height: (Number.isFinite(cmd.height) ? cmd.height : 1) * cellHeight,
                        animated,
                    });
                    if (animated) {
                        i += 1;
                    }
                    continue;
                }
                drawHighlightRect(cmd);
                continue;
            }
            if (cmd.op === 'rect_colour') {
                drawRectColour(cmd);
                continue;
            }
            if (cmd.op === 'table') {
                drawTable(cmd);
                continue;
            }
        }

        if (!ownsRedrawLatch) {
            return;
        }

        rafHandle = requestAnimationFrame(() => {
            try {
                paintTerminalCanvas();
                syncCursorLoopState();
            } catch (err) {
                console.error('terminal redraw RAF failed', err);
            } finally {
                clearRedrawLatch();
            }
        });

        // WebKit drops pending rAF callbacks across occlusion and display sleep;
        // without this the latch never clears and the canvas stops updating.
        rafWatchdog = setTimeout(() => {
            if (!rafPending) {
                return;
            }
            if (rafHandle) {
                cancelAnimationFrame(rafHandle);
            }
            try {
                paintTerminalCanvas();
                syncCursorLoopState();
            } catch (err) {
                console.error('terminal redraw fallback failed', err);
            } finally {
                clearRedrawLatch();
            }
            TerminalRequestRedraw().catch(() => {});
        }, 1000);
    } catch (err) {
        console.error('terminal redraw failed', err);
        if (ownsRedrawLatch) {
            clearRedrawLatch();
        }
    } finally {
        if (batchId !== null) {
            try {
                Promise.resolve(EventsEmit('ttyphoon:terminal-redraw-ack', batchId))
                    .catch((err) => console.error('terminal redraw acknowledgement failed', err));
            } catch (err) {
                console.error('terminal redraw acknowledgement failed', err);
            }
        }
    }
});

function resumeTerminalAfterSuspend() {
    clearRedrawLatch();
    if (!document.hidden) {
        // Backend pauses rendering while the window is hidden; this resumes it
        // if a native show event was missed.
        Promise.resolve(EventsEmit('ttyphoon:terminal-visible')).catch(() => {});
    }
    TerminalRequestRedraw().catch(() => {});
}

document.addEventListener('visibilitychange', () => {
    if (!document.hidden) {
        resumeTerminalAfterSuspend();
    }
});
window.addEventListener('focus', resumeTerminalAfterSuspend);
window.addEventListener('pageshow', resumeTerminalAfterSuspend);

EventsOn("terminalPanes", payload => {
    const panes = Array.isArray(payload?.[0]) ? payload[0] : payload;
    if (!Array.isArray(panes)) {
        return;
    }
    paneLayoutVersion += 1;
    terminalPanes = panes;
    updateHoveredPane();
    document.getElementById('terminal-viewport').setAttribute('data-tile-count', String(panes.length));
    renderPaneControls();
});

EventsOn("terminalTabs", payload => {
    // Handle both old format (array of tabs) and new format (object with tabs and tileCount)
    let tabs;
    let tileCount = 1;
    
    if (Array.isArray(payload?.[0])) {
        // Old format: [tabs array]
        tabs = payload[0];
    } else if (payload?.tabs) {
        // New format: {tabs: [], tileCount: N}
        tabs = payload.tabs;
        tileCount = payload.tileCount || 1;
    } else if (Array.isArray(payload)) {
        // Fallback: direct array
        tabs = payload;
    } else {
        tabs = [];
    }
    
    // Store tile count on viewport for CSS queries
    const viewport = document.getElementById('terminal-viewport');
    if (viewport) {
        viewport.setAttribute('data-tile-count', String(tileCount));
    }
    
    renderTerminalTabs(tabs);
    fitCanvasToWindow();
});

EventsOn("terminalStatusBarText", payload => {
    if (!terminalStatusEl) {
        return;
    }

    const text = Array.isArray(payload?.[0]) ? payload[0] : payload;
    terminalStatusEl.textContent = typeof text === 'string' ? text : '';
});

EventsOn("terminalCopyImageSelection", payload => {
    const data = Array.isArray(payload) ? payload[0] : payload;
    copyCanvasSelectionAsPng(data).catch(() => {});
});

GetWindowStyle().then((result) => {
    windowStyle = result;
    document.body.style.margin = '0';
    document.body.style.overflow = 'hidden';
    document.body.style.backgroundColor = `rgb(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue})`;
    applyTerminalStyles(result);
    bindSharedTooltipMouseTracking();
    bindSharedTooltipEvents(EventsOn);
    font.applyConfiguredFontFromWindowStyle(windowStyle);
    
    // Defer canvas fitting to ensure browser has computed layout dimensions
    requestAnimationFrame(() => {
        fitCanvasToWindow();
        
        font.loadGlyphSizeFromGo(windowStyle).then(() => {
            const { cellWidth, cellHeight } = font.getCellSize();
            TerminalSetGlyphSize(Math.floor(cellWidth), Math.floor(cellHeight)).catch(() => {});
            syncTerminalGridSize();

            //drawFrame();
            wireKeyboardEvents(canvas);
            wireMouseEvents(canvas, font.getCellSize);
            initTerminalPopupMenu(canvas);
            canvas.focus();

            TerminalGetTabs().then((tabs) => {
                renderTerminalTabs(tabs);
                fitCanvasToWindow();
            }).catch(() => {});

            if (typeof TerminalGetPanes === 'function') {
                const version = paneLayoutVersion;
                TerminalGetPanes().then((panes) => {
                    if (version === paneLayoutVersion && Array.isArray(panes)) {
                        terminalPanes = panes;
                        renderPaneControls();
                    }
                }).catch(() => {});
            }

            TerminalRequestRedraw().catch(() => {});
        });
    });
});

initInputBox(canvas);

// ------------------------------------------------------------------
// Notification overlay
// ------------------------------------------------------------------

const _notifyBg = ['#316db0', '#99c0d3', '#f2b71f', '#de333b', '#316db0', '#74953c'];
const _notifyFg = ['#000000', '#000000', '#000000', '#000000', '#000000', '#000000'];
const _stickySpinnerFrames = [
    "⣾", "⡥", "⡤", "⢀", "⡴", "⡪", "⢔", "⢙", "⢼", "⣊", "⣥", "⡼", "⡹", "⡵",
	"⠿", "⣇", "⠇", "⠧", "⣓", "⠻", "⢿", "⣴", "⣦", "⢷", "⡶", "⠛", "⠾", "⣟",
];
const _stickySpinnerTimers = new Map();

function sanitizeStickyMessage(message) {
    if (typeof message !== 'string') {
        return '';
    }

    // Strip an existing trailing braille spinner glyph from legacy sticky messages.
    return message.replace(/\s[\u2800-\u28FF]$/u, '');
}

function startStickySpinner(el, notifID) {
    if (!el) {
        return;
    }

    let spinner = el.querySelector('.notif-spinner');
    if (!spinner) {
        spinner = document.createElement('span');
        spinner.className = 'notif-spinner';
        spinner.setAttribute('aria-hidden', 'true');
        el.appendChild(spinner);
    }

    const existingTimer = _stickySpinnerTimers.get(notifID);
    if (existingTimer) {
        return;
    }

    let i = 0;
    spinner.textContent = _stickySpinnerFrames[i];
    const timer = setInterval(() => {
        i = (i + 1) % _stickySpinnerFrames.length;
        spinner.textContent = _stickySpinnerFrames[i];
    }, 100);
    _stickySpinnerTimers.set(notifID, timer);
}

function stopStickySpinner(notifID) {
    const timer = _stickySpinnerTimers.get(notifID);
    if (timer) {
        clearInterval(timer);
        _stickySpinnerTimers.delete(notifID);
    }
}

EventsOn('terminalStyleUpdate', payload => {
    const result = Array.isArray(payload?.[0]) ? payload[0] : payload;
    if (!result || !result.colors) {
        return;
    }
    windowStyle = result;
    applyTerminalStyles(result);

    const fontChanged = font.applyConfiguredFontFromWindowStyle(windowStyle);
    if (!fontChanged) {
        TerminalRequestRedraw().catch(() => {});
        return;
    }

    fitCanvasToWindow();
    font.loadGlyphSizeFromGo(windowStyle).then(() => {
        const { cellWidth, cellHeight } = font.getCellSize();
        TerminalSetGlyphSize(Math.floor(cellWidth), Math.floor(cellHeight)).catch(() => {});
        syncTerminalGridSize();
        fitCanvasToWindow();
        TerminalRequestRedraw().catch(() => {});
    }).catch(() => {
        TerminalRequestRedraw().catch(() => {});
    });
});

EventsOn('terminalNotification', payload => {
    const p = Array.isArray(payload?.[0]) ? payload[0] : payload;
    if (!p || !notifContainer) return;

    // update message if already shown (e.g. SetMessage)
    const existing = notifContainer.querySelector(`[data-notif-id="${p.id}"]`);
    if (existing) {
        const message = p.sticky ? sanitizeStickyMessage(p.message) : p.message;
        existing.querySelector('.notif-msg').textContent = message;
        if (p.sticky) {
            existing.classList.add('is-sticky');
            startStickySpinner(existing, p.id);
        }
        return;
    }

    const type  = p.type ?? 1;
    const bg    = _notifyBg[type] ?? _notifyBg[1];
    const fg    = _notifyFg[type] ?? '#000000';

    const el = document.createElement('div');
    el.className = 'terminal-notification';
    el.dataset.notifId = p.id;
    el.dataset.notifType = String(type);
    el.style.cssText = `background:${bg};color:${fg};`;
    el.innerHTML = `<span class="notif-icon" role="img" aria-label="notification icon"></span><span class="notif-msg"></span>`;
    const message = p.sticky ? sanitizeStickyMessage(p.message) : p.message;
    el.querySelector('.notif-msg').textContent = message;

    if (p.sticky) {
        el.classList.add('is-sticky');
        startStickySpinner(el, p.id);
        el.addEventListener('click', () => {
            const id = Number(p.id);
            if (!Number.isFinite(id)) {
                return;
            }
            CloseNotification(id).catch(() => {});
        });
    }

    if (!p.sticky) {
        // Add pie progress indicator for non-sticky notifications.
        const progress = document.createElement('span');
        progress.className = 'notif-progress';
        progress.setAttribute('aria-hidden', 'true');
        progress.style.setProperty('--progress-deg', '0deg');
        el.appendChild(progress);

        const durationMs = 5000;
        const startAt = Date.now();
        const progressInterval = setInterval(() => {
            const elapsed = Date.now() - startAt;
            const ratio = Math.max(0, Math.min(1, elapsed / durationMs));
            progress.style.setProperty('--progress-deg', `${Math.round(ratio * 360)}deg`);
            if (ratio >= 1) {
                clearInterval(progressInterval);
            }
        }, 50);

        const dismissTimeout = setTimeout(() => {
            clearInterval(progressInterval);
            el.classList.add('fade-out');
            el.addEventListener('animationend', () => el.remove(), { once: true });
        }, durationMs);

        // Store timers on element for cleanup.
        el._dismissTimeout = dismissTimeout;
        el._progressInterval = progressInterval;

        el.addEventListener('click', () => {
            clearTimeout(el._dismissTimeout);
            clearInterval(el._progressInterval);
            el.classList.add('fade-out');
            el.addEventListener('animationend', () => el.remove(), { once: true });
        });
    }

    notifContainer.appendChild(el);
});

EventsOn('terminalNotificationClose', payload => {
    const id = Array.isArray(payload) ? payload[0] : payload;
    if (!notifContainer) return;
    stopStickySpinner(id);
    const el = notifContainer.querySelector(`[data-notif-id="${id}"]`);
    if (el) {
        // Clear any pending timeout
        if (el._dismissTimeout) {
            clearTimeout(el._dismissTimeout);
        }
        if (el._progressInterval) {
            clearInterval(el._progressInterval);
        }
        
        el.classList.add('fade-out');
        el.addEventListener('animationend', () => el.remove(), { once: true });
    }
});

EventsOn('terminalBell', payload => {
    const p = Array.isArray(payload?.[0]) ? payload[0] : payload;
    if (p && p.sound && p.sound !== 'ding') {
        return;
    }

    if (!bellAudio) {
        return;
    }

    try {
        bellAudio.pause();
        bellAudio.currentTime = 0;
        const playPromise = bellAudio.play();
        if (playPromise && typeof playPromise.catch === 'function') {
            playPromise.catch(() => {});
        }
    } catch {
        // Ignore playback failures, e.g. if the webview blocks audio output.
    }
});

// Fullscreen image overlay
EventsOn('imageDisplayFullscreen', payload => {
    if (!payload || !payload.dataURL) return;

    showFullscreenImageOverlay({
        dataURL: payload.dataURL,
        sourceWidth: payload.sourceWidth,
        sourceHeight: payload.sourceHeight,
        onOpen: () => canvas.blur(),
        onClose: () => canvas.focus(),
    });
});
// ------------------------------------------------------------------

let resizeTimer = null;
window.addEventListener('resize', () => {
    fitCanvasToWindow();
    // Debounce so we don't spam Go on every pixel of a drag-resize.
    clearTimeout(resizeTimer);
    resizeTimer = setTimeout(() => {
        syncTerminalGridSize();
    }, 100);
});

window.addEventListener('ttyphoon-jupyter-tab-mode', (event) => {
    const enabled = event?.detail?.enabled === true;
    const active = event?.detail?.active !== false;
    const title = typeof event?.detail?.title === 'string' && event.detail.title.length > 0
        ? event.detail.title
        : 'Notes';

    jupyterTabEnabled = enabled;
    jupyterTabActive = enabled ? active : false;
    jupyterTabTitle = formatNotesTabTitle(title === 'Notes' ? '' : title);

    applyEmbeddedJupyterVisibility();
    syncAuxTerminalTabState();
    renderTerminalTabs(tabState);
    updateNotificationOffset();
});

window.addEventListener('notes-current-file', (event) => {
    const fileName = typeof event?.detail?.fileName === 'string' ? event.detail.fileName : '';
    jupyterTabTitle = formatNotesTabTitle(fileName);

    if (jupyterTabEnabled) {
        syncAuxTerminalTabState();
        renderTerminalTabs(tabState);
    }
});

window.addEventListener('ttyphoon-activate-notes-tab', () => {
    if (!jupyterTabEnabled) {
        return;
    }

    jupyterTabActive = true;
    applyEmbeddedJupyterVisibility();
    syncAuxTerminalTabState();
    renderTerminalTabs(tabState);
});

window.addEventListener('ttyphoon-activate-terminal-tab', () => {
    if (!jupyterTabEnabled) {
        return;
    }

    jupyterTabActive = false;
    applyEmbeddedJupyterVisibility();
    syncAuxTerminalTabState();
    renderTerminalTabs(tabState);
});

EventsOn('terminalActivateAuxTab', payload => {
    const p = Array.isArray(payload?.[0]) ? payload[0] : payload;
    const tabID = p?.id;
    if (tabID === '__tmux__') {
        jupyterTabActive = false;
        applyEmbeddedJupyterVisibility();
        syncAuxTerminalTabState();
        renderTerminalTabs(tabState);
        return;
    }

    if (tabID !== 'notes' || !jupyterTabEnabled) {
        return;
    }

    jupyterTabActive = true;
    applyEmbeddedJupyterVisibility();
    syncAuxTerminalTabState();
    renderTerminalTabs(tabState);
});
