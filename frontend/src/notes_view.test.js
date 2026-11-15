import { describe, expect, it } from 'vitest';
import { createNotesView } from './notes_view.js';

describe('createNotesView', () => {
    it('creates independent views whose parts are scoped to their own root', () => {
        const host = document.createElement('div');
        const first = createNotesView({ host });
        const second = createNotesView({ host });

        expect(first.id).not.toBe(second.id);
        expect(host.children).toHaveLength(2);
        expect(first.parts.preview).not.toBe(second.parts.preview);
        expect(first.root.contains(first.parts.preview)).toBe(true);
        expect(first.contains(second.parts.preview)).toBe(false);
        expect(host.querySelectorAll('[id]')).toHaveLength(0);

        first.destroy();
        expect(host.children).toHaveLength(1);
    });

    it('keeps tabs in order, opening new ones after the given tab', () => {
        const view = createNotesView();
        view.addTab('a');
        view.addTab('b');
        view.addTab('c', 'a');
        view.addTab('a');
        expect(view.tabs).toEqual(['a', 'c', 'b']);
        expect(view.removeTab('c')).toBe(1);
        expect(view.removeTab('missing')).toBe(-1);
        expect(view.tabs).toEqual(['a', 'b']);
    });

    it('moves the tab and remembered state of a renamed document to its new key', () => {
        const view = createNotesView();
        view.addTab('a');
        view.addTab('b');
        view.addTab('c');
        view.rememberDocument('a', { viewMode: 'editor' });
        view.shownDocumentKey = 'a';

        view.renameDocument('a', 'c');

        expect(view.tabs).toEqual(['c', 'b']);
        expect(view.recallDocument('c')).toEqual({ viewMode: 'editor' });
        expect(view.recallDocument('a')).toBeNull();
        expect(view.shownDocumentKey).toBe('c');
    });
});
