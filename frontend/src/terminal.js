import { GetWindowStyle, SendIpc, TerminalCopyImageDataURL, TerminalGetTabs, TerminalRequestRedraw, TerminalResize, TerminalSelectWindow, TerminalSetGlyphSize } from '../wailsjs/go/main/WApp';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { wireKeyboardEvents, wireMouseEvents } from './events';
import { createFontController } from './font';
import { drawGauge } from './gauge';
import { drawBlockChrome } from './block_chrome';
import { initTerminalPopupMenu } from './popup_menu';
import { initInputBox } from './inputbox';
import { showFullscreenImageOverlay } from './fullscreen-image-overlay';

(document.getElementById('terminal-pane') || document.querySelector('#app')).innerHTML = `
    <div id="terminal-app">
        <div id="terminal-tabs" role="tablist" aria-label="tmux windows"></div>
        <div id="terminal-viewport">
            <canvas id="ttyphoon-terminal"></canvas>
        </div>
        <div id="terminal-notifications"></div>
    </div>
`;

const tabsEl = document.getElementById('terminal-tabs');
const canvas = document.getElementById('ttyphoon-terminal');
const ctx = canvas.getContext('2d');
const offscreen = document.createElement('canvas');
const offCtx = offscreen.getContext('2d');
const font = createFontController(offCtx);
let windowStyle;
let rafPending = false;
let tabState = [];
const imageCache = new Map();
const terminalStatusEl = document.getElementById('terminal-status');
const notifContainer = document.getElementById('terminal-notifications');
let cursorRects = [];
let cursorPulseRaf = 0;
let jupyterTabEnabled = false;
let jupyterTabActive = false;
let jupyterTabTitle = 'Notes';

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
            --terminal-fg: rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue});
            --terminal-accent: rgb(${result.colors.yellow.Red}, ${result.colors.yellow.Green}, ${result.colors.yellow.Blue});
            --terminal-accent-soft: rgba(${result.colors.yellow.Red}, ${result.colors.yellow.Green}, ${result.colors.yellow.Blue}, 0);
            --terminal-accent-ring: rgba(${result.colors.yellow.Red}, ${result.colors.yellow.Green}, ${result.colors.yellow.Blue}, 0);
            --terminal-selection: rgb(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue});
            --terminal-selection-20: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.2);
            --terminal-green: rgb(${result.colors.green.Red}, ${result.colors.green.Green}, ${result.colors.green.Blue});
            --terminal-green-20: rgba(${result.colors.green.Red}, ${result.colors.green.Green}, ${result.colors.green.Blue}, 0.2);
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
            border-bottom: 2px solid rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue});
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
            border: 2px solid transparent;
            background: transparent;
            color: rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue});
            padding: 6px 12px;
            cursor: pointer;
            white-space: nowrap;
        }

        #terminal-tabs button[aria-selected="true"] {
            background-color: rgba(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue}, 0.2);
            border-color: rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}) !important;
        }

        .terminal-tab {
            border-top-left-radius: 5px !important;
            border-top-right-radius: 5px !important;
            border: 2px solid !important;
            border-bottom: 0 !important;
            border-color: rgba(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}, 0.2) !important;
        }

        .terminal-tab:hover {
            border-color: rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue}) !important;
        }

        #terminal-viewport {
            position: relative;
            flex: 1;
            min-height: 0;
            overflow: hidden;
            box-sizing: border-box;
            border: 1px solid transparent;
            transition: border-color 120ms ease, box-shadow 120ms ease;
        }

        #terminal-pane[data-terminal-focused="true"] #terminal-viewport {
            border-color: var(--terminal-accent-soft);
        }

        #terminal-pane[data-terminal-focus-visible="true"] #terminal-viewport {
            box-shadow: inset 0 0 0 1px var(--terminal-accent-ring);
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
    canvas.width = pane ? pane.clientWidth : window.innerWidth;
    canvas.height = pane ? pane.clientHeight : window.innerHeight;
    offscreen.width = canvas.width;
    offscreen.height = canvas.height;
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
    const bg = windowStyle?.colors?.bg;
    if (bg) {
        ctx.fillStyle = `rgb(${bg.Red}, ${bg.Green}, ${bg.Blue})`;
        ctx.fillRect(0, 0, canvas.width, canvas.height);
    }

    // Draw terminal content
    ctx.drawImage(offscreen, 0, 0);

    // Apply dim overlay if terminal is not focused
    if (window.terminalFocusedState === false) {
        ctx.fillStyle = 'rgba(0, 0, 0, 0.2)';
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

    const xCell = Number.isFinite(cmd.x) ? cmd.x : 0;
    const yCell = Number.isFinite(cmd.y) ? cmd.y : 0;
    const widthCells = Number.isFinite(cmd.width) && cmd.width > 0 ? cmd.width : 1;

    const x = xCell * cellWidth;
    const y = yCell * cellHeight;
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

    if (cmd.char) {
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
            offCtx.strokeText(cmd.char, x, y);
                
            offCtx.shadowColor = outline;
            offCtx.shadowBlur = 6;
            offCtx.fillStyle = fill;
        }
        offCtx.fillText(cmd.char, x, y);
        if (cmd.searchResult) {
            offCtx.shadowColor = 'transparent';
            offCtx.shadowBlur = 0;
            offCtx.lineWidth = 0;
            offCtx.strokeStyle = 'transparent';
        }
    }

    if (cmd.underline) {
        const lineY = y + cellHeight - 2;
        offCtx.fillRect(x, lineY, width, 1);
    }

    if (cmd.strike) {
        const lineY = y + Math.floor(cellHeight / 2);
        offCtx.fillRect(x, lineY, width, 1);
    }
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

