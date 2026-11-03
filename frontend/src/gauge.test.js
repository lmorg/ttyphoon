import { describe, expect, it, vi } from 'vitest';
import { drawGauge } from './gauge';

const getCellSize = () => ({ cellWidth: 10, cellHeight: 20 });
const foreground = { Red: 210, Green: 220, Blue: 230 };

function createContext() {
    const fills = [];
    const context = {
        globalAlpha: 0.8,
        fillStyle: 'previous',
        save: vi.fn(),
        restore: vi.fn(),
        beginPath: vi.fn(),
        roundRect: vi.fn(),
        fillRect: vi.fn(),
        fill: vi.fn(() => fills.push({ alpha: context.globalAlpha, colour: context.fillStyle })),
    };
    return { context, fills };
}

describe('terminal gauge scrollbar styling', () => {
    it.each([
        ['gauge_h', { width: 12 }, [20, 72, 60, 8, 4]],
        ['gauge_v', { height: 6 }, [22, 60, 8, 60, 4]],
    ])('draws %s with an 8px rounded foreground thumb and no track', (op, size, rectangle) => {
        const { context, fills } = createContext();
        drawGauge(context, getCellSize, {
            op, x: 2, y: 3, value: 50, max: 100,
            fg: { Red: 255, Green: 0, Blue: 0 }, ...size,
        }, foreground);

        expect(context.roundRect).toHaveBeenCalledOnce();
        expect(context.roundRect).toHaveBeenCalledWith(...rectangle);
        expect(context.fillRect).not.toHaveBeenCalled();
        expect(fills).toEqual([{ alpha: 0.5, colour: 'rgb(210, 220, 230)' }]);
        expect(context.save).toHaveBeenCalledOnce();
        expect(context.restore).toHaveBeenCalledOnce();
    });

    it.each([0, -1])('does not paint an empty gauge with value %s', (value) => {
        const { context } = createContext();
        drawGauge(context, getCellSize, { op: 'gauge_v', height: 6, value, max: 100, fg: foreground });
        expect(context.fill).not.toHaveBeenCalled();
        expect(context.fillRect).not.toHaveBeenCalled();
    });

    it('clamps the fill to the track extent', () => {
        const { context } = createContext();
        drawGauge(context, getCellSize, { op: 'gauge_v', x: 0, y: 0, height: 6, value: 200, max: 100, fg: foreground });
        expect(context.roundRect).toHaveBeenCalledWith(2, 0, 8, 120, 4);
    });
});