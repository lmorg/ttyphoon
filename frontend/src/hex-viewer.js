const BYTES_PER_ROW = 16;
const DEFAULT_ROW_HEIGHT = 22;
const OVERSCAN_ROWS = 40;
const stateByContainer = new WeakMap();

function getRowHeight(fontSize, adjustCellHeight) {
    const parsedFontSize = Number(fontSize);
    const parsedAdjust = Number(adjustCellHeight);
    if (!Number.isFinite(parsedFontSize) || parsedFontSize <= 0) {
        return DEFAULT_ROW_HEIGHT;
    }

    const delta = Number.isFinite(parsedAdjust) ? parsedAdjust : 0;
    return Math.max(1, Math.round(parsedFontSize + delta));
}

function escapeHtml(text) {
    return String(text || '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

function byteClass(byte) {
    const normalized = Number(byte) & 0xff;
    if (normalized === 0x00) {
        return 'notes-hex-byte-00';
    }
    if (normalized === 0xff) {
        return 'notes-hex-byte-FF';
    }

    return `notes-hex-byte-${normalized.toString(16).toUpperCase().charAt(0)}`;
}

function isPrintableAscii(value) {
    return value >= 0x20 && value <= 0x7e;
}

function parseHexOffset(text) {
    const raw = String(text || '').trim().toLowerCase().replace(/^0x/, '');
    if (!raw || !/^[0-9a-f]+$/.test(raw)) {
        return null;
    }

    return Number.parseInt(raw, 16);
}

function decodeBase64ToBytes(base64Data) {
    const encoded = String(base64Data || '').trim();
    if (!encoded) {
        return new Uint8Array(0);
    }

    const binary = atob(encoded);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i);
    }
    return bytes;
}

function renderHeaderRow() {
    const labels = ['0','1','2','3','4','5','6','7','8','9','A','B','C','D','E','F'];
    const hexCells = [];
    for (let i = 0; i < BYTES_PER_ROW; i++) {
        if (i === 8) {
            hexCells.push('<span class="notes-hex-gap"> </span>');
        }
        hexCells.push(`<span class="notes-hex-byte notes-hex-header-col"> ${labels[i]}</span>`);
        if (i !== BYTES_PER_ROW - 1) {
            hexCells.push(' ');
        }
    }
    return [
        '<div class="notes-hex-row notes-hex-header" aria-hidden="true">',
        '<span class="notes-hex-offset">  Offset</span>  ',
        `<span class="notes-hex-bytes">${hexCells.join('')}</span>  `,
        '<span class="notes-hex-ascii-wrap">|     ASCII      |</span>',
        '</div>',
    ].join('');
}

function renderRow(bytes, rowIndex) {
    const start = rowIndex * BYTES_PER_ROW;
    const end = Math.min(bytes.length, start + BYTES_PER_ROW);

    const hexCells = [];
    const asciiCells = [];

    for (let i = 0; i < BYTES_PER_ROW; i++) {
        if (i === 8) {
            hexCells.push('<span class="notes-hex-gap"> </span>');
        }

        const byteIndex = start + i;
        if (byteIndex < end) {
            const value = bytes[byteIndex];
            const hex = value.toString(16).padStart(2, '0').toUpperCase();
            hexCells.push(`<span class="notes-hex-byte ${byteClass(value)}">${hex}</span>`);

            if (isPrintableAscii(value)) {
                asciiCells.push(`<span class="notes-hex-ascii notes-hex-ascii-printable">${escapeHtml(String.fromCharCode(value))}</span>`);
            } else {
                asciiCells.push('<span class="notes-hex-ascii notes-hex-ascii-non-graphic">.</span>');
            }
        } else {
            hexCells.push('<span class="notes-hex-byte notes-hex-byte-empty">  </span>');
            asciiCells.push('<span class="notes-hex-ascii notes-hex-byte-empty"> </span>');
        }

        if (i !== BYTES_PER_ROW - 1) {
            hexCells.push(' ');
        }
    }

    const offset = start.toString(16).padStart(8, '0');
    return [
        '<div class="notes-hex-row">',
        `<span class="notes-hex-offset">${offset}</span>  `,
        `<span class="notes-hex-bytes">${hexCells.join('')}</span>  `,
        `<span class="notes-hex-ascii-wrap">|${asciiCells.join('')}|</span>`,
        '</div>',
    ].join('');
}

