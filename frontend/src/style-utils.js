/**
 * Shared CSS styling utilities for consistent theming across markdown views
 */

/**
 * Generate scrollbar styles
 * @param {Object} colors - Color palette from GetWindowStyle
 * @returns {string} CSS text for scrollbar styling
 */
export function getScrollbarStyles(colors) {
    return `
        ::-webkit-scrollbar {
            width: 5px;
            height: 5px;
            background-color: var(--bg);
            opacity: 0.5;
        }
        ::-webkit-scrollbar-track {
            background-color: var(--bg);
        }
        ::-webkit-scrollbar-thumb {
            background-color: var(--fg);
            border-radius: 4px;
        }
        ::-webkit-scrollbar-thumb:hover {
            background-color: var(--accent);
        }
    `;
}

/**
 * Generate selection style
 * @param {Object} colors - Color palette from GetWindowStyle
 * @returns {string} CSS text for selection styling
 */
export function getSelectionStyles(colors) {
    return `
        ::selection {
            background-color: rgb(${colors.selection.Red}, ${colors.selection.Green}, ${colors.selection.Blue});
        }
    `;
}

/**
 * Generate base markdown text size for a root selector
 * @param {string} selector - CSS selector for markdown root container
 * @param {number} fontSize - Base font size
 * @returns {string} CSS text for root font size
 */
export function getMarkdownBaseTextSizeStyles(selector, fontSize) {
    return `
        ${selector} {
            font-size: ${fontSize}px;
        }
    `;
}

/**
 * Generate markdown content styles (headings, links, code, blockquote, etc.)
 * Note: Uses CSS variables and explicit colors for compatibility
 * @param {Object} colors - Color palette from GetWindowStyle
 * @param {number} fontSize - Base font size
 * @param {string} classPrefix - CSS class prefix (e.g., 'markdown-body' or empty for global)
 * @returns {string} CSS text for markdown content styling
 */
export function getMarkdownContentStyles(colors, fontSize, classPrefix = '') {
    const prefix = classPrefix ? `.${classPrefix} ` : '';
    
    return `
        ${prefix}h1,
        ${prefix}h2,
        ${prefix}h3,
        ${prefix}h4,
        ${prefix}h5,
        ${prefix}h6 {
            color: ${classPrefix ? 'var(--accent)' : `rgb(${colors.yellow.Red}, ${colors.yellow.Green}, ${colors.yellow.Blue})`};
        }

        ${prefix}a {
            text-decoration: none;
            color: ${classPrefix ? 'var(--link)' : `rgb(${colors.link.Red}, ${colors.link.Green}, ${colors.link.Blue})`};
        }

        ${prefix}a:hover {
            text-decoration: underline;
        }

        ${prefix}pre,
        ${prefix}code {
            color: ${classPrefix ? 'var(--green)' : `rgb(${colors.green.Red}, ${colors.green.Green}, ${colors.green.Blue})`};
        }

        ${prefix}pre {
            border: 0;
            border-left: 2px solid ${classPrefix ? 'var(--green)' : `rgb(${colors.green.Red}, ${colors.green.Green}, ${colors.green.Blue})`};
            margin: 0;
            padding: 10px 10px 10px 20px;
            overflow-x: auto;
        }

        ${prefix}blockquote {
            border: 0;
            border-left: 2px solid ${classPrefix ? 'var(--magenta)' : `rgb(${colors.magenta.Red}, ${colors.magenta.Green}, ${colors.magenta.Blue})`};
            margin: 0;
            padding: 1px 1px 1px 20px;
            color: ${classPrefix ? 'var(--magenta)' : `rgb(${colors.magenta.Red}, ${colors.magenta.Green}, ${colors.magenta.Blue})`};
        }

        ${prefix}details {
            opacity: 0.5;
            width: 100%;
            border-radius: 0;
            border-width: 2px;
            border-style: solid;
            padding: 5px;
            margin-top: 5px;
        }

        ${prefix}summary {
            cursor: pointer;
        }
    `;
}

/**
 * Generate Highlight.js syntax highlighting theme
 * Note: Works with both CSS variables (for notes.js) and explicit colors (for markdown.js)
 * @param {Object} colors - Color palette from GetWindowStyle
 * @param {boolean} useCssVars - Whether to use CSS variables or explicit colors
 * @returns {string} CSS text for syntax highlighting
 */