function drawFrame(cmd = null) {
    if (!offCtx) {
        return;
    }

    const { cellWidth, cellHeight } = font.getCellSize();

    const xCell = Number.isFinite(cmd?.x) ? cmd.x : 0;
    const yCell = Number.isFinite(cmd?.y) ? cmd.y : 0;
    const widthCells = Number.isFinite(cmd?.width) && cmd.width > 0 ? cmd.width : null;
    const heightCells = Number.isFinite(cmd?.height) && cmd.height > 0 ? cmd.height : null;

    const x = Math.max(0, Math.floor(xCell * cellWidth));
    const y = Math.max(0, Math.floor(yCell * cellHeight));

    let width = offscreen.width;
    let height = offscreen.height;

    if (cellWidth > 0 && cellHeight > 0) {
        const cols = widthCells ?? Math.floor(offscreen.width / cellWidth);
        const rows = heightCells ?? Math.floor(offscreen.height / cellHeight);

        width = Math.max(0, cols * cellWidth);
        height = Math.max(0, rows * cellHeight);
    }

    width = Math.min(width, Math.max(0, offscreen.width-x)) + 2;
    height = Math.min(height, Math.max(0, offscreen.height-y)) + 2;

    if (width <= 0 || height <= 0) {
        return;
    }

    const bg = windowStyle?.colors?.bg;
    if (bg) {
        offCtx.fillStyle = `rgb(${bg.Red}, ${bg.Green}, ${bg.Blue})`;
        offCtx.fillRect(x, y, width, height);
    } else {
        offCtx.clearRect(x, y, width, height);
    }
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
    const y = yCell * cellHeight;
    const width = widthCells * cellWidth;
    const height = heightCells * cellHeight;

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

EventsOn("terminalRedraw", ops => {
    if (rafPending) {
        return;
    }
    rafPending = true;

    const drawOps = Array.isArray(ops?.[0]) ? ops[0] : ops;

    if (!Array.isArray(drawOps) || drawOps.length === 0) {
        rafPending = false;
        return;
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
        if (cmd.op === 'gauge_h' || cmd.op === 'gauge_v') {
            drawGauge(offCtx, font.getCellSize, cmd);
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

    requestAnimationFrame(() => {
        paintTerminalCanvas();
        syncCursorLoopState();

        rafPending = false;
    });
});

EventsOn("terminalTabs", payload => {
    const tabs = Array.isArray(payload?.[0]) ? payload[0] : payload;
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
    font.applyConfiguredFontFromWindowStyle(windowStyle);
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

        TerminalRequestRedraw().catch(() => {});
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
