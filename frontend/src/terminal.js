import { GetWindowStyle, TerminalGetTabs, TerminalRequestRedraw, TerminalResize, TerminalSelectWindow, TerminalSetGlyphSize } from '../wailsjs/go/main/WApp';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { wireKeyboardEvents, wireMouseEvents } from './events';
import { createFontController } from './font';
import { drawGauge } from './gauge';
import { drawBlockChrome } from './block_chrome';
import { initTerminalPopupMenu } from './popup_menu';
import { initInputBox } from './inputbox';

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

    for (const cmd of drawOps) {
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

EventsOn("terminalStatusBarText", payload => {
    if (!terminalStatusEl) {
        return;
    }

    const text = Array.isArray(payload?.[0]) ? payload[0] : payload;
    terminalStatusEl.textContent = typeof text === 'string' ? text : '';
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

const notifContainer = document.getElementById('terminal-notifications');

const _notifyBg = ['#316db0', '#99c0d3', '#f2b71f', '#de333b', '#316db0', '#74953c'];
const _notifyFg = ['#000000', '#000000', '#000000', '#000000', '#000000', '#000000'];

EventsOn('terminalNotification', payload => {
    const p = Array.isArray(payload?.[0]) ? payload[0] : payload;
    if (!p || !notifContainer) return;

    // update message if already shown (e.g. SetMessage)
    const existing = notifContainer.querySelector(`[data-notif-id="${p.id}"]`);
    if (existing) {
        existing.querySelector('.notif-msg').textContent = p.message;
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
    el.querySelector('.notif-msg').textContent = p.message;

    if (!p.sticky) {
        el.addEventListener('click', () => el.remove());
    }

    notifContainer.appendChild(el);
});

EventsOn('terminalNotificationClose', payload => {
    const id = Array.isArray(payload) ? payload[0] : payload;
    if (!notifContainer) return;
    const el = notifContainer.querySelector(`[data-notif-id="${id}"]`);
    if (el) el.remove();
});

// Fullscreen image overlay
EventsOn('imageDisplayFullscreen', payload => {
    if (!payload || !payload.dataURL) return;

    const existingOverlay = document.getElementById('fullscreen-image-overlay');
    if (existingOverlay) {
        existingOverlay.remove();
    }

    const overlay = document.createElement('div');
    overlay.id = 'fullscreen-image-overlay';
    overlay.style.cssText = `
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.95);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 999999;
        overflow: auto;
        padding: 20px;
        box-sizing: border-box;
    `;

    const container = document.createElement('div');
    container.style.cssText = `
        display: flex;
        align-items: center;
        justify-content: center;
        max-width: 100%;
        max-height: 100%;
    `;

    const img = document.createElement('img');
    img.src = payload.dataURL;
    img.style.cssText = `
        max-width: 100%;
        max-height: 100%;
        object-fit: contain;
        box-shadow: 0 0 30px rgba(255, 255, 255, 0.3);
        border-radius: 8px;
    `;

    // Info text
    const info = document.createElement('div');
    info.style.cssText = `
        position: absolute;
        bottom: 20px;
        right: 20px;
        color: rgba(255, 255, 255, 0.7);
        font-size: 12px;
        font-family: monospace;
        background: rgba(0, 0, 0, 0.5);
        padding: 8px 12px;
        border-radius: 4px;
    `;
    info.textContent = `${payload.sourceWidth}×${payload.sourceHeight} | Press ESC to close`;

    container.appendChild(img);
    overlay.appendChild(container);
    overlay.appendChild(info);
    document.body.appendChild(overlay);

    const closeOverlay = () => {
        document.removeEventListener('keydown', handleKey, true);
        overlay.removeEventListener('click', handleClick);
        overlay.remove();
        canvas.focus();
    };

    // Handle keyboard: ESC to close, capture all keys to prevent terminal input
    const handleKey = (e) => {
        // Always consume keystrokes while overlay is open so nothing reaches terminal.
        e.stopPropagation();
        e.preventDefault();

        if (e.key === 'Escape') {
            closeOverlay();
            return;
        }
    };

    // Handle click outside to close
    const handleClick = (e) => {
        if (e.target === overlay) {
            closeOverlay();
        }
    };

    document.addEventListener('keydown', handleKey, true);
    overlay.addEventListener('click', handleClick);
    canvas.blur();
});
// ------------------------------------------------------------------

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
