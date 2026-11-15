import { describe, expect, it, vi } from 'vitest';
import { createDocumentBuffer } from './notes_document_buffer';
import { createFakeMonacoAdapter } from './test-support/fake_monaco_adapter';

function attachFakeView(buffer) {
    const view = createFakeMonacoAdapter(null, { onChange: () => buffer.handleViewChange() });
    buffer.attachView(view);
    return view;
}

describe('notes document buffer', () => {
    it('edits its own text before a view is attached and notifies listeners', () => {
        const buffer = createDocumentBuffer('hello world');
        const listener = vi.fn();
        buffer.onChange(listener);

        buffer.replaceRange(6, 11, 'there', { source: 'test' });

        expect(buffer.getValue()).toBe('hello there');
        expect(buffer.getSelection()).toEqual({ start: 11, end: 11 });
        expect(listener).toHaveBeenCalledWith({ source: 'test' });
    });

    it('does not notify on setValue, and normalises CRLF', () => {
        const buffer = createDocumentBuffer();
        const listener = vi.fn();
        buffer.onChange(listener);

        buffer.setValue('a\r\nb\rc');

        expect(buffer.getValue()).toBe('a\nb\nc');
        expect(listener).not.toHaveBeenCalled();
    });

    it('hands pending text and selection to the view when attached, without notifying', () => {
        const buffer = createDocumentBuffer('pending text');
        buffer.setSelection(3, 7);
        const listener = vi.fn();
        buffer.onChange(listener);

        const view = attachFakeView(buffer);

        expect(view.getValue()).toBe('pending text');
        expect(view.getSelectionOffsets()).toEqual({ start: 3, end: 7 });
        expect(listener).not.toHaveBeenCalled();
    });

    it('routes edits through the attached view so they can be undone', () => {
        const buffer = createDocumentBuffer('{ "a": 1 }');
        const view = attachFakeView(buffer);
        const listener = vi.fn();
        buffer.onChange(listener);

        buffer.replaceAll('{"a":1}');

        expect(view.getValue()).toBe('{"a":1}');
        expect(listener).toHaveBeenCalledTimes(1);

        view.undo();
        expect(buffer.getValue()).toBe('{ "a": 1 }');
        expect(listener).toHaveBeenCalledTimes(2);
    });

    it('replaces the whole document as the smallest changed range', () => {
        const buffer = createDocumentBuffer('Name,Value\nAlpha,1');
        const view = attachFakeView(buffer);
        const replaceRange = vi.spyOn(view, 'replaceRange');

        buffer.replaceAll('Name,Value\nAlpha,1\n,', { selectionStart: 2 });

        expect(replaceRange).toHaveBeenCalledWith(18, 18, '\n,', 'notes-document-edit');
        expect(buffer.getSelection()).toEqual({ start: 2, end: 2 });
    });

    it('does nothing when replaceAll is given the current text', () => {
        const buffer = createDocumentBuffer('same');
        const listener = vi.fn();
        buffer.onChange(listener);

        buffer.replaceAll('same');

        expect(listener).not.toHaveBeenCalled();
    });

    it('maps the selection through an edit when preserveSelection is set', () => {
        const buffer = createDocumentBuffer('one\ntwo\nthree');
        const view = attachFakeView(buffer);
        view.setSelectionOffsets(8, 13);

        buffer.replaceRange(0, 3, 'first', { preserveSelection: true });

        expect(buffer.getValue()).toBe('first\ntwo\nthree');
        expect(buffer.getSelection()).toEqual({ start: 10, end: 15 });
    });
});
