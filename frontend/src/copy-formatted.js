const omitted = [
    'script', 'style', 'button', 'input', 'textarea', 'select', '[hidden]', '[aria-hidden="true"]',
    '.notes-table-row-filtered', '.notes-copy-block-actions', '.notes-sort-icon',
    '.notes-table-col-resize-handle', '.notes-cellref', '.markdown-modal-code-lines',
    '.notes-table-sql-filter-footer', '.notes-table-graph-toolbar',
].join(',');

const semanticTags = new Set([
    'p', 'div', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'strong', 'b', 'em', 'i', 'u', 'del', 's',
    'code', 'pre', 'blockquote', 'ul', 'ol', 'li', 'br', 'hr', 'table', 'thead', 'tbody', 'tfoot',
    'tr', 'th', 'td', 'a', 'sup', 'sub',
]);
const editable = 'input, textarea, [contenteditable]:not([contenteditable="false"]), .monaco-editor';

export function captureRenderedSelection(container, selection = window.getSelection()) {
    if (!container || !selection || selection.isCollapsed || selection.rangeCount === 0) return null;
    const ranges = Array.from({ length: selection.rangeCount }, (_, index) => selection.getRangeAt(index).cloneRange())
        .filter((range) => container.contains(range.startContainer) && container.contains(range.endContainer));
    if (!ranges.length) return null;
    return captureRanges(container, ranges);
}

export function captureRenderedElement(element) {
    const range = document.createRange();
    range.selectNode(element);
    return captureRanges(element.parentElement, [range]);
}

export function captureRenderedElements(elements) {
    const fragment = document.createElement('div');
    const images = [];
    for (const element of elements) {
        const snapshot = captureRenderedElement(element);
        if (!snapshot) continue;
        fragment.append(...snapshot.fragment.childNodes);
        images.push(...snapshot.images);
    }
    return fragment.childNodes.length ? { fragment, images } : null;
}

function captureRanges(container, ranges) {
    const fragment = document.createElement('div');
    const images = [];

    function copy(node, range, parent) {
        if (!range.intersectsNode(node)) return;
        if (node.nodeType === Node.TEXT_NODE) {
            const start = node === range.startContainer ? range.startOffset : 0;
            const end = node === range.endContainer ? range.endOffset : node.length;
            if (start < end) parent.append(document.createTextNode(node.data.slice(start, end)));
            return;
        }
        if (!(node instanceof Element) || node.matches(omitted) || getComputedStyle(node).display === 'none') return;
        if (node.matches('img, svg, canvas')) {
            const image = document.createElement('img');
            const alt = node.getAttribute('alt') || node.getAttribute('aria-label') || node.querySelector('title')?.textContent || 'Image';
            const source = node.getAttribute('src') || '';
            image.alt = alt;
            if (source && !source.startsWith('data:')) image.dataset.copySource = source;
            parent.append(image);
            // Snapshot SVG/canvas content before a later render can replace the source.
            const entry = { image, source, alt };
            try {
                if (node.localName === 'svg') entry.svg = snapshotSVG(node);
                if (node.localName === 'canvas') entry.source = node.toDataURL('image/png');
            } catch (error) {
                entry.error = error;
            }
            images.push(entry);
            return;
        }
        const tag = node.localName;
        const output = semanticTags.has(tag) ? document.createElement(tag === 's' ? 'del' : tag) : document.createDocumentFragment();
        if (output instanceof Element) {
            for (const key of ['start', 'colspan', 'rowspan']) {
                if (node.hasAttribute(key)) output.setAttribute(key, node.getAttribute(key));
            }
            if (tag === 'a') output.setAttribute('href', node.getAttribute('href') || '');
            if (tag === 'ol') {
                const first = Array.from(node.children).find((child) => child.localName === 'li' && range.intersectsNode(child));
                if (first) output.setAttribute('start', String(Number(node.getAttribute('start') || 1) + Array.from(node.children).filter((child) => child.localName === 'li').indexOf(first)));
            }
        }
        for (const child of node.childNodes) copy(child, range, output);
        // Empty boundary nodes are not selected content, but image-only selections are.
        if (output.childNodes.length || ['br', 'hr', 'td', 'th', 'tr', 'li'].includes(tag)) parent.append(output);
    }

    for (const range of ranges) {
        for (const child of container.childNodes) copy(child, range, fragment);
    }
    return fragment.childNodes.length ? { fragment, images } : null;
}

