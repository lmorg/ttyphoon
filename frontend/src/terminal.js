import { GetWindowStyle, TerminalGetTabs, TerminalRequestRedraw, TerminalResize, TerminalSelectWindow } from '../wailsjs/go/main/WApp';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { wireKeyboardEvents, wireMouseEvents } from './events';
import { createFontController } from './font';
import { drawGauge } from './gauge';
import { drawBlockChrome } from './block_chrome';
import { initTerminalPopupMenu } from './popup_menu';

(document.getElementById('terminal-pane') || document.querySelector('#app')).innerHTML = `
    <div id="terminal-app">
        <div id="terminal-tabs" role="tablist" aria-label="tmux windows"></div>
        <div id="terminal-viewport">
            <canvas id="ttyphoon-terminal"></canvas>
        </div>
    </div>
`;

const tabsEl = document.getElementById('terminal-tabs');
const canvas = document.getElementById('ttyphoon-terminal');
const ctx = canvas.getContext('2d');
const offscreen = document.createElement('canvas');
//const offscreen = document.getElementById('ttyphoon-terminal-buf');
const offCtx = offscreen.getContext('2d');
const font = createFontController(offCtx);
let windowStyle;
let rafPending = false;
let tabState = [];

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
        button.setAttribute('aria-selected', tab.active ? 'true' : 'false');
        button.dataset.windowId = tab.id || '';
        button.textContent = tab.name || tab.id || 'window';
        button.title = tab.id || button.textContent;

        button.addEventListener('click', () => {
            if (!tab.id) {
                return;
            }
            TerminalSelectWindow(tab.id).catch(() => {});
        });

        tabsEl.appendChild(button);
    }

    tabsEl.style.display = tabState.length > 0 ? 'flex' : 'none';
}

