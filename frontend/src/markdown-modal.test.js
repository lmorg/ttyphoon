import { beforeEach, describe, expect, it, vi } from 'vitest';

const { eventsOnMock, showLocalMenuMock } = vi.hoisted(() => ({
    eventsOnMock: vi.fn(),
    showLocalMenuMock: vi.fn(),
}));

vi.mock('./wails-runtime.js', () => ({ EventsOn: eventsOnMock }));
vi.mock('./popup_menu', () => ({ showLocalMenu: showLocalMenuMock }));

import { initMarkdownModal, showMarkdownModal } from './markdown-modal.js';

const tableMarkdown = [
    '| Name | Count |',
    '| --- | --- |',
    '| beta | 2 |',
].join('\n');

async function flushPromises() {
    await new Promise((resolve) => setTimeout(resolve, 0));
}

function modalEventHandler() {
    return eventsOnMock.mock.calls.find(([name]) => name === 'showMarkdownModal')?.[1];
}

describe('markdown modal', () => {
    beforeEach(() => {
        document.body.innerHTML = '';
        document.body.className = '';
        eventsOnMock.mockReset();
        showLocalMenuMock.mockReset();
    });

    it('keeps encoded newlines in table code cells after highlighting', async () => {
        await showMarkdownModal([
            '| Filename | Diff |',
            '| --- | --- |',
            '| a.txt | <pre><code>-old&#10;+new &#124; value</code></pre> |',
        ].join('\n'));

        const code = document.querySelector('#markdown-modal-body td pre code');
        expect(code?.textContent).toBe('-old\n+new | value');
        expect(document.querySelectorAll('#markdown-modal-body tbody tr')).toHaveLength(1);
    });

    it('passes the caller ID and tables to the Notes table hooks', async () => {
        const onTablesRendered = vi.fn();
        initMarkdownModal({ onTablesRendered });

        modalEventHandler()({ id: 'git-info:/repo', markdown: tableMarkdown });
        await flushPromises();

        const [container, options] = onTablesRendered.mock.calls[0];
        expect(options).toEqual({ id: 'git-info:/repo', wrapped: true });
        expect(container.querySelector('.notes-table-scroll-wrap.markdown-modal-table-scroll-wrap table')).not.toBeNull();
        expect(document.body.classList.contains('markdown-modal-open')).toBe(true);
    });

    it('adds Notes table actions to the table context menu', async () => {
        const filterItem = { title: 'SQL filter...', icon: 0xf0b0, onSelect: vi.fn() };
        const tableMenuItems = vi.fn(() => [filterItem]);
        initMarkdownModal({ tableMenuItems });
        await showMarkdownModal(tableMarkdown, 'command-meta');

        const cell = document.querySelector('#markdown-modal-body td');
        cell.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, clientX: 10, clientY: 10 }));

        const menu = showLocalMenuMock.mock.calls.at(-1)[0];
        expect(tableMenuItems).toHaveBeenCalledWith(cell.closest('table'));
        expect(menu.options).toEqual(['Word wrap', '-', 'SQL filter...']);
        menu.onSelect(2);
        expect(filterItem.onSelect).toHaveBeenCalled();
    });

    it('prints the original Markdown from both body and table menus without changing the modal', async () => {
        const onPrint = vi.fn();
        initMarkdownModal({ onPrint });
        const markdown = '# Full document\n\n' + tableMarkdown + '\n\nFinal paragraph';
        await showMarkdownModal(markdown);
        const body = document.getElementById('markdown-modal-body');
        body.querySelector('p').remove();
        const displayed = body.innerHTML;
        for (const selector of ['h1', 'td']) {
            body.querySelector(selector).dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }));
            const menu = showLocalMenuMock.mock.calls.at(-1)[0];
            expect(menu.options).toContain('Print...');
            menu.onSelect(menu.options.indexOf('Print...'));
            expect(onPrint).toHaveBeenLastCalledWith(markdown);
        }
        expect(onPrint).toHaveBeenCalledTimes(2);
        expect(body.innerHTML).toBe(displayed);
        expect(document.getElementById('markdown-modal-overlay').style.display).toBe('flex');
    });

    it('leaves Escape to an open Notes prompt before closing', async () => {
        initMarkdownModal();
        await showMarkdownModal(tableMarkdown);
        const overlay = document.getElementById('markdown-modal-overlay');
        const notesModal = document.createElement('div');
        notesModal.id = 'notes-modal';
        notesModal.dataset.open = 'true';
        document.body.appendChild(notesModal);

        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
        expect(overlay.style.display).toBe('flex');

        notesModal.dataset.open = 'false';
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
        expect(overlay.style.display).toBe('none');
        expect(document.body.classList.contains('markdown-modal-open')).toBe(false);
    });

    it('offers Copy selection in the body and table menus using a snapshot from menu open', async () => {
        const copy = vi.fn();
        const copyMenuItem = vi.fn(() => ({ title: 'Copy selection', icon: 0xf0c5, onSelect: copy }));
        initMarkdownModal({ copyMenuItem });
        await showMarkdownModal('# Heading\n\n' + tableMarkdown);
        const body = document.getElementById('markdown-modal-body');
        for (const selector of ['h1', 'td']) {
            body.querySelector(selector).dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }));
            const menu = showLocalMenuMock.mock.calls.at(-1)[0];
            expect(copyMenuItem).toHaveBeenLastCalledWith(body);
            expect(menu.options[0]).toBe('Copy selection');
            menu.onSelect(0);
        }
        expect(copy).toHaveBeenCalledTimes(2);
    });
});
