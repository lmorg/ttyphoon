import { beforeEach, describe, expect, it, vi } from 'vitest';

const { eventOnMock, eventOffMock, getAllScreensMock, getScreenMock } = vi.hoisted(() => ({
    eventOnMock: vi.fn(),
    eventOffMock: vi.fn(),
    getAllScreensMock: vi.fn(),
    getScreenMock: vi.fn(),
}));

vi.mock('@wailsio/runtime', () => ({
    Browser: { OpenURL: vi.fn() },
    Clipboard: { SetText: vi.fn() },
    Events: { On: eventOnMock, Off: eventOffMock },
    Screens: { GetAll: getAllScreensMock },
    Window: {
        GetScreen: getScreenMock,
        Position: vi.fn(),
        Size: vi.fn(),
        IsMaximised: vi.fn(),
        Maximise: vi.fn(),
        UnMaximise: vi.fn(),
        SetPosition: vi.fn(),
        SetSize: vi.fn(),
    },
}));

import { EventsOff, EventsOn, ScreenGetAll } from './wails-runtime.js';

describe('Wails v3 runtime compatibility', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('passes event data to existing handlers and returns the unsubscribe function', () => {
        const unsubscribe = vi.fn();
        const callback = vi.fn();
        eventOnMock.mockReturnValue(unsubscribe);

        expect(EventsOn('notesLog', callback)).toBe(unsubscribe);
        const [, handler] = eventOnMock.mock.calls[0];
        handler({ data: 'log entry' });

        expect(callback).toHaveBeenCalledWith('log entry');
    });

    it('normalizes v3 screen geometry and marks the active display', async () => {
        getAllScreensMock.mockResolvedValue([
            {
                ID: 'left',
                WorkArea: { X: -1200, Y: 40, Width: 1200, Height: 760 },
                IsPrimary: false,
            },
            {
                ID: 'main',
                WorkArea: { X: 0, Y: 0, Width: 1920, Height: 1080 },
                IsPrimary: true,
            },
        ]);
        getScreenMock.mockResolvedValue({ ID: 'left' });

        await expect(ScreenGetAll()).resolves.toEqual([
            { x: -1200, y: 40, width: 1200, height: 760, isCurrent: true, isPrimary: false },
            { x: 0, y: 0, width: 1920, height: 1080, isCurrent: false, isPrimary: true },
        ]);
    });

    it('forwards event names to the v3 event manager', () => {
        EventsOff('notesGrepBatch', 'notesGrepDone');

        expect(eventOffMock).toHaveBeenCalledWith('notesGrepBatch', 'notesGrepDone');
    });
});