function snapshotSVG(svg) {
    const clone = svg.cloneNode(true);
    const originals = [svg, ...svg.querySelectorAll('*')];
    const copies = [clone, ...clone.querySelectorAll('*')];
    const properties = ['fill', 'stroke', 'stroke-width', 'font-family', 'font-size', 'font-weight', 'color', 'background-color'];
    originals.forEach((element, index) => {
        const style = getComputedStyle(element);
        for (const property of properties) {
            const value = style.getPropertyValue(property);
            if (value) copies[index].style.setProperty(property, value);
        }
    });
    const bounds = svg.getBoundingClientRect();
    const viewBox = svg.getAttribute('viewBox')?.trim().split(/[\s,]+/).map(Number);
    const validViewBox = viewBox?.length === 4 && viewBox.every(Number.isFinite)
        && viewBox[2] > 0 && viewBox[3] > 0;
    const intrinsicLength = (name) => {
        const value = svg.getAttribute(name)?.trim();
        return value && /^\d+(?:\.\d+)?(?:px)?$/.test(value) ? parseFloat(value) : 0;
    };
    const width = validViewBox ? viewBox[2] : intrinsicLength('width') || bounds.width || 800;
    const height = validViewBox ? viewBox[3] : intrinsicLength('height') || bounds.height || 600;
    // Panel layout must not constrain the exported SVG's intrinsic dimensions.
    for (const property of ['width', 'height', 'max-width', 'max-height', 'min-width', 'min-height']) {
        clone.style.removeProperty(property);
    }
    clone.setAttributeNS('http://www.w3.org/2000/xmlns/', 'xmlns', 'http://www.w3.org/2000/svg');
    clone.setAttribute('width', String(width));
    clone.setAttribute('height', String(height));
    return new XMLSerializer().serializeToString(clone);
}

export async function imageToPNG({ source, svg, canvas }) {
    if (canvas) return canvas.toDataURL('image/png');
    const image = new Image();
    image.crossOrigin = 'anonymous';
    const loaded = new Promise((resolve, reject) => {
        image.onload = resolve;
        image.onerror = () => reject(new Error('Image could not be loaded'));
    });
    image.src = svg
        ? `data:image/svg+xml;base64,${btoa(Array.from(new TextEncoder().encode(svg), (byte) => String.fromCharCode(byte)).join(''))}`
        : source;
    let timer;
    try {
        await Promise.race([
            loaded,
            new Promise((_, reject) => { timer = setTimeout(() => reject(new Error('Image loading timed out')), 10000); }),
        ]);
        if (!image.naturalWidth || !image.naturalHeight || image.naturalWidth > 16384 || image.naturalHeight > 16384
            || image.naturalWidth * image.naturalHeight > 16 * 1024 * 1024) {
            throw new Error('Image dimensions exceed the copy limit');
        }
        const output = document.createElement('canvas');
        output.width = image.naturalWidth;
        output.height = image.naturalHeight;
        const context = output.getContext('2d');
        if (!context) throw new Error('PNG rendering is unavailable');
        context.drawImage(image, 0, 0);
        return output.toDataURL('image/png');
    } finally {
        clearTimeout(timer);
        image.onload = null;
        image.onerror = null;
    }
}

export async function prepareRenderedCopy(snapshot, rasterize = imageToPNG) {
    const warnings = [];
    for (const entry of snapshot.images) {
        try {
            if (entry.error) throw entry.error;
            const png = await rasterize(entry);
            if (!png.startsWith('data:image/png;base64,')) throw new Error('Image conversion returned no PNG');
            entry.image.src = png;
        } catch (error) {
            const label = entry.source ? `![${entry.alt}](${entry.source})` : `[${entry.alt}]`;
            entry.image.replaceWith(document.createTextNode(label));
            warnings.push(`${entry.alt}: ${error instanceof Error ? error.message : error}`);
        }
    }
    return { html: snapshot.fragment.innerHTML, warnings };
}

export function createRenderedSelectionCopier({ write, notify, rasterize = imageToPNG }) {
    let pending = Promise.resolve();
    const copy = (snapshot, writer = write) => {
        if (!snapshot) return Promise.resolve();
        // Keep clipboard writes in the order the user requested them.
        pending = pending.then(async () => {
            try {
                const { html, warnings } = await prepareRenderedCopy(snapshot, rasterize);
                const uploadWarnings = await writer(html);
                if (Array.isArray(uploadWarnings)) warnings.push(...uploadWarnings);
                if (warnings.length) notify(`Copied with image fallback: ${warnings.join('; ')}`, 'warn');
            } catch (error) {
                notify(`Failed to copy selection: ${error instanceof Error ? error.message : error}`, 'error');
            }
        });
        return pending;
    };
    return {
        copy,
        menuItem(container) {
            const snapshot = captureRenderedSelection(container);
            return { title: 'Copy selection', icon: 0xf0c5, onSelect: () => copy(snapshot) };
        },
        handleCopy(event) {
            if (event.target instanceof Element && event.target.closest(editable)) return;
            const selection = window.getSelection();
            if (!selection || selection.isCollapsed) return;
            const node = selection.anchorNode;
            const element = node instanceof Element ? node : node?.parentElement;
            if (element?.closest(editable)) return;
            let container = element?.closest('[data-copy-rendered], .markdown-body');
            while (container && !container.contains(selection.focusNode)) {
                container = container.parentElement?.closest('[data-copy-rendered], .markdown-body');
            }
            const snapshot = captureRenderedSelection(container, selection);
            if (!snapshot) return;
            event.preventDefault();
            event.stopImmediatePropagation();
            void copy(snapshot);
        },
    };
}
