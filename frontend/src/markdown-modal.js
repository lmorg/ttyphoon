import { EventsOn } from './wails-runtime.js';
import './markdown-modal.css';
import { marked } from 'marked';
import { showLocalMenu } from './popup_menu';
import {
    configureMarked,
    applySyntaxHighlighting,
    renderMermaidDiagrams,
    processWailsImages,
    enableFullscreenImages,
} from './markdown-utils';

configureMarked();

let markdownModalTableWordWrapEnabled = true;
let markdownModalHooks = {};
let currentMarkdownContent = '';

function applyTableWordWrapMode(container) {
    if (!container) {
        return;
    }

    if (markdownModalTableWordWrapEnabled) {
        container.classList.remove('markdown-modal-table-wordwrap-off');
    } else {
        container.classList.add('markdown-modal-table-wordwrap-off');
    }
}

function buildLineNumbers(lineCount) {
    return Array.from({ length: Math.max(1, lineCount) }, (_, index) => String(index + 1)).join('\n');
}

function wrapTablesForHorizontalScroll(container) {
    if (!container) {
        return;
    }

    const tables = container.querySelectorAll('table');
    tables.forEach((table) => {
        if (!(table instanceof HTMLElement)) {
            return;
        }

        const parent = table.parentElement;
        if (!parent || parent.classList.contains('notes-table-scroll-wrap') || parent.classList.contains('markdown-modal-table-scroll-wrap')) {
            return;
        }

        const wrapper = document.createElement('div');
        wrapper.className = 'markdown-modal-table-scroll-wrap notes-table-scroll-wrap';
        table.before(wrapper);
        wrapper.appendChild(table);
    });
}

function enhanceCodeBlocks(container) {
    const codeBlocks = container.querySelectorAll('pre > code[class*="language-"]');

    codeBlocks.forEach((code) => {
        const pre = code.parentElement;
        if (!pre || pre.parentElement?.classList.contains('markdown-modal-code-shell')) {
            return;
        }

        const languageClass = Array.from(code.classList).find((name) => name.startsWith('language-'));
        const language = languageClass ? languageClass.slice('language-'.length) : '';
        if (!language || language === 'mermaid' || language === 'raw') {
            return;
        }

        const lineCount = String(code.textContent || '').split('\n').length;

        const shell = document.createElement('div');
        shell.className = 'markdown-modal-code-shell';

        const lineNumbers = document.createElement('div');
        lineNumbers.className = 'markdown-modal-code-lines';

        const lineNumbersInner = document.createElement('div');
        lineNumbersInner.className = 'markdown-modal-code-lines-inner';
        lineNumbersInner.textContent = buildLineNumbers(lineCount);
        lineNumbers.appendChild(lineNumbersInner);

        const viewport = document.createElement('div');
        viewport.className = 'markdown-modal-code-viewport';

        pre.classList.add('markdown-modal-code-pre');

        pre.addEventListener('scroll', () => {
            lineNumbersInner.style.transform = `translateY(-${pre.scrollTop}px)`;
        });

        pre.replaceWith(shell);
        viewport.appendChild(pre);
        shell.appendChild(lineNumbers);
        shell.appendChild(viewport);
    });
}

function ensureMarkdownModalDom() {
    if (document.getElementById('markdown-modal-overlay')) {
        return;
    }

    const overlay = document.createElement('div');
    overlay.id = 'markdown-modal-overlay';
    overlay.className = 'markdown-modal-overlay';
    overlay.style.display = 'none';

    const dialog = document.createElement('div');
    dialog.id = 'markdown-modal-dialog';
    dialog.className = 'markdown-modal-dialog';

    const body = document.createElement('div');
    body.id = 'markdown-modal-body';
    body.className = 'markdown-modal-body markdown-body';

    dialog.appendChild(body);
    overlay.appendChild(dialog);
    document.body.appendChild(overlay);
}

function closeMarkdownModal() {
    currentMarkdownContent = '';
    document.body.classList.remove('markdown-modal-open');
    const overlay = document.getElementById('markdown-modal-overlay');
    if (overlay) {
        overlay.style.display = 'none';
        const body = document.getElementById('markdown-modal-body');
        if (body) {
            body.innerHTML = '';
        }
    }
}

