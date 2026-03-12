import { GetWindowStyle } from '../wailsjs/go/main/WApp';
import { EventsOn } from '../wailsjs/runtime/runtime';

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
let lastMouseCell = { x: 0, y: 0 };
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

function mouseButtonToGo(button) {
    switch (button) {
    case 0:
        return 1;
    case 1:
        return 2;
    case 2:
        return 3;
    case 3:
        return 4;
    case 4:
        return 5;
    default:
        return 1;
    }
}

function eventToCell(event) {
    const rect = canvas.getBoundingClientRect();
    const x = Math.floor((event.clientX - rect.left) / cellWidth);
    const y = Math.floor((event.clientY - rect.top) / cellHeight);
    return { x, y };
}

function wireMouseEvents() {
    canvas.addEventListener('contextmenu', (event) => {
        event.preventDefault();
    });

    canvas.addEventListener('mousedown', (event) => {
        const pos = eventToCell(event);
        lastMouseCell = pos;
        window['go']['main']['WApp']['TerminalMouseButton'](
            pos.x,
            pos.y,
            mouseButtonToGo(event.button),
            event.detail || 1,
            true,
        ).catch(() => {});
    });

    canvas.addEventListener('mouseup', (event) => {
        const pos = eventToCell(event);
        lastMouseCell = pos;
        window['go']['main']['WApp']['TerminalMouseButton'](
            pos.x,
            pos.y,
            mouseButtonToGo(event.button),
            event.detail || 1,
            false,
        ).catch(() => {});
    });

    canvas.addEventListener('mousemove', (event) => {
        const pos = eventToCell(event);
        const relX = pos.x - lastMouseCell.x;
        const relY = pos.y - lastMouseCell.y;
        lastMouseCell = pos;
        window['go']['main']['WApp']['TerminalMouseMotion'](
            pos.x,
            pos.y,
            relX,
            relY,
            event.buttons,
        ).catch(() => {});
    });

    canvas.addEventListener('wheel', (event) => {
        event.preventDefault();
        const pos = eventToCell(event);
        const moveX = Math.sign(event.deltaX);
        const moveY = -Math.sign(event.deltaY);
        window['go']['main']['WApp']['TerminalMouseWheel'](
            pos.x,
            pos.y,
            moveX,
            moveY,
        ).catch(() => {});
    }, { passive: false });
}

function scheduleRaf() {
    if (rafPending) {
        return;
    }
    rafPending = true;
    requestAnimationFrame(() => {
        ctx.drawImage(offscreen, 0, 0);
        rafPending = false;
    });
}

EventsOn("terminalRedraw", ops => {
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
        }
    }

    // Schedule a blit to the visible canvas on the next display frame.
    // Multiple events arriving between frames all draw to offscreen;
    // only one blit happens per vsync, eliminating flicker.
    scheduleRaf();
});

GetWindowStyle().then((result) => {
    windowStyle = result;
    document.body.style.margin = '0';
    document.body.style.overflow = 'hidden';
    document.body.style.backgroundColor = `rgb(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue})`;
    applyConfiguredFontFromWindowStyle();
    fitCanvasToWindow();
    loadGlyphSizeFromGo().then(() => {
        drawFrame();
        wireMouseEvents();
        window['go']['main']['WApp']['TerminalRequestRedraw']().catch(() => {});
    });
});

window.addEventListener('resize', () => {
    fitCanvasToWindow();
    drawFrame();
});
