import './style.css';
import './app.css';
import { ScreenGetAll, WindowGetPosition, WindowGetSize, WindowSetPosition, WindowSetSize } from '../wailsjs/runtime/runtime';

// Remove any body margin/padding immediately so there is no layout flash.
document.body.style.margin = '0';
document.body.style.padding = '0';
document.body.style.overflow = 'hidden';

const app = document.getElementById('app') || document.body;

// The split layout: notes on the left half, terminal on the right half.
// Both panes are created synchronously here.  notes.js and terminal.js are
// loaded as dynamic imports below, so their module bodies run *after* this
// synchronous code — they will find #notes-pane and #terminal-pane in the DOM.
app.style.cssText = [
    'display:flex',
    'width:100vw',
    'height:100vh',
    'margin:0',
    'padding:0',
    'overflow:hidden',
    'box-sizing:border-box',
].join(';');

const notesPane = document.createElement('div');
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

const splitHandle = document.createElement('div');
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

const splitToggle = document.createElement('button');
splitToggle.type = 'button';
splitToggle.id = 'notes-terminal-toggle';
splitToggle.setAttribute('aria-label', 'Collapse notes pane');
splitToggle.title = 'Collapse notes pane';
splitToggle.textContent = '◀';
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
].join(';');
splitHandle.appendChild(splitToggle);

const terminalPane = document.createElement('div');
terminalPane.id = 'terminal-pane';
terminalPane.style.cssText = [
    'flex:1',
    'height:100%',
    'overflow:hidden',
    'position:relative',
    'min-width:0',
].join(';');

app.appendChild(notesPane);
app.appendChild(splitHandle);
app.appendChild(terminalPane);

let isDraggingSplit = false;
let notesCollapsed = false;
let lastNotesWidthPercent = 50;
let lastCollapseDeltaPx = 0;

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

        void adjustWindowFrameBy(-lastCollapseDeltaPx);
    } else {
        notesCollapsed = false;
        const minPercent = (Math.max(240, Math.round(rect.width * 0.15)) / rect.width) * 100;
        const maxPercent = ((rect.width - Math.max(240, Math.round(rect.width * 0.15)) - splitHandle.offsetWidth) / rect.width) * 100;
        const restored = Math.min(Math.max(lastNotesWidthPercent, minPercent), maxPercent);

        notesPane.style.width = `${restored}%`;
        notesPane.style.borderRight = '1px solid rgba(255,255,255,0.12)';
        splitToggle.textContent = '◀';
        splitToggle.setAttribute('aria-label', 'Collapse notes pane');
        splitToggle.title = 'Collapse notes pane';

        if (lastCollapseDeltaPx > 0) {
            void adjustWindowFrameBy(lastCollapseDeltaPx);
        }
    }

    //window.dispatchEvent(new Event('resize'));
}

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
});

splitToggle.addEventListener('click', (event) => {
    event.preventDefault();
    event.stopPropagation();
    toggleNotesPaneCollapsed();
});

// Dynamic imports — the promises resolve asynchronously, but the resolution
// microtask queue starts only after this synchronous module body finishes.
// By then #notes-pane and #terminal-pane exist, so each module finds its root.
import('./notes.js');
import('./terminal.js');
