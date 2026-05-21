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

function createLineNumbers(content) {
    const totalLines = Math.max(1, String(content || '').split('\n').length);
    return Array.from({ length: totalLines }, (_, i) => String(i + 1)).join('\n');
}

function convertToJupyterCodeBlocks(container) {
    const codeBlocks = container.querySelectorAll('pre');

    codeBlocks.forEach((pre) => {
        const code = pre.querySelector('code');
        if (!code) {
            return;
        }

        const langClass = Array.from(code.classList).find((cls) => cls.startsWith('language-'));
        if (!langClass) {
            return;
        }

        const language = langClass.replace('language-', '');
        if (!language || language === 'mermaid') {
            return;
        }

        const content = code.textContent || '';

        const wrapper = document.createElement('div');
        wrapper.className = 'jupyter-code-block';

        const toolbar = document.createElement('div');
        toolbar.className = 'jupyter-toolbar';

        const runtimeBadge = document.createElement('span');
        runtimeBadge.className = 'jupyter-runtime-dropdown';
        runtimeBadge.textContent = language;
        runtimeBadge.title = `Runtime: ${language}`;

        toolbar.appendChild(runtimeBadge);

        const codeEditor = document.createElement('div');
        codeEditor.className = 'jupyter-code-editor';

        const lineNumbers = document.createElement('pre');
        lineNumbers.className = 'jupyter-line-numbers';
        const lineNumbersInner = document.createElement('div');
        lineNumbersInner.className = 'jupyter-line-numbers-inner';
        lineNumbersInner.textContent = createLineNumbers(content);
        lineNumbers.appendChild(lineNumbersInner);

        const codeArea = document.createElement('div');
        codeArea.className = 'jupyter-code-area';

        const highlighted = document.createElement('pre');
        highlighted.className = 'jupyter-highlight';
        const highlightedCode = document.createElement('code');
        highlightedCode.className = `language-${language}`;
        highlightedCode.textContent = content;
        highlighted.appendChild(highlightedCode);

        const editable = document.createElement('textarea');
        editable.className = 'jupyter-code-editable';
        editable.setAttribute('readonly', 'readonly');
        editable.setAttribute('spellcheck', 'false');
        editable.value = content;

        editable.addEventListener('scroll', () => {
            highlighted.scrollTop = editable.scrollTop;
            highlighted.scrollLeft = editable.scrollLeft;
            lineNumbersInner.style.transform = `translateY(-${editable.scrollTop}px)`;
        });

        codeArea.appendChild(highlighted);
        codeArea.appendChild(editable);
        codeEditor.appendChild(lineNumbers);
        codeEditor.appendChild(codeArea);

        wrapper.appendChild(toolbar);
        wrapper.appendChild(codeEditor);

        pre.replaceWith(wrapper);
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

    convertToJupyterCodeBlocks(body);
    await applySyntaxHighlighting(body);
    await renderMermaidDiagrams(body);
    await processWailsImages(body);
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
