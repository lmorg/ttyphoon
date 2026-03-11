import { GetWindowStyle, GetTerminalDrawOps } from '../wailsjs/go/main/WApp';

document.querySelector('#app').innerHTML = `
    <canvas id="ttyphoon-terminal"></canvas>
`;

const canvas = document.getElementById('ttyphoon-terminal');
const ctx = canvas.getContext('2d');
let windowStyle;
let cellWidth = 10;
let cellHeight = 20;
let fontSize = 18;
let fontFamily = 'monospace';
let glyphSizeCached = false;
let lastMouseCell = { x: 0, y: 0 };

function fitCanvasToWindow() {
    canvas.width = window.innerWidth;
    canvas.height = window.innerHeight;
}

function applyConfiguredFontFromWindowStyle() {
    const parsed = parseInt(windowStyle?.fontSize, 10);
    if (!Number.isNaN(parsed) && parsed > 0) {
        fontSize = parsed;
    }

    if (windowStyle?.fontFamily) {
        fontFamily = windowStyle.fontFamily;
    }

    if (ctx) {
        ctx.font = `${fontSize}px ${fontFamily}`;
    }
}

function configureFontMetricsFallback() {
    if (!ctx) {
        return;
    }

    applyConfiguredFontFromWindowStyle();

    ctx.font = `${fontSize}px ${fontFamily}`;
    const metrics = ctx.measureText('M');
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
    if (!ctx) {
        return;
    }

    const xCell = Number.isFinite(cmd.x) ? cmd.x : 0;
    const yCell = Number.isFinite(cmd.y) ? cmd.y : 0;
    const widthCells = Number.isFinite(cmd.width) && cmd.width > 0 ? cmd.width : 1;

    const x = xCell * cellWidth;
    const y = yCell * cellHeight;
    const width = widthCells * cellWidth;

    if (cmd.bg) {
        ctx.fillStyle = `rgb(${cmd.bg.Red}, ${cmd.bg.Green}, ${cmd.bg.Blue})`;
        ctx.fillRect(x, y, width, cellHeight);
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
    ctx.font = fontParts.join(' ');
    ctx.textBaseline = 'top';

    if (cmd.fg) {
        ctx.fillStyle = `rgb(${cmd.fg.Red}, ${cmd.fg.Green}, ${cmd.fg.Blue})`;
    } else {
        ctx.fillStyle = '#ffffff';
    }

    if (cmd.char) {
        ctx.fillText(cmd.char, x, y);
    }

    if (cmd.underline) {
        const lineY = y + cellHeight - 2;
        ctx.fillRect(x, lineY, width, 1);
    }

    if (cmd.strike) {
        const lineY = y + Math.floor(cellHeight / 2);
        ctx.fillRect(x, lineY, width, 1);
    }
}

function drawFrame() {
    if (!ctx) {
        return;
    }

    const bg = windowStyle?.colors?.bg;
    if (bg) {
        ctx.fillStyle = `rgb(${bg.Red}, ${bg.Green}, ${bg.Blue})`;
        ctx.fillRect(0, 0, canvas.width, canvas.height);
    } else {
        ctx.clearRect(0, 0, canvas.width, canvas.height);
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

async function flushDrawOps() {
    const ops = await GetTerminalDrawOps();
    if (!Array.isArray(ops) || ops.length === 0) {
        return;
    }

    for (const cmd of ops) {
        if (cmd.op === 'frame') {
            drawFrame();
            continue;
        }
        if (cmd.op === 'cell') {
            drawCell(cmd);
        }
    }
}

function startRendererLoop() {
    setInterval(() => {
        flushDrawOps().catch(() => {});
    }, 16);
}

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
        startRendererLoop();
    });
});

window.addEventListener('resize', () => {
    fitCanvasToWindow();
    drawFrame();
});