export function getHighlightJsTheme(colors, useCssVars = true) {
    const fg = useCssVars ? 'var(--fg)' : `rgb(${colors.fg.Red}, ${colors.fg.Green}, ${colors.fg.Blue})`;
    const blueBright = useCssVars ? 'var(--blue-bright)' : `rgb(${colors.blueBright.Red}, ${colors.blueBright.Green}, ${colors.blueBright.Blue})`;
    const magenta = useCssVars ? 'var(--magenta)' : `rgb(${colors.magenta.Red}, ${colors.magenta.Green}, ${colors.magenta.Blue})`;
    const green = useCssVars ? 'var(--green)' : `rgb(${colors.green.Red}, ${colors.green.Green}, ${colors.green.Blue})`;
    const yellow = useCssVars ? 'var(--accent)' : `rgb(${colors.yellow.Red}, ${colors.yellow.Green}, ${colors.yellow.Blue})`;
    const cyan = useCssVars ? 'var(--cyan)' : `rgb(${colors.cyan.Red}, ${colors.cyan.Green}, ${colors.cyan.Blue})`;
    const red = useCssVars ? 'var(--red)' : `rgb(${colors.red.Red}, ${colors.red.Green}, ${colors.red.Blue})`;

    return `
        pre code.hljs {
            display: block;
            overflow-x: auto;
            background: transparent;
            color: ${fg};
        }

        .hljs-comment,
        .hljs-quote {
            color: ${blueBright};
            font-style: italic;
        }

        .hljs-keyword,
        .hljs-selector-tag,
        .hljs-subst {
            color: ${magenta};
            font-weight: bold;
        }

        .hljs-string,
        .hljs-title,
        .hljs-name,
        .hljs-type,
        .hljs-attribute,
        .hljs-symbol,
        .hljs-bullet,
        .hljs-addition,
        .hljs-built_in {
            color: ${green};
        }

        .hljs-number,
        .hljs-literal,
        .hljs-variable,
        .hljs-template-variable {
            color: ${yellow};
        }

        .hljs-section,
        .hljs-meta,
        .hljs-function,
        .hljs-class,
        .hljs-title.class_ {
            color: ${cyan};
        }

        .hljs-deletion,
        .hljs-regexp,
        .hljs-link {
            color: ${red};
        }

        .hljs-punctuation,
        .hljs-tag {
            color: ${fg};
        }
    `;
}

/**
 * Generate checkbox styles for interactive markdown checkboxes
 * @param {Object} colors - Color palette from GetWindowStyle
 * @param {number} fontSize - Base font size for checkbox sizing
 * @param {string} classPrefix - CSS class prefix (e.g., 'markdown-body')
 * @returns {string} CSS text for checkbox styling
 */
export function getCheckboxStyles(colors, fontSize, classPrefix = 'markdown-body') {
    const prefix = classPrefix ? `.${classPrefix} ` : '';
    
    return `
        ${prefix}input[type="checkbox"] {
            appearance: none;
            -webkit-appearance: none;
            -moz-appearance: none;
            cursor: pointer;
            margin-right: 6px;
            width: ${fontSize}px;
            height: ${fontSize}px;
            border: 2px solid var(--red);
            background: transparent;
            position: relative;
            vertical-align: middle;
            flex-shrink: 0;
        }

        ${prefix}input[type="checkbox"]:hover {
            border-color: var(--accent);
        }

        ${prefix}input[type="checkbox"]:checked:hover {
            border-color: var(--accent);
            background: var(--accent);
        }

        ${prefix}input[type="checkbox"]:checked {
            background: var(--green);
            border-color: var(--green);
        }

        ${prefix}input[type="checkbox"]:checked::after {
            content: '✓';
            position: absolute;
            color: var(--bg);
            font-size: ${fontSize}px;
            font-weight: bold;
            left: 50%;
            top: 50%;
            transform: translate(-50%, -50%);
        }
    `;
}

/**
 * Generate all common markdown styles in one call
 * @param {Object} windowStyleResult - Complete result from GetWindowStyle()
 * @param {Object} options - Configuration options
 * @param {string} options.classPrefix - CSS class prefix for scoped styles
 * @param {boolean} options.useCssVars - Whether to use CSS variables
 * @param {boolean} options.includeCheckboxes - Whether to include checkbox styles
 * @returns {string} Complete CSS text for all markdown styles
 */
export function getAllMarkdownStyles(windowStyleResult, options = {}) {
    const {
        classPrefix = '',
        useCssVars = true,
        includeCheckboxes = false
    } = options;

    const { colors, fontSize } = windowStyleResult;

    let styles = '';
    styles += getSelectionStyles(colors);
    styles += getScrollbarStyles(colors);
    styles += getMarkdownContentStyles(colors, fontSize, classPrefix);
    styles += getHighlightJsTheme(colors, useCssVars);
    
    if (includeCheckboxes) {
        styles += getCheckboxStyles(colors, fontSize, classPrefix);
    }

    return styles;
}
