import { describe, expect, it } from 'vitest';

import { initNotesLogPanel } from './notes-log-panel.js';

function createElements() {
    return {
        appRoot: document.createElement('div'),
        logOutput: document.createElement('div'),
        toolsLogWordwrap: document.createElement('button'),
        toolsLogTimestamp: document.createElement('button'),
        toolsLogError: document.createElement('button'),
        toolsLogWarn: document.createElement('button'),
        toolsLogInfo: document.createElement('button'),
        toolsLogDebug: document.createElement('button'),
        toolsLogTrace: document.createElement('button'),
        toolsLogSelectAll: document.createElement('button'),
        toolsLogMaximize: document.createElement('button'),
    };
}

function getLineTexts(logOutput) {
    return Array.from(logOutput.querySelectorAll('.notes-log-line')).map((line) => line.textContent);
}

describe('notes log panel', () => {
    it('caps rendered log output at max log lines', () => {
        const elements = createElements();
        const handlers = {};
        const eventsOn = (name, handler) => {
            handlers[name] = handler;
        };

        initNotesLogPanel(elements, eventsOn, 3);

        handlers.notesLog('one\ntwo\nthree\nfour\n');

        expect(elements.logOutput.childElementCount).toBe(3);
        expect(getLineTexts(elements.logOutput)).toEqual(['two', 'three', 'four']);
    });

    it('applies updated max line limits immediately', () => {
        const elements = createElements();
        const handlers = {};
        const eventsOn = (name, handler) => {
            handlers[name] = handler;
        };

        const panel = initNotesLogPanel(elements, eventsOn, 10);

        handlers.notesLog('alpha\nbeta\ngamma\ndelta\n');
        panel.setMaxLogLines(2);

        expect(elements.logOutput.childElementCount).toBe(2);
        expect(getLineTexts(elements.logOutput)).toEqual(['gamma', 'delta']);
    });

    it('filters log levels and selects only currently visible lines', () => {
        const elements = createElements();
        const handlers = {};
        const eventsOn = (name, handler) => {
            handlers[name] = handler;
        };

        initNotesLogPanel(elements, eventsOn, 10);
        handlers.notesLog([
            '2026/10/08 12:00:00 [error] error',
            '2026/10/08 12:00:01 [warn] warning',
            '2026/10/08 12:00:02 [info] information',
            '2026/10/08 12:00:03 [debug] debug',
            '2026/10/08 12:00:04 [trace] trace',
        ].join('\n'));

        expect(elements.logOutput.dataset.showError).toBe('true');
        expect(elements.logOutput.dataset.showWarn).toBe('true');
        expect(elements.logOutput.dataset.showInfo).toBe('true');
        expect(elements.logOutput.dataset.showDebug).toBe('false');
        expect(elements.logOutput.dataset.showTrace).toBe('false');
        expect(elements.toolsLogError.dataset.enabled).toBe('true');
        expect(elements.toolsLogWarn.dataset.enabled).toBe('true');
        expect(elements.toolsLogInfo.dataset.enabled).toBe('true');
        expect(elements.toolsLogDebug.dataset.enabled).toBe('false');
        expect(elements.toolsLogTrace.dataset.enabled).toBe('false');
        expect(elements.toolsLogWordwrap.dataset.enabled).toBe('false');
        expect(elements.toolsLogTimestamp.dataset.enabled).toBe('false');

        elements.toolsLogError.click();
        elements.toolsLogWarn.click();
        elements.toolsLogInfo.click();
        elements.toolsLogDebug.click();
        elements.toolsLogSelectAll.click();

        const lines = Array.from(elements.logOutput.querySelectorAll('.notes-log-line'));
        expect(lines.map((line) => line.dataset.selected)).toEqual([
            'false', 'false', 'false', 'true', 'false',
        ]);
        expect(elements.logOutput.dataset.showError).toBe('false');
        expect(elements.logOutput.dataset.showWarn).toBe('false');
        expect(elements.logOutput.dataset.showInfo).toBe('false');
        expect(elements.logOutput.dataset.showDebug).toBe('true');
        expect(elements.logOutput.dataset.showTrace).toBe('false');
        expect(elements.toolsLogDebug.getAttribute('aria-pressed')).toBe('true');

        elements.toolsLogTrace.click();
        elements.toolsLogSelectAll.click();
        expect(elements.logOutput.dataset.showTrace).toBe('true');
        expect(elements.toolsLogTrace.getAttribute('aria-pressed')).toBe('true');
        expect(lines.map((line) => line.dataset.selected)).toEqual([
            'false', 'false', 'false', 'true', 'true',
        ]);

        elements.toolsLogWordwrap.click();
        elements.toolsLogTimestamp.click();
        expect(elements.toolsLogWordwrap.dataset.enabled).toBe('true');
        expect(elements.toolsLogTimestamp.dataset.enabled).toBe('true');
        expect(elements.toolsLogWordwrap.getAttribute('aria-pressed')).toBe('true');
        expect(elements.toolsLogTimestamp.getAttribute('aria-pressed')).toBe('true');
    });

    it('keeps the Font Awesome maximize glyph while toggling the log panel', () => {
        const elements = createElements();
        const handlers = {};
        const eventsOn = (name, handler) => {
            handlers[name] = handler;
        };

        initNotesLogPanel(elements, eventsOn, 10);
        expect(elements.toolsLogMaximize.textContent).toBe(String.fromCodePoint(0xf065));

        elements.toolsLogMaximize.click();
        expect(elements.toolsLogMaximize.dataset.enabled).toBe('true');
        expect(elements.toolsLogMaximize.textContent).toBe(String.fromCodePoint(0xf065));

        elements.toolsLogMaximize.click();
        expect(elements.toolsLogMaximize.dataset.enabled).toBe('false');
        expect(elements.toolsLogMaximize.textContent).toBe(String.fromCodePoint(0xf065));
    });
});
