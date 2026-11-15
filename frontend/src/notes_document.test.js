import { describe, expect, it, vi } from 'vitest';
import { createNotesDocumentRegistry, notesDocumentKey } from './notes_document';

describe('notesDocumentKey', () => {
    it('qualifies $PROJECT paths with the project root', () => {
        expect(notesDocumentKey('$PROJECT/src/a.go', '/work/one/')).toBe('/work/one/src/a.go');
        expect(notesDocumentKey('$PROJECT/src/a.go', '/work/two')).toBe('/work/two/src/a.go');
        expect(notesDocumentKey('$PROJECTS/a.go', '/work/one')).toBe('$PROJECTS/a.go');
        expect(notesDocumentKey('$NOTES/a.md', '/work/one')).toBe('$NOTES/a.md');
    });
});

describe('createNotesDocumentRegistry', () => {
    it('shares one document per file and keeps project files apart', () => {
        const registry = createNotesDocumentRegistry();
        const notes = registry.open('$NOTES/a.md', { fileType: 'markdown' });

        expect(registry.open('$NOTES/a.md', { fileType: 'code' })).toBe(notes);
        expect(notes.fileType).toBe('markdown');

        const one = registry.open('$PROJECT/a.go', { project: '/work/one' });
        const two = registry.open('$PROJECT/a.go', { project: '/work/two' });
        expect(two).not.toBe(one);
        expect(registry.get('$PROJECT/a.go', '/work/one')).toBe(one);
        expect(registry.size).toBe(3);
    });

    it('keeps text separately for each document', () => {
        const registry = createNotesDocumentRegistry();
        const a = registry.open('$NOTES/a.md');
        const b = registry.open('$NOTES/b.md');

        a.buffer.setValue('alpha');
        b.buffer.setValue('beta');

        expect(registry.get('$NOTES/a.md').buffer.getValue()).toBe('alpha');
        expect(registry.get('$NOTES/b.md').buffer.getValue()).toBe('beta');
    });

    it('clears pending timers and releases the document once on close', async () => {
        vi.useFakeTimers();
        try {
            const onClose = vi.fn(() => 'released');
            const registry = createNotesDocumentRegistry({ onClose });
            const doc = registry.open('$NOTES/a.md');
            const autosave = vi.fn();
            const lspChange = vi.fn();
            doc.autosaveTimer = setTimeout(autosave, 100);
            doc.lspChangeTimer = setTimeout(lspChange, 100);

            expect(registry.close('$NOTES/a.md')).toBe('released');
            expect(registry.close(doc)).toBeUndefined();
            vi.advanceTimersByTime(200);

            expect(onClose).toHaveBeenCalledTimes(1);
            expect(onClose).toHaveBeenCalledWith(doc);
            expect(autosave).not.toHaveBeenCalled();
            expect(lspChange).not.toHaveBeenCalled();
            expect(registry.has('$NOTES/a.md')).toBe(false);
            expect(registry.open('$NOTES/a.md')).not.toBe(doc);
        } finally {
            vi.useRealTimers();
        }
    });
});
