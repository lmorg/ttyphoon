import { EventsOn } from '../wailsjs/runtime/runtime';
import './markdown-modal.css';
import { marked } from 'marked';
import {
    configureMarked,
    applySyntaxHighlighting,
    renderMermaidDiagrams,
    processWailsImages,
    enableFullscreenImages,
} from './markdown-utils';

configureMarked();

function buildLineNumbers(lineCount) {
    return Array.from({ length: Math.max(1, lineCount) }, (_, index) => String(index + 1)).join('\n');
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
    const overlay = document.getElementById('markdown-modal-overlay');
    if (overlay) {
        overlay.style.display = 'none';
        const body = document.getElementById('markdown-modal-body');
        if (body) {
            body.innerHTML = '';
        }
    }
}

export async function showMarkdownModal(markdownContent) {
    ensureMarkdownModalDom();

    const overlay = document.getElementById('markdown-modal-overlay');
    const body = document.getElementById('markdown-modal-body');

    if (!overlay || !body) {
        return;
    }

    body.innerHTML = marked.parse(markdownContent);
    await renderMermaidDiagrams(body);
    await applySyntaxHighlighting(body);
    await processWailsImages(body);
    enhanceCodeBlocks(body);
    enableFullscreenImages(body);

    overlay.style.display = 'flex';
}

export function initMarkdownModal() {
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
        if (e.key === 'Escape' && overlay.style.display !== 'none') {
            e.stopPropagation();
            closeMarkdownModal();
        }
    }, true);

    EventsOn('showMarkdownModal', (markdownContent) => {
        void showMarkdownModal(markdownContent);
    });
}
