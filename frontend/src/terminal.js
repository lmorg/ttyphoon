import { GetWindowStyle } from '../wailsjs/go/main/WApp';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { wireKeyboardEvents, wireMouseEvents } from './events';

document.querySelector('#app').innerHTML = `
    <canvas id="ttyphoon-terminal"></canvas>
`;

const canvas = document.getElementById('ttyphoon-terminal');
const ctx = canvas.getContext('2d');
const offscreen = document.createElement('canvas');
//const offscreen = document.getElementById('ttyphoon-terminal-buf');
const offCtx = offscreen.getContext('2d');
let windowStyle;
let cellWidth = 10;
let cellHeight = 20;
let fontSize = 18;
let fontFamily = 'monospace';
let glyphSizeCached = false;
let rafPending = false;

function fitCanvasToWindow() {
    canvas.width = window.innerWidth;
    canvas.height = window.innerHeight;
    offscreen.width = canvas.width;
    offscreen.height = canvas.height;
}

function applyConfiguredFontFromWindowStyle() {
    const parsed = parseInt(windowStyle?.fontSize, 10);
    if (!Number.isNaN(parsed) && parsed > 0) {
        fontSize = parsed;
    }

    if (windowStyle?.fontFamily) {
        fontFamily = windowStyle.fontFamily;
    }

    if (offCtx) {
        offCtx.font = `${fontSize}px ${fontFamily}`;
    }
}

function configureFontMetricsFallback() {
    if (!offCtx) {
        return;
    }

    applyConfiguredFontFromWindowStyle();

    offCtx.font = `${fontSize}px ${fontFamily}`;
    const metrics = offCtx.measureText('M');
    cellWidth = Math.ceil(metrics.width || fontSize * 0.6);
    cellHeight = Math.ceil((metrics.fontBoundingBoxAscent || fontSize) + (metrics.fontBoundingBoxDescent || fontSize * 0.2));
}

async function loadGlyphSizeFromGo() {
    if (glyphSizeCached) {
        return;
    }

    try {
        const glyph = await window['go']['main']['WApp']['GetTerminalGlyphSize']();
        if (glyph && glyph.X > 0 && glyph.Y > 0) {
            cellWidth = glyph.X;
            cellHeight = glyph.Y;
            glyphSizeCached = true;
            return;
        }
    } catch {
        // fallback below
    }

    configureFontMetricsFallback();
    glyphSizeCached = true;
}