export function renderHexDump(container, base64Data, options = {}) {
    if (!container) {
        return;
    }

    const previous = stateByContainer.get(container);
    if (previous && typeof previous.cleanup === 'function') {
        previous.cleanup();
    }

    const bytes = decodeBase64ToBytes(base64Data);
    const totalRows = Math.ceil(bytes.length / BYTES_PER_ROW);
    const rowHeight = getRowHeight(options.fontSize, options.adjustCellHeight);

    container.innerHTML = `
        <div class="notes-hex-toolbar">
            <span class="notes-hex-toolbar-label">Jump to offset</span>
            <input class="notes-hex-offset-input" type="text" placeholder="0x0" autocomplete="off" spellcheck="false" />
            <button class="notes-hex-offset-go" type="button">Go</button>
            <span class="notes-hex-size">${bytes.length} bytes</span>
        </div>
        <div class="notes-hex-viewport">
            ${renderHeaderRow()}
            <pre class="notes-hex-dump"></pre>
        </div>
    `;

    const viewport = container.querySelector('.notes-hex-viewport');
    const dump = container.querySelector('.notes-hex-dump');
    const offsetInput = container.querySelector('.notes-hex-offset-input');
    const goButton = container.querySelector('.notes-hex-offset-go');
    if (!viewport || !dump || !offsetInput || !goButton) {
        return;
    }

    let lastStart = -1;
    let lastEnd = -1;
    const renderVisibleRows = () => {
        const scrollTop = viewport.scrollTop;
        const viewportHeight = viewport.clientHeight || 0;
        const startRow = Math.max(0, Math.floor(scrollTop / rowHeight) - OVERSCAN_ROWS);
        const visibleRows = Math.ceil(viewportHeight / rowHeight) + (OVERSCAN_ROWS * 2);
        const endRow = Math.min(totalRows, startRow + visibleRows);

        if (startRow === lastStart && endRow === lastEnd) {
            return;
        }
        lastStart = startRow;
        lastEnd = endRow;

        const rows = [];
        for (let row = startRow; row < endRow; row++) {
            rows.push(renderRow(bytes, row));
        }

        dump.style.paddingTop = `${startRow * rowHeight}px`;
        dump.style.paddingBottom = `${Math.max(0, totalRows - endRow) * rowHeight}px`;
        dump.innerHTML = rows.join('');
    };

    const onScroll = () => {
        renderVisibleRows();
    };
    viewport.addEventListener('scroll', onScroll);

    const onResize = () => {
        renderVisibleRows();
    };
    window.addEventListener('resize', onResize);

    const jumpToOffset = () => {
        const parsed = parseHexOffset(offsetInput.value);
        if (parsed === null || bytes.length === 0) {
            return;
        }

        const clamped = Math.max(0, Math.min(parsed, bytes.length - 1));
        const row = Math.floor(clamped / BYTES_PER_ROW);
        viewport.scrollTop = row * rowHeight;
        offsetInput.value = `0x${clamped.toString(16)}`;
        renderVisibleRows();
    };

    const onInputKeyDown = (event) => {
        if (event.key === 'Enter') {
            event.preventDefault();
            jumpToOffset();
        }
    };

    const onGoClick = () => {
        jumpToOffset();
    };

    offsetInput.addEventListener('keydown', onInputKeyDown);
    goButton.addEventListener('click', onGoClick);

    renderVisibleRows();

    stateByContainer.set(container, {
        cleanup: () => {
            viewport.removeEventListener('scroll', onScroll);
            window.removeEventListener('resize', onResize);
            offsetInput.removeEventListener('keydown', onInputKeyDown);
            goButton.removeEventListener('click', onGoClick);
        },
    });
}

