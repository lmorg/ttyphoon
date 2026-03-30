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
            white-space: pre-wrap;
            word-wrap: break-word;
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

        ${prefix}table {
            width: 100%;
            border-collapse: collapse;
            /*border: 1px solid ${classPrefix ? 'color-mix(in srgb, var(--fg) 22%, transparent)' : `rgba(${colors.fg.Red}, ${colors.fg.Green}, ${colors.fg.Blue}, 0.22)`};*/
        }

        ${prefix}th,
        ${prefix}td {
            border-bottom: 1px solid ${classPrefix ? 'color-mix(in srgb, var(--fg) 18%, transparent)' : `rgba(${colors.fg.Red}, ${colors.fg.Green}, ${colors.fg.Blue}, 0.18)`};
            padding: 4px 8px;
        }

        ${prefix}thead th {
            border-bottom: 1px solid ${classPrefix ? 'color-mix(in srgb, var(--fg) 28%, transparent)' : `rgba(${colors.fg.Red}, ${colors.fg.Green}, ${colors.fg.Blue}, 0.28)`};
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

/**
 * Generate Swagger/OpenAPI UI styles (Postman-like interface)
 * @param {Object} colors - Color palette from GetWindowStyle
 * @param {number} fontSize - Base font size
 * @returns {string} CSS text for Swagger UI styling
 */
export function getSwaggerUIStyles(colors, fontSize) {
    const fgRgb = `${colors.fg.Red}, ${colors.fg.Green}, ${colors.fg.Blue}`;
    const bgRgb = `${colors.bg.Red}, ${colors.bg.Green}, ${colors.bg.Blue}`;
    const greenRgb = `${colors.green.Red}, ${colors.green.Green}, ${colors.green.Blue}`;
    const redRgb = `${colors.red.Red}, ${colors.red.Green}, ${colors.red.Blue}`;
    const cyanRgb = `${colors.cyan.Red}, ${colors.cyan.Green}, ${colors.cyan.Blue}`;
    const yellowRgb = `${colors.yellow.Red}, ${colors.yellow.Green}, ${colors.yellow.Blue}`;
    const blueBrightRgb = `${colors.blueBright.Red}, ${colors.blueBright.Green}, ${colors.blueBright.Blue}`;

    return `
        /* Swagger UI Container */
        .swagger-ui {
            display: flex;
            flex-direction: column;
            min-height: 0;
            gap: 12px;
            color: var(--fg);
            background-color: var(--bg);
        }
            
        .swagger-layout {
            display: flex;
            flex-direction: column;
            gap: 12px;
            min-height: auto;
            height: 100%;
            overflow: auto;
        }

        .swagger-endpoints-pane {
            min-height: 0;
            height: 250px;
            max-height: 250px;
            overflow: auto;
            border: 1px solid rgba(${fgRgb}, 0.2);
            border-radius: 4px;
            background-color: rgba(${fgRgb}, 0.03);
            padding: 8px;
            flex-shrink: 0;
            margin-bottom: 20px;
        }

        /* Spec Info Header */
        .swagger-info {
            padding: 4px 0 8px;
            margin-bottom: 4px;
        }

        .swagger-info-title {
            margin: 0 0 4px;
        }

        .swagger-info-description {
            margin: 0;
        }

        .swagger-info-description p {
            margin: 0 0 6px;
        }

        .swagger-info-description p:last-child {
            margin-bottom: 0;
        }

        .swagger-info-meta {
            display: flex;
            flex-direction: column;
            gap: 6px;
            margin-top: 8px;
        }

        .swagger-info-meta-item {
            display: flex;
            flex-wrap: wrap;
            gap: 8px;
            align-items: baseline;
        }

        .swagger-info-meta-label {
            color: var(--accent);
            font-weight: bold;
            font-size: 0.9em;
        }

        .swagger-info-meta-value {
            color: rgba(${fgRgb}, 0.8);
            min-width: 0;
            word-break: break-word;
        }

        .swagger-endpoints-header {
            font-size: 0.85em;
            font-weight: bold;
            color: var(--accent);
            margin: 2px 0 8px;
            letter-spacing: 0.03em;
        }

        .swagger-endpoint-filter {
            width: 100%;
            box-sizing: border-box;
            border: 1px solid rgba(${fgRgb}, 0.25);
            background: rgba(${bgRgb}, 1);
            color: var(--fg);
            border-radius: 4px;
            padding: 7px 9px;
            margin-bottom: 8px;
            font-size: 0.9em;
            outline: none;
            top: 0;
            position: sticky;
        }

        .swagger-endpoint-filter:focus {
            border-color: var(--accent);
        }

        .swagger-endpoint-filter::placeholder {
            color: rgba(${fgRgb}, 0.55);
        }

        .swagger-main-pane {
            min-height: auto;
            overflow: visible;
            display: flex;
            flex-direction: column;
            gap: 12px;
            padding-right: 4px;
        }

        .swagger-endpoint-sticky {
            position: sticky;
            top: 0;
            z-index: 2;
            background-color: var(--bg);
            padding-bottom: 0px;
            margin-bottom: -10px;
        }

        .swagger-endpoint-heading {
            margin: 0;
            margin-bottom: 10px;
        }

        .swagger-endpoint-title {
            margin: 0;
        }

        /* Empty State */
        .swagger-empty-state {
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 100px;
            color: rgba(${fgRgb}, 0.5);
            font-style: italic;
            top: 0;
            z-index: 1;
        }

        .swagger-empty-field {
            margin: 0;
            color: rgba(${fgRgb}, 0.5);
            font-size: 0.9em;
        }

        /* Method + URL Bar */
        .swagger-method-url-bar {
            display: flex;
            gap: 8px;
            align-items: center;
            padding: 8px;
            background-color: rgba(${fgRgb}, 0.05);
            border: 1px solid rgba(${fgRgb}, 0.2);
            border-radius: 4px;
        }

        .swagger-method-selector {
            padding: 6px 12px;
            border: 1px solid rgba(${fgRgb}, 0.3);
            border-radius: 3px;
            background-color: var(--bg);
            color: var(--fg);
            font-weight: bold;
            cursor: not-allowed;
            opacity: 0.7;
            min-width: 80px;
            font-family: var(--font-family);
        }

        .swagger-url-input {
            flex: 1;
            padding: 6px 12px;
            border: 1px solid rgba(${fgRgb}, 0.2);
            background-color: rgba(${fgRgb}, 0.05);
            color: var(--fg);
            border-radius: 3px;
            font-family: var(--font-family);
            font-size: 0.9em;
        }

        .swagger-send-btn {
            padding: 6px 16px;
            background-color: rgba(${greenRgb}, 0.15);
            color: rgb(${greenRgb});
            border: 1px solid rgb(${greenRgb});
            border-radius: 3px;
            cursor: pointer;
            font-weight: bold;
            font-size: 0.9em;
            transition: background-color 0.2s ease;
        }

        .swagger-send-btn:hover {
            background-color: rgba(${greenRgb}, 0.3);
        }

        .swagger-send-btn:disabled,
        .swagger-send-btn[data-sending="true"] {
            opacity: 0.5;
            cursor: not-allowed;
        }

        .swagger-live-badge {
            font-size: 0.75em;
            font-weight: bold;
            padding: 2px 6px;
            border-radius: 3px;
            background-color: rgba(${greenRgb}, 0.15);
            color: rgb(${greenRgb});
            border: 1px solid rgba(${greenRgb}, 0.4);
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }

        .swagger-status-error {
            background-color: rgba(${redRgb}, 0.2);
            color: rgb(${redRgb});
            border-color: rgba(${redRgb}, 0.4);
        }

        /* Request/Response Tabs */
        .swagger-request-tabs,
        .swagger-response-tabs {
            display: flex;
            gap: 0;
            border-bottom: 1px solid rgba(${fgRgb}, 0.2);
            margin-bottom: 8px;
        }

        .swagger-request-tab,
        .swagger-response-tab {
            padding: 8px 16px;
            border: none;
            background: none;
            color: rgba(${fgRgb}, 0.6);
            cursor: pointer;
            border-bottom: 2px solid transparent;
            transition: all 0.2s ease;
            font-size: 0.95em;
        }

        .swagger-request-tab:hover,
        .swagger-response-tab:hover {
            color: var(--fg);
        }

        .swagger-request-tab[aria-selected="true"],
        .swagger-response-tab[aria-selected="true"] {
            color: var(--accent);
            border-bottom-color: var(--accent);
        }

        /* Tab Panels */
        .swagger-request-panel,
        .swagger-response-panel {
            display: none;
            padding: 8px 0;
        }

        .swagger-request-panel-active,
        .swagger-response-panel-active {
            display: block !important;
        }

        /* Headers List */
        .swagger-headers-list {
            display: flex;
            flex-direction: column;
            gap: 4px;
        }

        .swagger-header-item {
            display: flex;
            gap: 12px;
            padding: 6px 8px;
            background-color: rgba(${fgRgb}, 0.03);
            border-radius: 2px;
            font-family: var(--font-family);
            font-size: 0.85em;
            border-left: 2px solid rgba(${yellowRgb}, 0.3);
        }

        .swagger-header-name {
            font-weight: bold;
            color: var(--accent);
            min-width: 150px;
            flex-shrink: 0;
        }

        .swagger-header-value {
            color: var(--fg);
            flex: 1;
            overflow: auto;
            word-break: break-all;
        }

        .swagger-header-input,
        .swagger-header-select {
            flex: 1;
            padding: 3px 6px;
            border: 1px solid rgba(${fgRgb}, 0.2);
            background-color: rgba(${fgRgb}, 0.05);
            color: var(--fg);
            border-radius: 3px;
            font-family: var(--font-family);
            font-size: 0.85em;
            outline: none;
            min-width: 0;
        }

        .swagger-header-input:focus,
        .swagger-header-select:focus {
            border-color: var(--accent);
            background-color: rgba(${fgRgb}, 0.08);
        }

        /* Body Editor */
        .swagger-body-editor {
            width: 100%;
            min-height: 120px;
            max-height: 300px;
            padding: 8px;
            background-color: rgba(${fgRgb}, 0.02);
            border: 1px solid rgba(${fgRgb}, 0.2);
            border-radius: 3px;
            color: rgb(${greenRgb});
            font-family: var(--font-family);
            font-size: 0.85em;
            resize: vertical;
        }

        /* Parameters Table */
        .swagger-params-table {
            width: 100%;
            border-collapse: collapse;
            font-size: 0.85em;
        }

        .swagger-params-table thead {
            background-color: rgba(${yellowRgb}, 0.1);
            color: var(--accent);
        }

        .swagger-params-table th {
            padding: 6px;
            text-align: left;
            font-weight: bold;
            border-bottom: 1px solid rgba(${fgRgb}, 0.2);
        }

        .swagger-params-table td {
            padding: 6px;
            border-bottom: 1px solid rgba(${fgRgb}, 0.1);
            font-family: var(--font-family);
        }

        .swagger-params-table tr:hover {
            background-color: rgba(${fgRgb}, 0.03);
        }

        /* Response Section */
        .swagger-response-section {
            /*border: 1px solid rgba(${fgRgb}, 0.2);
            border-radius: 4px;
            background-color: rgba(${fgRgb}, 0.03);
            padding: 12px;
            max-height: 300px;
            overflow: auto;*/
        }

        .swagger-response-section .markdown-body h2 {
            margin: 0 0 10px;
        }

        .swagger-response-header {
            display: flex;
            gap: 12px;
            align-items: center;
            margin-bottom: 12px;
            padding-bottom: 8px;
            border-bottom: 1px solid rgba(${fgRgb}, 0.15);
        }

        .swagger-status-badge {
            padding: 4px 12px;
            border-radius: 3px;
            font-weight: bold;
            font-size: 0.9em;
            flex-shrink: 0;
        }

        .swagger-status-2xx {
            background-color: rgba(${greenRgb}, 0.2);
            color: rgb(${greenRgb});
        }

        .swagger-status-3xx {
            background-color: rgba(${cyanRgb}, 0.2);
            color: rgb(${cyanRgb});
        }

        .swagger-status-4xx {
            background-color: rgba(${redRgb}, 0.2);
            color: rgb(${redRgb});
        }

        .swagger-status-5xx {
            background-color: rgba(${redRgb}, 0.2);
            color: rgb(${redRgb});
        }

        .swagger-response-meta {
            font-size: 0.85em;
            color: rgba(${fgRgb}, 0.6);
            font-style: italic;
        }

        /* Response Body Display */
        .swagger-response-body {
            margin: 0;
            padding: 8px;
            background-color: rgba(${fgRgb}, 0.05);
            border: 1px solid rgba(${fgRgb}, 0.15);
            border-radius: 3px;
            color: rgb(${greenRgb});
            font-family: var(--font-family);
            font-size: 0.85em;
            overflow: auto;
            max-height: 200px;
        }

        /* Method Badges for Endpoints List */
        .swagger-method-badge {
            padding: 2px 8px;
            border-radius: 2px;
            font-weight: bold;
            font-size: 0.8em;
            flex-shrink: 0;
            min-width: 40px;
            text-align: center;
        }

        .swagger-method-get {
            background-color: rgba(${blueBrightRgb}, 0.3);
            color: rgb(${blueBrightRgb});
        }

        .swagger-method-post {
            background-color: rgba(${greenRgb}, 0.3);
            color: rgb(${greenRgb});
        }

        .swagger-method-put {
            background-color: rgba(${cyanRgb}, 0.3);
            color: rgb(${cyanRgb});
        }

        .swagger-method-delete {
            background-color: rgba(${redRgb}, 0.3);
            color: rgb(${redRgb});
        }

        .swagger-method-patch {
            background-color: rgba(${yellowRgb}, 0.3);
            color: rgb(${yellowRgb});
        }

        .swagger-method-head,
        .swagger-method-options {
            background-color: rgba(${fgRgb}, 0.2);
            color: var(--fg);
        }

        /* Endpoints List */
        .swagger-endpoints-list {
            display: flex;
            flex-direction: column;
            gap: 4px;
        }

        .swagger-endpoint-item {
            display: flex;
            gap: 12px;
            align-items: center;
            padding: 8px;
            border: 1px solid rgba(${fgRgb}, 0.15);
            border-radius: 3px;
            background-color: rgba(${fgRgb}, 0.02);
            cursor: pointer;
            text-align: left;
            transition: all 0.2s ease;
            width: 100%;
            color: inherit;
        }

        .swagger-endpoint-item:hover {
            background-color: rgba(${fgRgb}, 0.05);
            border-color: rgba(${fgRgb}, 0.25);
        }

        .swagger-endpoint-selected {
            background-color: rgba(${yellowRgb}, 0.1);
            border-color: var(--accent);
        }

        .swagger-endpoint-path {
            font-family: var(--font-family);
            font-weight: bold;
            color: var(--fg);
            flex: 1;
        }

        .swagger-endpoint-summary {
            font-size: 0.85em;
            color: rgba(${fgRgb}, 0.6);
            font-style: italic;
        }

        /* Request Builder */
        .swagger-request-builder {
            display: flex;
            flex-direction: column;
            gap: 12px;
            flex: 0 1 auto;
            padding-right: 8px;
            overflow: visible;
        }

        /* Collapsible Sections */
        .swagger-section-title {
            display: flex;
            align-items: center;
            gap: 8px;
            cursor: pointer;
            color: var(--accent);
            font-weight: bold;
            padding: 8px;
            user-select: none;
            border-radius: 3px;
            transition: background-color 0.2s ease;
        }

        .swagger-section-title:hover {
            background-color: rgba(${yellowRgb}, 0.1);
        }

        .swagger-section-title::before {
            content: '▶';
            transition: transform 0.2s ease;
            display: inline-block;
            flex-shrink: 0;
        }

        .swagger-section-title[data-expanded="true"]::before {
            transform: rotate(90deg);
        }

        .swagger-section-content {
            max-height: 0;
            overflow: hidden;
            transition: max-height 0.3s ease;
        }

        .swagger-section-content[data-expanded="true"] {
            max-height: 1000px;
        }

        /* Parameters Form */
        .swagger-params-form {
            display: flex;
            flex-direction: column;
            gap: 12px;
        }

        .swagger-param-item {
            display: flex;
            flex-direction: column;
            gap: 4px;
        }

        .swagger-param-label {
            display: flex;
            flex-direction: column;
            gap: 2px;
            cursor: pointer;
        }

        .swagger-param-name {
            font-weight: bold;
            color: var(--accent);
            font-size: 0.95em;
        }

        .swagger-param-meta {
            font-size: 0.8em;
            color: rgba(${fgRgb}, 0.5);
            font-family: var(--font-family);
        }

        .swagger-param-input {
            padding: 6px 8px;
            border: 1px solid rgba(${fgRgb}, 0.2);
            background-color: rgba(${fgRgb}, 0.05);
            color: var(--fg);
            border-radius: 3px;
            font-family: var(--font-family);
            font-size: 0.9em;
            transition: border-color 0.2s ease;
        }

        .swagger-param-input:focus {
            outline: none;
            border-color: var(--accent);
            background-color: rgba(${fgRgb}, 0.08);
        }

        .swagger-param-input:required {
            border-left: 3px solid rgb(${blueBrightRgb});
        }

        .swagger-param-description {
            color: rgba(${fgRgb}, 0.5);
            padding: 2px 0 !important;
            margin-top: -16px !important;
        }
    `;
}
