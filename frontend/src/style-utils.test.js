import { describe, expect, it } from 'vitest';

import { getAllMarkdownStyles, getSwaggerUIStyles } from './style-utils';

const colors = {
    fg: { Red: 230, Green: 237, Blue: 243 },
    bg: { Red: 30, Green: 34, Blue: 40 },
    green: { Red: 61, Green: 127, Blue: 199 },
    red: { Red: 220, Green: 80, Blue: 80 },
    cyan: { Red: 90, Green: 180, Blue: 220 },
    yellow: { Red: 153, Green: 192, Blue: 211 },
    blueBright: { Red: 140, Green: 170, Blue: 210 },
    magenta: { Red: 180, Green: 100, Blue: 210 },
    link: { Red: 110, Green: 170, Blue: 240 },
    selection: { Red: 49, Green: 109, Blue: 176 },
};

describe('style-utils', () => {
    it('combines markdown base theme fragments', () => {
        const css = getAllMarkdownStyles({ colors, fontSize: 14 }, {
            classPrefix: 'markdown-body',
            includeCheckboxes: true,
        });

        expect(css).toContain('::selection');
        expect(css).toContain('::-webkit-scrollbar');
        expect(css).toContain('.markdown-body h1');
        expect(css).toContain('input[type="checkbox"]');
    });

    it('includes swagger styles for popup-trigger controls', () => {
        const css = getSwaggerUIStyles(colors, 14);

        expect(css).toContain('.swagger-method-selector');
        expect(css).toContain('.swagger-header-dropdown');
        expect(css).toContain('Font Awesome Solid');
        expect(css).toContain('.swagger-endpoint-filter');
    });
});