export function getHexDumpStyles(fontSize, adjustCellHeight = 0) {
    const rowHeight = getRowHeight(fontSize, adjustCellHeight);
    return `
        #notes-hex-wrap {
            flex: 1;
            display: none;
            min-width: 0;
            min-height: 0;
            overflow: hidden;
            padding: 0;
        }

        #notes-hex-wrap[data-active="true"] {
            display: block;
        }

        #notes-hex {
            min-width: 0;
            width: 100%;
            height: 100%;
            display: grid;
            grid-template-rows: auto minmax(0, 1fr);
            gap: 6px;
        }

        .notes-hex-toolbar {
            display: flex;
            align-items: center;
            gap: 8px;
            font-size: ${Math.max(10, fontSize - 2)}px;
            color: var(--fg);
            font-family: var(--font-family);
        }

        .notes-hex-toolbar-label {
            opacity: 0.85;
        }

        .notes-hex-size {
            margin-left: auto;
            opacity: 0.7;
        }

        .notes-hex-offset-input {
            width: 112px;
            border: 1px solid rgba(255, 255, 255, 0.25);
            background: transparent;
            color: var(--fg);
            padding: 3px 8px;
            border-radius: 4px;
            outline: none;
            font-size: ${Math.max(10, fontSize - 2)}px;
            font-family: var(--font-family);
        }

        .notes-hex-offset-input:focus {
            border-color: var(--accent);
        }

        .notes-hex-offset-go {
            border: 1px solid rgba(255, 255, 255, 0.3);
            background: transparent;
            color: var(--fg);
            padding: 3px 10px;
            border-radius: 4px;
            cursor: pointer;
            font-size: ${Math.max(10, fontSize - 2)}px;
            font-family: var(--font-family);
        }

        .notes-hex-offset-go:hover {
            border-color: var(--accent);
            color: var(--accent);
        }

        .notes-hex-viewport {
            width: 100%;
            min-width: 0;
            max-width: 100%;
            height: 100%;
            overflow-y: auto;
            overflow-x: auto;
            min-height: 0;
        }

        .notes-hex-header {
            display: block;
            min-width: max-content;
            position: sticky;
            top: 0;
            z-index: 1;
            background: var(--bg);
            border-bottom: 1px solid rgba(255, 255, 255, 0.15);
            margin-bottom: 4px;
            padding-bottom: 4px;
            font-size: ${fontSize}px;
            color: rgba(255, 255, 255, 0.8);
            font-family: var(--font-family);
            white-space: pre;
        }

        .notes-hex-dump {
            display: block;
            min-width: max-content;
            margin: 0;
            white-space: pre;
            font-size: ${fontSize}px;
            line-height: ${rowHeight}px;
            background: transparent;
            color: var(--fg);
            font-family: var(--font-family);
        }

        .notes-hex-row {
            font-family: var(--font-family);
        }

        .notes-hex-offset {
            color: #8abbc3;
        }

        .notes-hex-ascii-printable {
            color: #fc6a5d;
        }

        .notes-hex-ascii-non-graphic {
            color: #50fa7b;
        }

        .notes-hex-byte-empty {
            color: transparent;
        }

        .notes-hex-byte-00 {
            color: #808080;
        }

        .notes-hex-byte-0 {
            color: oklch(75% 0.18 360);
        }

        .notes-hex-byte-1 {
            color: oklch(75% 0.18 23);
        }

        .notes-hex-byte-2 {
            color: oklch(75% 0.18 50);
        }

        .notes-hex-byte-3 {
            color: oklch(75% 0.18 65);
        }

        .notes-hex-byte-4 {
            color: oklch(75% 0.18 77);
        }

        .notes-hex-byte-5 {
            color: oklch(75% 0.18 103);
        }

        .notes-hex-byte-6 {
            color: oklch(75% 0.18 130);
        }

        .notes-hex-byte-7 {
            color: oklch(75% 0.18 142);
        }

        .notes-hex-byte-8 {
            color: oklch(75% 0.18 150);
        }

        .notes-hex-byte-9 {
            color: oklch(75% 0.18 163);
        }

        .notes-hex-byte-A {
            color: oklch(75% 0.18 184);
        }

        .notes-hex-byte-B {
            color: oklch(75% 0.18 209);
        }

        .notes-hex-byte-C {
            color: oklch(75% 0.18 232);
        }

        .notes-hex-byte-D {
            color: oklch(75% 0.18 254);
        }

        .notes-hex-byte-E {
            color: oklch(75% 0.18 294);
        }

        .notes-hex-byte-F {
            color: oklch(75% 0.18 328);
        }

        .notes-hex-byte-FF {
            color: #ffffff;
        }
    `;
}