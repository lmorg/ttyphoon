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
});
