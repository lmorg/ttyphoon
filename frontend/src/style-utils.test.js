import { describe, expect, it } from 'vitest';

import { getAllMarkdownStyles, getMarkdownContentStyles, getSwaggerUIStyles } from './style-utils';

const colors = {
    fg: { Red: 230, Green: 237, Blue: 243 },
    bg: { Red: 30, Green: 34, Blue: 40 },
    accent: { Red: 120, Green: 180, Blue: 240 },
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
        const regularFontIndex = css.indexOf('Font Awesome Regular');
        const solidFontIndex = css.indexOf('Font Awesome Solid');
        expect(regularFontIndex).toBeGreaterThanOrEqual(0);
        expect(solidFontIndex).toBeGreaterThan(regularFontIndex);
        expect(css).toContain('font-weight: 400;');
        expect(css).toContain('.swagger-endpoint-filter');
    });

    it.each(['', 'markdown-body'])('uses a single grid without collapsed header borders with prefix "%s"', (classPrefix) => {
        const css = getMarkdownContentStyles(colors, 14, classPrefix);
        const prefix = classPrefix ? `.${classPrefix} ` : '';
        const style = document.createElement('style');
        style.textContent = css;
        document.head.appendChild(style);
        const rules = Array.from(style.sheet.cssRules);
        style.remove();
        const tableRule = rules.find((rule) => rule.selectorText === `${prefix}table`);
        const cellRule = rules.find((rule) => {
            const selectors = rule.selectorText.split(',').map((selector) => selector.trim());
            return selectors.length === 2 && selectors.includes(`${prefix}th`) && selectors.includes(`${prefix}td`);
        });
        const headerRule = rules.find((rule) => rule.selectorText === `${prefix}thead th`);
        const lastColumnRule = rules.find((rule) => rule.selectorText === `${prefix}tr > :last-child`);
        const lastRowRule = rules.find((rule) => rule.selectorText.includes('table > :last-child > tr:last-child'));

        expect(tableRule.style.getPropertyValue('border-collapse')).toBe('separate');
        expect(tableRule.style.getPropertyValue('border-spacing')).toBe('0');
        expect(cellRule.style.getPropertyValue('border-width')).toBe('0 1px 1px 0');
        expect(headerRule.style.getPropertyValue('border-color')).toBe('var(--bg)');
        expect(headerRule.style.getPropertyValue('border-width')).toBe('');
        expect(lastColumnRule.style.getPropertyValue('border-right-width')).toBe('0');
        expect(lastRowRule.style.getPropertyValue('border-bottom-width')).toBe('0');
    });
});