function drawCell(cmd) {
    if (!offCtx) {
        return;
    }

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

    const fontParts = [];
    if (cmd.italic) {
        fontParts.push('italic');
    }
    if (cmd.bold) {
        fontParts.push('bold');
    }
    fontParts.push(`${fontSize}px`);
    fontParts.push(fontFamily);
    offCtx.font = fontParts.join(' ');
    offCtx.textBaseline = 'top';

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

function drawFrame() {
    if (!offCtx) {
        return;
    }

    const bg = windowStyle?.colors?.bg;
    if (bg) {
        offCtx.fillStyle = `rgb(${bg.Red}, ${bg.Green}, ${bg.Blue})`;
        offCtx.fillRect(0, 0, offscreen.width, offscreen.height);
    } else {
        offCtx.clearRect(0, 0, offscreen.width, offscreen.height);
    }
}

function drawGauge(cmd) {
    if (!offCtx || !cmd?.fg || !Number.isFinite(cmd.max) || cmd.max <= 0) {
        return;
    }

    const x = (Number.isFinite(cmd.x) ? cmd.x : 0) * cellWidth;
    const y = (Number.isFinite(cmd.y) ? cmd.y : 0) * cellHeight;
    const ratio = Math.max(0, Math.min(1, (Number.isFinite(cmd.value) ? cmd.value : 0) / cmd.max));

    const base = `rgb(${cmd.fg.Red}, ${cmd.fg.Green}, ${cmd.fg.Blue})`;

    if (cmd.op === 'gauge_h') {
        const widthCells = Number.isFinite(cmd.width) && cmd.width > 0 ? cmd.width : 1;
        const fullW = widthCells * cellWidth;

        offCtx.globalAlpha = 0.13;
        offCtx.fillStyle = base;
        offCtx.fillRect(x, y, fullW, cellHeight);

        offCtx.globalAlpha = 0.75;
        offCtx.fillRect(x, y, Math.floor(fullW * ratio), cellHeight);
        offCtx.globalAlpha = 1;
        return;
    }

    if (cmd.op === 'gauge_v') {
        const heightCells = Number.isFinite(cmd.height) && cmd.height > 0 ? cmd.height : 1;
        const fullH = heightCells * cellHeight;

        offCtx.globalAlpha = 0.13;
        offCtx.fillStyle = base;
        offCtx.fillRect(x, y, cellWidth, fullH);

        const fillH = Math.floor(fullH * ratio);
        offCtx.globalAlpha = 0.75;
        offCtx.fillRect(x, y, cellWidth, fillH);
        offCtx.globalAlpha = 1;
    }
}

function drawBlockChrome(cmd) {
    if (!offCtx || !cmd?.fg) {
        return;
    }

    const xCell = Number.isFinite(cmd.x) ? cmd.x : 0;
    const yCell = Number.isFinite(cmd.y) ? cmd.y : 0;
    const heightCells = Number.isFinite(cmd.height) && cmd.height > 0 ? cmd.height : 1;

    const x = xCell * cellWidth;
    const y = yCell * cellHeight;
    const h = heightCells * cellHeight;
    const barWidth = Math.max(2, Math.floor(cellWidth * (cmd.folded ? 0.5 : 0.25)));

    offCtx.fillStyle = `rgb(${cmd.fg.Red}, ${cmd.fg.Green}, ${cmd.fg.Blue})`;
    offCtx.globalAlpha = 0.75;
    offCtx.fillRect(x, y, barWidth, h);

    if (!cmd.folded && Number.isFinite(cmd.endX) && cmd.endX >= xCell) {
        const lineY = y + h;
        const lineEndX = ((cmd.endX + 1) * cellWidth) - 1;
        offCtx.fillRect(x, lineY, Math.max(1, lineEndX - x + 1), 1);
    }

    offCtx.globalAlpha = 1;
}

EventsOn("terminalRedraw", ops => {
    if (rafPending) {
        return;
    }
   rafPending = true;

    const drawOps = Array.isArray(ops?.[0]) ? ops[0] : ops;

    if (!Array.isArray(drawOps) || drawOps.length === 0) {
        return;
    }

    for (const cmd of drawOps) {
        if (cmd.op === 'frame') {
            drawFrame();
            continue;
        }
        if (cmd.op === 'cell') {
            drawCell(cmd);
            continue;
        }
        if (cmd.op === 'gauge_h' || cmd.op === 'gauge_v') {
            drawGauge(cmd);
            continue;
        }
        if (cmd.op === 'block_chrome') {
            drawBlockChrome(cmd);
        }
    }

    requestAnimationFrame(() => {
        ctx.drawImage(offscreen, 0, 0);
        rafPending = false;
    });
});

GetWindowStyle().then((result) => {
    windowStyle = result;
    document.body.style.margin = '0';
    document.body.style.overflow = 'hidden';
    document.body.style.backgroundColor = `rgb(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue})`;
    applyConfiguredFontFromWindowStyle();
    fitCanvasToWindow();
    loadGlyphSizeFromGo().then(() => {
        //drawFrame();
        wireKeyboardEvents(canvas);
        wireMouseEvents(canvas, () => ({ cellWidth, cellHeight }));
        canvas.focus();
        window['go']['main']['WApp']['TerminalRequestRedraw']().catch(() => {});
    });
});

window.addEventListener('resize', () => {
    fitCanvasToWindow();
    //drawFrame();
});