export async function showMarkdownModal(markdownContent, id = '') {
    ensureMarkdownModalDom();

    const overlay = document.getElementById('markdown-modal-overlay');
    const body = document.getElementById('markdown-modal-body');

    if (!overlay || !body) {
        return;
    }

    currentMarkdownContent = markdownContent;
    body.innerHTML = marked.parse(markdownContent);
    await renderMermaidDiagrams(body);
    await applySyntaxHighlighting(body);
    processWailsImages(body);
    wrapTablesForHorizontalScroll(body);
    markdownModalHooks.onTablesRendered?.(body, { id, wrapped: markdownModalTableWordWrapEnabled });
    applyTableWordWrapMode(body);
    enhanceCodeBlocks(body);
    enableFullscreenImages(body);

    document.body.classList.add('markdown-modal-open');
    overlay.style.display = 'flex';
}

export function initMarkdownModal(hooks = {}) {
    markdownModalHooks = hooks;
    ensureMarkdownModalDom();

    const overlay = document.getElementById('markdown-modal-overlay');
    if (!overlay) {
        return;
    }

    // Close on backdrop click (but not on dialog click)
    overlay.addEventListener('pointerdown', (e) => {
        if (e.target === overlay) {
            closeMarkdownModal();
        }
    });

    // Close on Escape key
    document.addEventListener('keydown', (e) => {
        // Let a Notes prompt opened from the modal, such as the SQL filter, handle Escape first.
        if (e.key === 'Escape' && document.getElementById('notes-modal')?.dataset.open === 'true') {
            return;
        }
        if (e.key === 'Escape' && overlay.style.display !== 'none') {
            e.stopPropagation();
            closeMarkdownModal();
        }
    }, true);

    overlay.addEventListener('contextmenu', (e) => {
        const body = document.getElementById('markdown-modal-body');
        if (!body || overlay.style.display === 'none') {
            return;
        }

        const target = e.target instanceof Element ? e.target : null;
        const tableTarget = target?.closest?.('.markdown-modal-table-scroll-wrap, table, th, td');
        if (!tableTarget && !markdownModalHooks.onPrint && !markdownModalHooks.copyMenuItem) {
            return;
        }

        e.preventDefault();
        e.stopPropagation();

        const isWrapEnabled = !body.classList.contains('markdown-modal-table-wordwrap-off');
        const table = tableTarget?.closest('table') || tableTarget?.querySelector('table');
        const copyItem = markdownModalHooks.copyMenuItem?.(body);
        const items = copyItem ? [copyItem] : [];
        if (tableTarget) items.push(...(items.length ? [{ title: '-', icon: 0 }] : []), {
            title: 'Word wrap',
            icon: isWrapEnabled ? 0xf00c : 0x20,
            onSelect: () => {
                markdownModalTableWordWrapEnabled = !isWrapEnabled;
                applyTableWordWrapMode(body);
            },
        });
        const tableItems = table ? markdownModalHooks.tableMenuItems?.(table) || [] : [];
        if (tableItems.length > 0) {
            items.push({ title: '-', icon: 0 }, ...tableItems);
        }
        if (markdownModalHooks.onPrint) {
            const markdown = currentMarkdownContent;
            if (items.length) items.push({ title: '-', icon: 0 });
            items.push({ title: 'Print...', icon: 0xf02f, onSelect: () => markdownModalHooks.onPrint(markdown) });
        }

        showLocalMenu({
            title: 'Select an action',
            options: items.map((item) => item.title),
            icons: items.map((item) => item.icon),
            x: e.clientX,
            y: e.clientY,
            showNextToMouseCursor: true,
            onSelect: (index) => items[index]?.onSelect?.(),
        });
    });

    EventsOn('showMarkdownModal', (payload) => {
        const message = Array.isArray(payload) ? payload[0] : payload;
        if (typeof message === 'string') {
            void showMarkdownModal(message);
            return;
        }
        void showMarkdownModal(String(message?.markdown || ''), String(message?.id || ''));
    });
}