function applyTerminalStyles(result) {
    const existing = document.getElementById('terminal-theme');
    if (existing) {
        existing.remove();
    }

    const style = document.createElement('style');
    style.id = 'terminal-theme';
    style.textContent = `
        #terminal-app {
            display: flex;
            flex-direction: column;
            height: 100%;
            width: 100%;
            min-height: 0;
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
            scrollbar-width: thin;
            box-sizing: border-box;
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
        offCtx.fillText(cmd.char, x, y);
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

function getCommandBoundsCells(cmd) {
    const x = Number.isFinite(cmd?.x) ? cmd.x : 0;
    const y = Number.isFinite(cmd?.y) ? cmd.y : 0;

    switch (cmd?.op) {
    case 'cell': {
        const width = Number.isFinite(cmd.width) && cmd.width > 0 ? cmd.width : 1;
        return { x, y, width, height: 1 };
    }

    case 'gauge_h': {
        const width = Number.isFinite(cmd.width) && cmd.width > 0 ? cmd.width : 1;
        return { x, y, width, height: 1 };
    }

    case 'gauge_v': {
        const height = Number.isFinite(cmd.height) && cmd.height > 0 ? cmd.height : 1;
        return { x, y, width: 1, height };
    }

    case 'block_chrome': {
        const height = Number.isFinite(cmd.height) && cmd.height > 0 ? cmd.height : 1;
        let width = 1;
        if (!cmd.folded && Number.isFinite(cmd.endX) && cmd.endX >= x) {
            width = (cmd.endX - x) + 1;
        }
        return { x, y, width, height: height + 1 };
    }

    case 'highlight_rect':
    case 'rect_colour':
    case 'tile_overlay': {
        const width = Number.isFinite(cmd.width) && cmd.width > 0 ? cmd.width : 0;
        const height = Number.isFinite(cmd.height) && cmd.height > 0 ? cmd.height : 0;
        if (width <= 0 || height <= 0) {
            return null;
        }
        return { x, y, width, height };
    }

    default:
        return null;
    }
}

function getDrawOpsBoundsCells(drawOps) {
    let minX = Number.POSITIVE_INFINITY;
    let minY = Number.POSITIVE_INFINITY;
    let maxX = Number.NEGATIVE_INFINITY;
    let maxY = Number.NEGATIVE_INFINITY;

    for (const cmd of drawOps) {
        if (cmd?.op === 'frame') {
            continue;
        }

        const bounds = getCommandBoundsCells(cmd);
        if (!bounds) {
            continue;
        }

        minX = Math.min(minX, bounds.x);
        minY = Math.min(minY, bounds.y);
        maxX = Math.max(maxX, bounds.x + bounds.width);
        maxY = Math.max(maxY, bounds.y + bounds.height);
    }

    if (!Number.isFinite(minX) || !Number.isFinite(minY) || !Number.isFinite(maxX) || !Number.isFinite(maxY)) {
        return null;
    }

    return {
        x: Math.max(0, minX),
        y: Math.max(0, minY),
        width: Math.max(0, maxX - Math.max(0, minX)),
        height: Math.max(0, maxY - Math.max(0, minY)),
    };
}

function drawFrame(boundsCells = null) {
    if (!offCtx) {
        return;
    }

    const { cellWidth, cellHeight } = font.getCellSize();

    let x = 0;
    let y = 0;
    let width = offscreen.width;
    let height = offscreen.height;

    if (boundsCells && cellWidth > 0 && cellHeight > 0) {
        x = Math.floor(boundsCells.x * cellWidth);
        y = Math.floor(boundsCells.y * cellHeight);
        width = Math.ceil(boundsCells.width * cellWidth);
        height = Math.ceil(boundsCells.height * cellHeight);

        // Cursor/highlight strokes and antialiasing can paint a couple of
        // pixels outside the nominal cell box. Pad clears to avoid ghosting.
        const bleedPx = 2;
        x -= bleedPx;
        y -= bleedPx;
        width += bleedPx * 2;
        height += bleedPx * 2;

        if (x < 0) {
            width += x;
            x = 0;
        }
        if (y < 0) {
            height += y;
            y = 0;
        }

        width = Math.min(width, offscreen.width - x);
        height = Math.min(height, offscreen.height - y);

        if (width <= 0 || height <= 0) {
            return;
        }
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
        offCtx.fillStyle = `rgb(${cmd.bg.Red}, ${cmd.bg.Green}, ${cmd.bg.Blue})`;
        offCtx.fillRect(x + 1, y + 1, Math.max(0, width - 2), Math.max(0, height - 2));
        offCtx.restore();
    }
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

    const frameBounds = getDrawOpsBoundsCells(drawOps);

    for (const cmd of drawOps) {
        if (cmd.op === 'frame') {
            drawFrame(frameBounds);
            continue;
        }
        if (cmd.op === 'cell') {
            drawCell(cmd);
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
            drawTileOverlay(cmd);
            continue;
        }
        if (cmd.op === 'highlight_rect' || cmd.op === 'rect_colour') {
            drawHighlightRect(cmd);
        }
    }

    requestAnimationFrame(() => {
        ctx.drawImage(offscreen, 0, 0);
        rafPending = false;
    });
});

EventsOn("terminalTabs", payload => {
    const tabs = Array.isArray(payload?.[0]) ? payload[0] : payload;
    renderTerminalTabs(tabs);
    fitCanvasToWindow();
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

let resizeTimer = null;
window.addEventListener('resize', () => {
    fitCanvasToWindow();
    // Debounce so we don't spam Go on every pixel of a drag-resize.
    clearTimeout(resizeTimer);
    resizeTimer = setTimeout(() => {
        const { cellWidth, cellHeight } = font.getCellSize();
        if (cellWidth > 0 && cellHeight > 0) {
            const cols = Math.floor(canvas.width / cellWidth) - 1;
            const rows = Math.floor(canvas.height / cellHeight);
            if (cols > 0 && rows > 0) {
                TerminalResize(cols, rows).catch(() => {});
            }
        }
    }, 100);
});
