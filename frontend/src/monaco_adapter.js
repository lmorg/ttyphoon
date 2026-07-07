import { showSpellcheckSuggestionsPopup } from './spellcheck';
import { MODE_LABELS } from './vim-mode';

let monacoModulePromise = null;
const HLJS_FALLBACK_PREFIX = 'hljs-fallback-';

const HLJS_STATE = {
    clone() {
        return this;
    },
    equals(other) {
        return other === this;
    },
};

// Each entry eagerly registers a Monarch grammar so syntax highlighting works
// deterministically. We deliberately avoid monaco's basic-languages lazy
// `_.contribution` mechanism: in this version it only *defines* registerLanguage
// (registering zero languages unless editor.main is pulled in), and its lazy
// grammar loaders are fragile under our Vite bundle. Registering grammars
// ourselves runs entirely on the main thread — no web workers, no conflict with
// our LSP integration.
const GRAMMAR_LOADERS = [
    { ids: ['go'], load: () => import('monaco-editor/esm/vs/basic-languages/go/go.js') },
    { ids: ['hcl'], load: () => import('monaco-editor/esm/vs/basic-languages/hcl/hcl.js') },
    { ids: ['python'], load: () => import('monaco-editor/esm/vs/basic-languages/python/python.js') },
    { ids: ['rust'], load: () => import('monaco-editor/esm/vs/basic-languages/rust/rust.js') },
    { ids: ['c', 'cpp'], load: () => import('monaco-editor/esm/vs/basic-languages/cpp/cpp.js') },
    { ids: ['csharp'], load: () => import('monaco-editor/esm/vs/basic-languages/csharp/csharp.js') },
    { ids: ['fsharp'], load: () => import('monaco-editor/esm/vs/basic-languages/fsharp/fsharp.js') },
    { ids: ['java'], load: () => import('monaco-editor/esm/vs/basic-languages/java/java.js') },
    { ids: ['kotlin'], load: () => import('monaco-editor/esm/vs/basic-languages/kotlin/kotlin.js') },
    { ids: ['swift'], load: () => import('monaco-editor/esm/vs/basic-languages/swift/swift.js') },
    { ids: ['php'], load: () => import('monaco-editor/esm/vs/basic-languages/php/php.js') },
    { ids: ['ruby'], load: () => import('monaco-editor/esm/vs/basic-languages/ruby/ruby.js') },
    { ids: ['perl'], load: () => import('monaco-editor/esm/vs/basic-languages/perl/perl.js') },
    { ids: ['pascal'], load: () => import('monaco-editor/esm/vs/basic-languages/pascal/pascal.js') },
    { ids: ['clojure'], load: () => import('monaco-editor/esm/vs/basic-languages/clojure/clojure.js') },
    { ids: ['elixir'], load: () => import('monaco-editor/esm/vs/basic-languages/elixir/elixir.js') },
    { ids: ['julia'], load: () => import('monaco-editor/esm/vs/basic-languages/julia/julia.js') },
    { ids: ['lua'], load: () => import('monaco-editor/esm/vs/basic-languages/lua/lua.js') },
    { ids: ['r'], load: () => import('monaco-editor/esm/vs/basic-languages/r/r.js') },
    { ids: ['scala'], load: () => import('monaco-editor/esm/vs/basic-languages/scala/scala.js') },
    { ids: ['scheme'], load: () => import('monaco-editor/esm/vs/basic-languages/scheme/scheme.js') },
    { ids: ['st'], load: () => import('monaco-editor/esm/vs/basic-languages/st/st.js') },
    { ids: ['dart'], load: () => import('monaco-editor/esm/vs/basic-languages/dart/dart.js') },
    { ids: ['tcl'], load: () => import('monaco-editor/esm/vs/basic-languages/tcl/tcl.js') },
    { ids: ['vb'], load: () => import('monaco-editor/esm/vs/basic-languages/vb/vb.js') },
    { ids: ['objective-c'], load: () => import('monaco-editor/esm/vs/basic-languages/objective-c/objective-c.js') },
    { ids: ['shell'], load: () => import('monaco-editor/esm/vs/basic-languages/shell/shell.js') },
    { ids: ['powershell'], load: () => import('monaco-editor/esm/vs/basic-languages/powershell/powershell.js') },
    { ids: ['yaml'], load: () => import('monaco-editor/esm/vs/basic-languages/yaml/yaml.js') },
    { ids: ['ini'], load: () => import('monaco-editor/esm/vs/basic-languages/ini/ini.js') },
    { ids: ['sql'], load: () => import('monaco-editor/esm/vs/basic-languages/sql/sql.js') },
    { ids: ['markdown'], load: () => import('monaco-editor/esm/vs/basic-languages/markdown/markdown.js') },
    { ids: ['html'], load: () => import('monaco-editor/esm/vs/basic-languages/html/html.js') },
    { ids: ['xml'], load: () => import('monaco-editor/esm/vs/basic-languages/xml/xml.js') },
    { ids: ['css'], load: () => import('monaco-editor/esm/vs/basic-languages/css/css.js') },
    { ids: ['scss'], load: () => import('monaco-editor/esm/vs/basic-languages/scss/scss.js') },
    { ids: ['less'], load: () => import('monaco-editor/esm/vs/basic-languages/less/less.js') },
    { ids: ['dockerfile'], load: () => import('monaco-editor/esm/vs/basic-languages/dockerfile/dockerfile.js') },
    { ids: ['javascript'], load: () => import('monaco-editor/esm/vs/basic-languages/javascript/javascript.js') },
    { ids: ['typescript'], load: () => import('monaco-editor/esm/vs/basic-languages/typescript/typescript.js') },
];

// Minimal main-thread JSON grammar. Monaco's real JSON support lives in a
// worker-backed language service (vs/language/json); we only need colours here,
// so a small Monarch tokenizer keeps it worker-free.
function registerJsonGrammar(monaco, registeredIds) {
    if (!registeredIds.has('json')) {
        monaco.languages.register({ id: 'json', extensions: ['.json', '.jsonc'] });
    }
    monaco.languages.setLanguageConfiguration('json', {
        comments: { lineComment: '//', blockComment: ['/*', '*/'] },
        brackets: [['{', '}'], ['[', ']']],
        autoClosingPairs: [
            { open: '{', close: '}' },
            { open: '[', close: ']' },
            { open: '"', close: '"' },
        ],
    });
    monaco.languages.setMonarchTokensProvider('json', {
        tokenizer: {
            root: [
                [/"(?:[^"\\]|\\.)*"(?=\s*:)/, 'type'],
                [/"(?:[^"\\]|\\.)*"/, 'string'],
                [/\b(?:true|false|null)\b/, 'keyword'],
                [/-?\d+(?:\.\d+)?(?:[eE][-+]?\d+)?/, 'number'],
                [/[{}[\],:]/, 'delimiter'],
                [/\/\/.*$/, 'comment'],
                [/\/\*/, 'comment', '@comment'],
            ],
            comment: [
                [/[^/*]+/, 'comment'],
                [/\*\//, 'comment', '@pop'],
                [/[/*]/, 'comment'],
            ],
        },
    });
}

async function registerLanguageGrammars(monaco) {
    const registeredIds = new Set(monaco.languages.getLanguages().map((lang) => lang.id));

    await Promise.all(GRAMMAR_LOADERS.map(async ({ ids, load }) => {
        let mod;
        try {
            mod = await load();
        } catch (err) {
            console.error('Failed to load Monaco grammar', ids, err);
            return;
        }

        for (const id of ids) {
            if (!registeredIds.has(id)) {
                monaco.languages.register({ id });
                registeredIds.add(id);
            }
            if (mod.language) {
                monaco.languages.setMonarchTokensProvider(id, mod.language);
            }
            if (mod.conf) {
                monaco.languages.setLanguageConfiguration(id, mod.conf);
            }
        }
    }));

    registerJsonGrammar(monaco, registeredIds);
}

function loadMonacoModule() {
    if (!monacoModulePromise) {
        monacoModulePromise = import('monaco-editor/esm/vs/editor/editor.api')
            .then(async (monaco) => {
                await registerLanguageGrammars(monaco);
                return monaco;
            });
    }

    return monacoModulePromise;
}

function toMonacoLanguage(language) {
    const value = String(language || '').trim().toLowerCase();
    if (!value) {
        return 'plaintext';
    }

    const map = {
        plaintext: 'plaintext',
        text: 'plaintext',
        javascript: 'javascript',
        typescript: 'typescript',
        python: 'python',
        go: 'go',
        rust: 'rust',
        c: 'c',
        cpp: 'cpp',
        csharp: 'csharp',
        fsharp: 'fsharp',
        java: 'java',
        kotlin: 'kotlin',
        swift: 'swift',
        php: 'php',
        ruby: 'ruby',
        perl: 'perl',
        pascal: 'pascal',
        clojure: 'clojure',
        elixir: 'elixir',
        julia: 'julia',
        lua: 'lua',
        r: 'r',
        scala: 'scala',
        scheme: 'scheme',
        st: 'st',
        dart: 'dart',
        tcl: 'tcl',
        vb: 'vb',
        objectivec: 'objective-c',
        'objective-c': 'objective-c',
        bash: 'shell',
        powershell: 'powershell',
        json: 'json',
        yaml: 'yaml',
        toml: 'ini',
        ini: 'ini',
        sql: 'sql',
        terraform: 'hcl',
        murex: 'shell',
        markdown: 'markdown',
        html: 'html',
        xml: 'xml',
        css: 'css',
        scss: 'scss',
        less: 'less',
        dockerfile: 'dockerfile',
    };

    return map[value] || value || 'plaintext';
}

function toHighlightJsLanguage(language, highlightJs) {
    if (!highlightJs || typeof highlightJs.getLanguage !== 'function') {
        return '';
    }

    const value = String(language || '').trim().toLowerCase();
    if (!value) {
        return '';
    }

    const map = {
        'objective-c': 'objectivec',
        objectivec: 'objectivec',
        makefile: 'makefile',
        nu: 'bash',
    };

    const candidate = map[value] || value;
    if (highlightJs.getLanguage(candidate)) {
        return candidate;
    }

    return '';
}

function mapHljsClassesToMonacoToken(classes) {
    const normalized = classes.map((entry) => String(entry || '').replace(/^hljs-/, '').toLowerCase());

    if (normalized.some((entry) => entry === 'comment' || entry === 'quote')) {
        return 'comment';
    }
    if (normalized.some((entry) => entry === 'string' || entry === 'regexp')) {
        return 'string';
    }
    if (normalized.some((entry) => entry === 'number')) {
        return 'number';
    }
    if (normalized.some((entry) => entry === 'keyword' || entry === 'built_in' || entry === 'builtin' || entry === 'literal')) {
        return 'keyword';
    }
    if (normalized.some((entry) => entry === 'type' || entry === 'class' || entry === 'title.class' || entry === 'title.class_')) {
        return 'type';
    }

    return '';
}

function createHighlightJsFallbackProvider(highlightJs, hljsLanguage) {
    return {
        getInitialState() {
            return HLJS_STATE;
        },
        tokenize(line) {
            const source = String(line || '');
            if (!source) {
                return { tokens: [], endState: HLJS_STATE };
            }

            let html;
            try {
                html = highlightJs.highlight(source, { language: hljsLanguage, ignoreIllegals: true }).value;
            } catch {
                return { tokens: [], endState: HLJS_STATE };
            }

            const root = document.createElement('span');
            root.innerHTML = html;

            const segments = [];
            const walk = (node, classStack) => {
                if (!node) {
                    return;
                }

                if (node.nodeType === 3) {
                    const text = String(node.nodeValue || '');
                    if (text) {
                        segments.push({ text, classes: classStack });
                    }
                    return;
                }

                if (node.nodeType !== 1) {
                    return;
                }

                const ownClasses = String(node.getAttribute('class') || '').trim().split(/\s+/).filter(Boolean);
                const nextStack = ownClasses.length > 0 ? classStack.concat(ownClasses) : classStack;
                for (const child of node.childNodes) {
                    walk(child, nextStack);
                }
            };

            for (const child of root.childNodes) {
                walk(child, []);
            }

            const tokens = [];
            let offset = 0;
            for (const segment of segments) {
                const length = segment.text.length;
                if (length <= 0) {
                    continue;
                }

                const token = mapHljsClassesToMonacoToken(segment.classes);
                if (token) {
                    const prev = tokens.length > 0 ? tokens[tokens.length - 1] : null;
                    if (!prev || prev.scopes !== token || prev.startIndex !== offset) {
                        tokens.push({ startIndex: offset, scopes: token });
                    }
                }

                offset += length;
            }

            return { tokens, endState: HLJS_STATE };
        },
    };
}

function parseRgb(color, fallback = { Red: 0, Green: 0, Blue: 0 }) {
    const source = color && typeof color === 'object' ? color : fallback;
    return {
        r: Math.max(0, Math.min(255, Number(source.Red) || 0)),
        g: Math.max(0, Math.min(255, Number(source.Green) || 0)),
        b: Math.max(0, Math.min(255, Number(source.Blue) || 0)),
    };
}

function rgbToHex({ r, g, b }) {
    const toHex = (n) => n.toString(16).padStart(2, '0');
    return `${toHex(r)}${toHex(g)}${toHex(b)}`;
}

function withAlphaHex(rgb, alpha = 1) {
    const base = rgbToHex(rgb);
    const a = Math.max(0, Math.min(255, Math.round((Number(alpha) || 0) * 255))).toString(16).padStart(2, '0');
    return `${base}${a}`;
}

function mix(a, b, ratio) {
    const t = Math.max(0, Math.min(1, Number(ratio) || 0));
    return {
        r: Math.round((a.r * (1 - t)) + (b.r * t)),
        g: Math.round((a.g * (1 - t)) + (b.g * t)),
        b: Math.round((a.b * (1 - t)) + (b.b * t)),
    };
}

function luminance({ r, g, b }) {
    return (0.2126 * r) + (0.7152 * g) + (0.0722 * b);
}

function toFontMetrics(options = {}) {
    const fontSize = Number(options.fontSize);
    const safeFontSize = Number.isFinite(fontSize) && fontSize > 0 ? fontSize : 13;
    const lineHeight = Number(options.lineHeight);

    return {
        fontFamily: String(options.fontFamily || '').trim() || undefined,
        fontSize: safeFontSize,
        lineHeight: Number.isFinite(lineHeight) && lineHeight > 0
            ? Math.round(lineHeight)
            : Math.round(safeFontSize * 1.4),
    };
}

function createVimModeIndicator() {
    const el = document.createElement('div');
    el.className = 'vim-mode-indicator';
    el.setAttribute('aria-live', 'polite');
    el.setAttribute('aria-atomic', 'true');
    el.style.cssText = [
        'position:fixed',
        'padding:2px 8px',
        'border-radius:3px',
        'font-size:0.75em',
        'font-family:monospace',
        'font-weight:bold',
        'pointer-events:none',
        'user-select:none',
        'z-index:10000',
        'opacity:0',
        'transition:opacity 0.1s',
        'background:var(--bg,#1e2228)',
        'color:var(--accent,#588acc)',
        'border:1px solid var(--accent,#588acc)',
        'white-space:nowrap',
    ].join(';');
    document.body.appendChild(el);
    return el;
}

function pulseAlpha(now = performance.now()) {
    const phase = (now % 1000) / 1000;
    return 0.1 + (0.7 * (0.5 + 0.5 * Math.sin(phase * Math.PI * 2)));
}

function normalizeVimModeLabel(mode, subMode) {
    const vimMode = String(mode || '').toLowerCase();
    const vimSubMode = String(subMode || '').toLowerCase();
    if (vimMode === 'insert') {
        return MODE_LABELS.insert || '';
    }
    if (vimMode === 'replace') {
        return MODE_LABELS.replace || '-- REPLACE --';
    }
    if (vimMode === 'normal') {
        if (vimSubMode.includes('replace')) {
            return MODE_LABELS['replace-once'] || '-- REPLACE (r) --';
        }
        return MODE_LABELS.normal || '-- VIM KEYS --';
    }
    // Visual and other modal states still indicate that Vim keys are active.
    if (vimMode) {
        return MODE_LABELS.normal || '-- VIM KEYS --';
    }
    return '';
}

export async function createMonacoAdapter(container, options = {}) {
    if (!container) {
        return null;
    }

    const monaco = await loadMonacoModule();
    const highlightJs = options.highlightJs && typeof options.highlightJs.highlight === 'function'
        ? options.highlightJs
        : null;
    const registeredLanguageIds = new Set(monaco.languages.getLanguages().map((lang) => lang.id));
    const fallbackProviderDisposables = [];

    function ensureHighlightJsFallbackLanguage(language) {
        const hljsLanguage = toHighlightJsLanguage(language, highlightJs);
        if (!hljsLanguage) {
            return '';
        }

        const fallbackId = `${HLJS_FALLBACK_PREFIX}${hljsLanguage}`;
        if (registeredLanguageIds.has(fallbackId)) {
            return fallbackId;
        }

        monaco.languages.register({ id: fallbackId });
        registeredLanguageIds.add(fallbackId);
        fallbackProviderDisposables.push(
            monaco.languages.setTokensProvider(
                fallbackId,
                createHighlightJsFallbackProvider(highlightJs, hljsLanguage),
            ),
        );

        return fallbackId;
    }

    function resolveMonacoLanguage(language) {
        const requested = toMonacoLanguage(language || 'plaintext');
        if (registeredLanguageIds.has(requested)) {
            return requested;
        }

        const fallback = ensureHighlightJsFallbackLanguage(requested)
            || ensureHighlightJsFallbackLanguage(language);
        if (fallback) {
            return fallback;
        }

        return 'plaintext';
    }

    const initialValue = String(options.value || '');
    const initialLanguage = resolveMonacoLanguage(options.language || 'plaintext');
    const fontMetrics = toFontMetrics(options);

    const model = monaco.editor.createModel(initialValue, initialLanguage);
    const themeName = 'ttyphoon-monaco';
    let vimAdapter = null;
    let vimIndicator = null;
    let currentVimLabel = '';
    let currentVimMode = '';
    let currentVimSubMode = '';
    let vimReplaceOncePending = false;
    let vimAdapterDisposing = false;
    let vimCursorPulseRaf = 0;

    function applyTheme(colors = {}) {
        const bg = parseRgb(colors.bg, { Red: 0, Green: 0, Blue: 0 });
        const fg = parseRgb(colors.fg, { Red: 225, Green: 225, Blue: 225 });
        const accent = parseRgb(colors.accent, fg);
        const selection = parseRgb(colors.selection, mix(bg, fg, 0.28));
        const lineNo = mix(fg, bg, 0.3);
        const cursorSource = (colors && typeof colors.cursor === 'object')
            ? colors.cursor
            : mix(accent, fg, 0.25);
        const cursor = parseRgb(cursorSource, mix(accent, fg, 0.25));
        const darkBase = luminance(bg) < 128;

        const bgHex = rgbToHex(bg);
        const fgHex = rgbToHex(fg);
        const accentHex = rgbToHex(accent);
        const lineNoHex = rgbToHex(lineNo);
        const cursorHex = rgbToHex(cursor);
        const selectionHex = rgbToHex(selection);

        monaco.editor.defineTheme(themeName, {
            base: darkBase ? 'vs-dark' : 'vs',
            inherit: true,
            rules: [
                { token: 'comment', foreground: rgbToHex(mix(fg, bg, 0.45)) },
                { token: 'keyword', foreground: rgbToHex(accent) },
                { token: 'string', foreground: rgbToHex(parseRgb(colors.green, mix(fg, bg, 0.2))) },
                { token: 'number', foreground: rgbToHex(parseRgb(colors.cyan, mix(fg, bg, 0.15))) },
                { token: 'type', foreground: rgbToHex(parseRgb(colors.blue, accent)) },
            ],
            colors: {
                'editor.background': `#${bgHex}`,
                'editor.foreground': `#${fgHex}`,
                'editorCursor.foreground': `#${cursorHex}`,
                'editor.lineHighlightBackground': `#${withAlphaHex(lineNo, 0.12)}`,
                'editorLineNumber.foreground': `#${lineNoHex}`,
                'editorLineNumber.activeForeground': `#${fgHex}`,
                'editor.selectionBackground': `#${withAlphaHex(selection, 0.45)}`,
                'editor.inactiveSelectionBackground': `#${withAlphaHex(selection, 0.25)}`,
                'editorWidget.background': `#${bgHex}`,
                'editorWidget.foreground': `#${fgHex}`,
                'editorWidget.border': `#${withAlphaHex(fg, 0.24)}`,
                'editorHoverWidget.background': `#${bgHex}`,
                'editorHoverWidget.foreground': `#${fgHex}`,
                'editorSuggestWidget.background': `#${bgHex}`,
                'editorSuggestWidget.foreground': `#${fgHex}`,
                'editorSuggestWidget.selectedBackground': `#${withAlphaHex(selection, 0.35)}`,
                'input.background': `#${bgHex}`,
                'input.foreground': `#${fgHex}`,
                'input.border': `#${withAlphaHex(fg, 0.24)}`,
                'list.hoverBackground': `#${withAlphaHex(selection, 0.22)}`,
                'focusBorder': `#${withAlphaHex(accent, 0.7)}`,
                'editorSuggestWidget.highlightForeground': `#${accentHex}`,
                'textLink.foreground': `#${accentHex}`,
            },
        });

        monaco.editor.setTheme(themeName);
    }

    if (options.themeColors && typeof options.themeColors === 'object') {
        applyTheme(options.themeColors);
    }
    const editor = monaco.editor.create(container, {
        model,
        automaticLayout: true,
        contextmenu: false,
        lightbulb: { enabled: false },
        wordWrap: 'off',
        cursorBlinking: 'blink',
        hover: { enabled: false },
        minimap: { enabled: false },
        overviewRulerLanes: 0,
        hideCursorInOverviewRuler: true,
        scrollbar: {
            vertical: 'visible',
            horizontal: 'auto',
            alwaysConsumeMouseWheel: false,
            verticalScrollbarSize: 5,
            horizontalScrollbarSize: 5,
        },
        fontFamily: fontMetrics.fontFamily,
        fontSize: fontMetrics.fontSize,
        lineHeight: fontMetrics.lineHeight,
        smoothScrolling: true,
        tabSize: 4,
        insertSpaces: true,
    });

    function setMonacoVimActiveClass(enabled) {
        const root = editor.getDomNode();
        if (!root) {
            return;
        }

        if (enabled) {
            root.classList.add('notes-monaco-vim-active');
            return;
        }

        root.classList.remove('notes-monaco-vim-active');
    }

    function stopMonacoVimCursorPulse() {
        if (vimCursorPulseRaf !== 0) {
            cancelAnimationFrame(vimCursorPulseRaf);
            vimCursorPulseRaf = 0;
        }
    }

    function startMonacoVimCursorPulse() {
        if (vimCursorPulseRaf !== 0) {
            return;
        }

        const tick = () => {
            const root = editor.getDomNode();
            if (!root || !vimAdapter) {
                vimCursorPulseRaf = 0;
                return;
            }

            const alpha = String(pulseAlpha());
            const cursors = root.querySelectorAll('.cursors-layer .cursor, .cursors-layer .cursor-secondary');
            for (const cursor of cursors) {
                cursor.style.opacity = alpha;
            }

            vimCursorPulseRaf = requestAnimationFrame(tick);
        };

        vimCursorPulseRaf = requestAnimationFrame(tick);
    }

    function isMonacoOverlayOpen() {
        const root = editor.getDomNode();
        if (!root) {
            return false;
        }

        return Boolean(
            root.querySelector('.suggest-widget.visible')
            || root.querySelector('.parameter-hints-widget.visible')
            || root.querySelector('.find-widget.visible')
            || root.querySelector('.rename-box')
            || root.querySelector('.editor-widget.visible')
            || root.querySelector('.monaco-hover.visible'),
        );
    }

    function positionVimIndicator() {
        if (!vimIndicator || currentVimLabel === '') {
            return;
        }

        const root = editor.getDomNode();
        const position = editor.getPosition();
        if (!root || !position) {
            return;
        }

        const caret = editor.getScrolledVisiblePosition(position);
        const rect = root.getBoundingClientRect();
        if (!caret || rect.width <= 0 || rect.height <= 0) {
            return;
        }

        const badgeW = vimIndicator.offsetWidth || 120;
        const badgeH = vimIndicator.offsetHeight || 20;

        const rawLeft = rect.left + caret.left;
        const rawTop = rect.top + caret.top + caret.height + 2;

        const left = Math.min(Math.max(rawLeft, rect.left), rect.right - badgeW);
        const top = rawTop + badgeH > rect.bottom
            ? (rect.top + caret.top) - badgeH - 2
            : rawTop;

        vimIndicator.style.left = `${Math.round(left)}px`;
        vimIndicator.style.top = `${Math.round(top)}px`;
    }

    function updateVimIndicator(mode, subMode) {
        currentVimMode = String(mode || '').toLowerCase();
        currentVimSubMode = String(subMode || '').toLowerCase();
        const label = normalizeVimModeLabel(mode, subMode);
        currentVimLabel = label;
        if (!vimIndicator) {
            return;
        }
        vimIndicator.textContent = label;
        vimIndicator.style.opacity = label ? '1' : '0';
        if (label) {
            positionVimIndicator();
        }
    }

    function applyVimCursorStyle(mode = currentVimMode) {
        const vimMode = String(mode || '').toLowerCase();
        editor.updateOptions({
            cursorStyle: vimMode === 'insert' ? 'line' : 'block',
            cursorBlinking: 'solid',
        });
    }

    async function enableVimMode() {
        if (vimAdapter || vimAdapterDisposing) {
            return;
        }

        vimIndicator = vimIndicator || createVimModeIndicator();

        try {
            const monacoVim = await import('monaco-vim');
            if (vimAdapter || vimAdapterDisposing) {
                return;
            }

            vimAdapter = monacoVim.initVimMode(editor, null);
            setMonacoVimActiveClass(true);
            startMonacoVimCursorPulse();
            vimAdapter.on?.('vim-mode-change', (event) => {
                vimReplaceOncePending = false;
                updateVimIndicator(event?.mode, event?.subMode);
                applyVimCursorStyle(event?.mode);
            });
            vimAdapter.on?.('vim-keypress', (key) => {
                if (String(key || '') === 'r' && currentVimMode === 'normal') {
                    vimReplaceOncePending = true;
                    updateVimIndicator('normal', 'replace-once');
                    applyVimCursorStyle('normal');
                }
            });
            vimAdapter.on?.('vim-command-done', () => {
                if (vimReplaceOncePending && currentVimMode === 'normal') {
                    vimReplaceOncePending = false;
                    updateVimIndicator('normal', currentVimSubMode);
                }
                applyVimCursorStyle(currentVimMode);
            });
            updateVimIndicator('normal', '');
            applyVimCursorStyle('normal');
        } catch (err) {
            console.error('Failed to enable Monaco Vim mode', err);
            stopMonacoVimCursorPulse();
            if (vimIndicator) {
                vimIndicator.remove();
                vimIndicator = null;
            }
            currentVimLabel = '';
            currentVimMode = '';
            currentVimSubMode = '';
            vimReplaceOncePending = false;
        }
    }

    const changeDisposable = editor.onDidChangeModelContent(() => {
        if (typeof options.onChange === 'function') {
            options.onChange(editor.getValue());
        }
    });

    const cursorDisposable = editor.onDidChangeCursorSelection(() => {
        positionVimIndicator();
        if (typeof options.onSelectionChange === 'function') {
            const selection = editor.getSelection();
            if (!selection) {
                return;
            }
            const start = model.getOffsetAt(selection.getStartPosition());
            const end = model.getOffsetAt(selection.getEndPosition());
            options.onSelectionChange(start, end);
        }
    });

    const mouseMoveDisposable = editor.onMouseMove((event) => {
        const clientX = Number(event?.event?.browserEvent?.clientX);
        const clientY = Number(event?.event?.browserEvent?.clientY);
        if (!Number.isFinite(clientX) || !Number.isFinite(clientY)) {
            return;
        }
        if (typeof options.onPointerMove === 'function') {
            options.onPointerMove(clientX, clientY);
        }
    });

    const keyDownDisposable = editor.onKeyDown((event) => {
        const browserEvent = event?.browserEvent;
        if (!browserEvent || String(browserEvent.key || '') !== 'Escape') {
            return;
        }

        if (vimAdapter || isMonacoOverlayOpen()) {
            return;
        }

        browserEvent.preventDefault?.();
        browserEvent.stopPropagation?.();
        void enableVimMode();
    });

    const scrollDisposable = editor.onDidScrollChange(() => {
        positionVimIndicator();
    });

    const blurDisposable = editor.onDidBlurEditorWidget(() => {
        if (typeof options.onBlur === 'function') {
            options.onBlur();
        }
        if (vimIndicator && currentVimLabel) {
            vimIndicator.style.opacity = '0';
        }
    });

    const focusDisposable = editor.onDidFocusEditorWidget(() => {
        if (vimAdapter) {
            applyVimCursorStyle(currentVimMode);
        }
        if (vimIndicator && currentVimLabel) {
            vimIndicator.style.opacity = '1';
            positionVimIndicator();
        }
    });

    const mouseDownDisposable = editor.onMouseDown((event) => {
        const browserEvent = event?.event?.browserEvent;
        const position = event?.target?.position;

        if (!browserEvent || Number(browserEvent.button) !== 0 || !position) {
            return;
        }

        const misspelling = getMisspellingAtOffset(model.getOffsetAt(position));
        if (!misspelling) {
            return;
        }

        browserEvent.preventDefault?.();
        browserEvent.stopPropagation?.();

        showSpellcheckSuggestionsPopup(
            Number(browserEvent.clientX) || 0,
            Number(browserEvent.clientY) || 0,
            misspelling,
            (suggestion) => {
                replaceMisspellingWithSuggestion(misspelling, suggestion);
            },
        );
    });

    const domNode = editor.getDomNode();
    const onPasteHandler = typeof options.onPaste === 'function'
        ? (event) => options.onPaste(event)
        : null;
    const onContextMenuHandler = typeof options.onContextMenu === 'function'
        ? (event) => options.onContextMenu(event)
        : null;

    if (domNode && onPasteHandler) {
        domNode.addEventListener('paste', onPasteHandler);
    }

    if (domNode && onContextMenuHandler) {
        domNode.addEventListener('contextmenu', onContextMenuHandler);
    }

    const lspDisposables = [];
    let lspCodeActionCommandDisposable = null;
    let typosDecorationIds = [];
    let currentTyposMisspellings = [];

    function disposeLsp() {
        while (lspDisposables.length > 0) {
            const disposable = lspDisposables.pop();
            disposable?.dispose?.();
        }
        if (lspCodeActionCommandDisposable) {
            lspCodeActionCommandDisposable.dispose();
            lspCodeActionCommandDisposable = null;
        }
        monaco.editor.setModelMarkers(model, 'notes-lsp', []);
    }

    function setTyposDecorations(list) {
        const misspellings = Array.isArray(list) ? list : [];
        currentTyposMisspellings = misspellings
            .map((item) => ({
                ...item,
                wordStart: Math.max(0, Math.min(Number(item?.wordStart) || 0, model.getValueLength())),
                wordLength: Math.max(1, Number(item?.wordLength) || 1),
            }))
            .filter((item) => item.wordStart < model.getValueLength());
        const next = misspellings
            .map((item) => {
                const startOffset = Math.max(0, Math.min(Number(item?.wordStart) || 0, model.getValueLength()));
                const rawLength = Math.max(1, Number(item?.wordLength) || 1);
                const endOffset = Math.max(startOffset + 1, Math.min(startOffset + rawLength, model.getValueLength()));
                const startPos = model.getPositionAt(startOffset);
                const endPos = model.getPositionAt(endOffset);

                const suggestions = Array.isArray(item?.suggestions)
                    ? item.suggestions.filter((entry) => String(entry || '').trim() !== '')
                    : [];
                const hoverLines = [];
                if (item?.misspeltWord) {
                    hoverLines.push(`Misspelling: ${String(item.misspeltWord)}`);
                }
                if (suggestions.length > 0) {
                    hoverLines.push(`Suggestions: ${suggestions.join(', ')}`);
                }

                return {
                    range: {
                        startLineNumber: startPos.lineNumber,
                        startColumn: startPos.column,
                        endLineNumber: endPos.lineNumber,
                        endColumn: endPos.column,
                    },
                    options: {
                        inlineClassName: 'notes-monaco-typo-squiggle',
                        hoverMessage: hoverLines.length > 0 ? [{ value: hoverLines.join('\n') }] : undefined,
                    },
                };
            })
            .filter((item) => item.range.endLineNumber >= item.range.startLineNumber);

        typosDecorationIds = editor.deltaDecorations(typosDecorationIds, next);
    }

    function getMisspellingAtOffset(offset) {
        const safeOffset = Math.max(0, Math.min(Number(offset) || 0, model.getValueLength()));
        return currentTyposMisspellings.find((item) => {
            const start = Math.max(0, Number(item?.wordStart) || 0);
            const length = Math.max(1, Number(item?.wordLength) || 1);
            return safeOffset >= start && safeOffset < start + length;
        }) || null;
    }

    function replaceMisspellingWithSuggestion(misspelling, suggestion) {
        const start = Math.max(0, Math.min(Number(misspelling?.wordStart) || 0, model.getValueLength()));
        const end = Math.max(start, Math.min(start + (Number(misspelling?.wordLength) || 1), model.getValueLength()));
        const startPos = model.getPositionAt(start);
        const endPos = model.getPositionAt(end);
        const text = String(suggestion || '');

        editor.executeEdits('notes-monaco-spellcheck', [{
            range: {
                startLineNumber: startPos.lineNumber,
                startColumn: startPos.column,
                endLineNumber: endPos.lineNumber,
                endColumn: endPos.column,
            },
            text,
            forceMoveMarkers: true,
        }]);

        const nextPos = model.getPositionAt(start + text.length);
        editor.setPosition(nextPos);
        editor.setSelection({
            startLineNumber: nextPos.lineNumber,
            startColumn: nextPos.column,
            endLineNumber: nextPos.lineNumber,
            endColumn: nextPos.column,
        });
        editor.focus();
    }

    function toRange(range) {
        const startLine = (Number(range?.start?.line) || 0) + 1;
        const startColumn = (Number(range?.start?.character) || 0) + 1;
        const endLine = (Number(range?.end?.line) || Number(range?.start?.line) || 0) + 1;
        const endColumn = (Number(range?.end?.character) || Number(range?.start?.character) || 0) + 1;
        return {
            startLineNumber: Math.max(1, startLine),
            startColumn: Math.max(1, startColumn),
            endLineNumber: Math.max(1, endLine),
            endColumn: Math.max(1, endColumn),
        };
    }

    function toMarkerSeverity(severity) {
        switch (Number(severity) || 0) {
        case 1:
            return monaco.MarkerSeverity.Error;
        case 2:
            return monaco.MarkerSeverity.Warning;
        case 3:
            return monaco.MarkerSeverity.Info;
        case 4:
            return monaco.MarkerSeverity.Hint;
        default:
            return monaco.MarkerSeverity.Info;
        }
    }

    return {
        getValue() {
            return editor.getValue();
        },

        setValue(value) {
            const text = String(value || '');
            if (editor.getValue() === text) {
                return;
            }
            editor.setValue(text);
        },

        setLanguage(language) {
            monaco.editor.setModelLanguage(model, resolveMonacoLanguage(language));
        },

        setDiagnostics(diagnostics) {
            const markers = (Array.isArray(diagnostics) ? diagnostics : []).map((diag) => {
                const range = toRange(diag?.range || {});
                return {
                    ...range,
                    severity: toMarkerSeverity(diag?.severity),
                    message: String(diag?.message || ''),
                    source: String(diag?.source || 'lsp'),
                };
            });
            monaco.editor.setModelMarkers(model, 'notes-lsp', markers);
        },

        setTyposMisspellings(list) {
            setTyposDecorations(list);
        },

        configureLsp(callbacks = {}) {
            disposeLsp();

            const languageId = model.getLanguageId();
            const codeActionCommandId = `notes.applyCodeAction.${Date.now()}`;

            lspCodeActionCommandDisposable = monaco.editor.registerCommand(codeActionCommandId, async (_accessor, args) => {
                if (typeof callbacks.applyCodeAction !== 'function') {
                    return;
                }
                const result = await callbacks.applyCodeAction(args || {});
                if (result && result.changed === true && typeof result.content === 'string') {
                    editor.setValue(String(result.content || ''));
                }
            });

            lspDisposables.push(monaco.languages.registerCompletionItemProvider(languageId, {
                triggerCharacters: ['.', ':', '>'],
                provideCompletionItems: async (_model, position, _context) => {
                    if (typeof callbacks.completion !== 'function') {
                        return { suggestions: [] };
                    }
                    const items = await callbacks.completion({ line: position.lineNumber - 1, character: position.column - 1 });
                    const suggestions = (Array.isArray(items) ? items : []).map((item) => ({
                        label: String(item?.label || ''),
                        insertText: String(item?.insertText || item?.label || ''),
                        kind: Number(item?.kind) || monaco.languages.CompletionItemKind.Text,
                        detail: item?.detail ? String(item.detail) : undefined,
                        documentation: item?.documentation ? String(item.documentation) : undefined,
                    }));
                    return { suggestions };
                },
            }));

            lspDisposables.push(monaco.languages.registerSignatureHelpProvider(languageId, {
                signatureHelpTriggerCharacters: ['(', ','],
                provideSignatureHelp: async (_model, position) => {
                    if (typeof callbacks.signature !== 'function') {
                        return null;
                    }
                    const text = await callbacks.signature({ line: position.lineNumber - 1, character: position.column - 1 });
                    if (!text) {
                        return null;
                    }
                    return {
                        value: {
                            signatures: [{ label: String(text), parameters: [] }],
                            activeSignature: 0,
                            activeParameter: 0,
                        },
                        dispose: () => {},
                    };
                },
            }));

            lspDisposables.push(monaco.languages.registerDocumentFormattingEditProvider(languageId, {
                provideDocumentFormattingEdits: async () => {
                    if (typeof callbacks.formatDocument !== 'function') {
                        return [];
                    }
                    const result = await callbacks.formatDocument();
                    if (!result || !result.changed || typeof result.content !== 'string') {
                        return [];
                    }
                    const fullRange = model.getFullModelRange();
                    return [{ range: fullRange, text: String(result.content || '') }];
                },
            }));

            lspDisposables.push(monaco.languages.registerDocumentRangeFormattingEditProvider(languageId, {
                provideDocumentRangeFormattingEdits: async (_model, range) => {
                    if (typeof callbacks.formatRange !== 'function') {
                        return [];
                    }
                    const result = await callbacks.formatRange({
                        start: { line: range.startLineNumber - 1, character: range.startColumn - 1 },
                        end: { line: range.endLineNumber - 1, character: range.endColumn - 1 },
                    });
                    if (!result || !result.changed || typeof result.content !== 'string') {
                        return [];
                    }
                    const fullRange = model.getFullModelRange();
                    return [{ range: fullRange, text: String(result.content || '') }];
                },
            }));

            lspDisposables.push(monaco.languages.registerDefinitionProvider(languageId, {
                provideDefinition: async (_model, position) => {
                    if (typeof callbacks.definition !== 'function') {
                        return [];
                    }
                    const defs = await callbacks.definition({ line: position.lineNumber - 1, character: position.column - 1 });
                    return (Array.isArray(defs) ? defs : []).map((item) => ({
                        uri: monaco.Uri.parse(String(item?.uri || model.uri.toString())),
                        range: {
                            startLineNumber: (Number(item?.line) || 0) + 1,
                            startColumn: (Number(item?.character) || 0) + 1,
                            endLineNumber: (Number(item?.line) || 0) + 1,
                            endColumn: (Number(item?.character) || 0) + 1,
                        },
                    }));
                },
            }));

            lspDisposables.push(monaco.languages.registerRenameProvider(languageId, {
                resolveRenameLocation: async (_model, position) => {
                    if (typeof callbacks.prepareRename !== 'function') {
                        return null;
                    }
                    const prepared = await callbacks.prepareRename({ line: position.lineNumber - 1, character: position.column - 1 });
                    if (!prepared || prepared.canRename !== true) {
                        return { rejectReason: 'Rename not available at this position' };
                    }

                    return {
                        range: {
                            startLineNumber: position.lineNumber,
                            startColumn: Math.max(1, position.column - 1),
                            endLineNumber: position.lineNumber,
                            endColumn: Math.max(position.column, position.column + 1),
                        },
                        text: String(prepared.placeholder || ''),
                    };
                },
                provideRenameEdits: async (_model, position, newName) => {
                    if (typeof callbacks.rename !== 'function') {
                        return { edits: [] };
                    }
                    const result = await callbacks.rename({
                        line: position.lineNumber - 1,
                        character: position.column - 1,
                        newName,
                    });
                    if (!result || !result.changed || typeof result.content !== 'string') {
                        return { edits: [] };
                    }
                    return {
                        edits: [{
                            resource: model.uri,
                            textEdit: {
                                range: model.getFullModelRange(),
                                text: String(result.content || ''),
                            },
                        }],
                    };
                },
            }));

            lspDisposables.push(monaco.languages.registerCodeActionProvider(languageId, {
                provideCodeActions: async (_model, range) => {
                    if (typeof callbacks.codeActions !== 'function') {
                        return { actions: [], dispose: () => {} };
                    }
                    const payload = {
                        line: range.startLineNumber - 1,
                        character: range.startColumn - 1,
                    };
                    const actions = await callbacks.codeActions(payload);
                    const mapped = (Array.isArray(actions) ? actions : []).map((action) => ({
                        title: String(action?.title || 'Code action'),
                        kind: String(action?.kind || 'quickfix'),
                        command: {
                            id: codeActionCommandId,
                            title: String(action?.title || 'Code action'),
                            arguments: [action],
                        },
                    }));
                    return { actions: mapped, dispose: () => {} };
                },
            }));
        },

        getSelectionOffsets() {
            const selection = editor.getSelection();
            if (!selection) {
                return { start: 0, end: 0 };
            }

            const start = model.getOffsetAt(selection.getStartPosition());
            const end = model.getOffsetAt(selection.getEndPosition());
            return { start, end };
        },

        setSelectionOffsets(start, end) {
            const safeStart = Math.max(0, Math.min(Number(start) || 0, model.getValueLength()));
            const safeEnd = Math.max(safeStart, Math.min(Number(end) || safeStart, model.getValueLength()));
            const startPos = model.getPositionAt(safeStart);
            const endPos = model.getPositionAt(safeEnd);
            editor.setSelection({
                startLineNumber: startPos.lineNumber,
                startColumn: startPos.column,
                endLineNumber: endPos.lineNumber,
                endColumn: endPos.column,
            });
            positionVimIndicator();
        },

        getSelectionText() {
            const selection = editor.getSelection();
            if (!selection) {
                return '';
            }
            return model.getValueInRange(selection);
        },

        replaceRange(start, end, text, source = 'notes-monaco-edit') {
            const safeStart = Math.max(0, Math.min(Number(start) || 0, model.getValueLength()));
            const safeEnd = Math.max(safeStart, Math.min(Number(end) || safeStart, model.getValueLength()));
            const startPos = model.getPositionAt(safeStart);
            const endPos = model.getPositionAt(safeEnd);

            editor.executeEdits(source, [{
                range: {
                    startLineNumber: startPos.lineNumber,
                    startColumn: startPos.column,
                    endLineNumber: endPos.lineNumber,
                    endColumn: endPos.column,
                },
                text: String(text || ''),
                forceMoveMarkers: true,
            }]);
        },

        findMatches(query, options = {}) {
            const search = String(query || '');
            if (!search) {
                return [];
            }

            const isRegex = options.isRegex === true;
            const matchCase = options.matchCase === true;
            const matches = model.findMatches(
                search,
                true,
                isRegex,
                matchCase,
                null,
                false,
                Number(options.limit) || 10000,
            );

            return (matches || []).map((match) => {
                const start = model.getOffsetAt(match.range.getStartPosition());
                const end = model.getOffsetAt(match.range.getEndPosition());
                return { start, end };
            });
        },

        revealOffset(offset) {
            const pos = model.getPositionAt(Math.max(0, Math.min(Number(offset) || 0, model.getValueLength())));
            editor.revealPositionInCenter(pos);
        },

        focus() {
            editor.focus();
        },

        openCommandPalette() {
            editor.focus();
            const action = editor.getAction?.('editor.action.quickCommand');
            if (action && typeof action.run === 'function') {
                action.run();
                return;
            }
            editor.trigger('notes-monaco', 'editor.action.quickCommand', null);
        },

        layout(dimensions) {
            if (dimensions
                && Number.isFinite(dimensions.width) && dimensions.width > 0
                && Number.isFinite(dimensions.height) && dimensions.height > 0) {
                editor.layout({ width: Math.round(dimensions.width), height: Math.round(dimensions.height) });
                return;
            }

            // Measure the real, visible box so Monaco never lays out against a
            // stale or zero-sized container (the classic "created while hidden"
            // footgun that leaves the editor blank until a later resize).
            const rect = container.getBoundingClientRect();
            if (rect && rect.width > 0 && rect.height > 0) {
                editor.layout({ width: Math.round(rect.width), height: Math.round(rect.height) });
                positionVimIndicator();
                return;
            }

            editor.layout();
            positionVimIndicator();
        },

        setWordWrap(enabled) {
            editor.updateOptions({ wordWrap: enabled ? 'on' : 'off' });
        },

        setTypography(nextOptions = {}) {
            const next = toFontMetrics(nextOptions);
            editor.updateOptions({
                fontFamily: next.fontFamily,
                fontSize: next.fontSize,
                lineHeight: next.lineHeight,
            });
        },

        applyTheme(colors) {
            applyTheme(colors || {});
        },

        dispose() {
            vimAdapterDisposing = true;
            if (vimAdapter) {
                vimAdapter.dispose?.();
                vimAdapter = null;
            }
            stopMonacoVimCursorPulse();
            setMonacoVimActiveClass(false);
            if (vimIndicator) {
                vimIndicator.remove();
                vimIndicator = null;
            }
            currentVimLabel = '';
            currentVimMode = '';
            currentVimSubMode = '';
            vimReplaceOncePending = false;
            changeDisposable.dispose();
            cursorDisposable.dispose();
            mouseMoveDisposable.dispose();
            blurDisposable.dispose();
            focusDisposable.dispose();
            mouseDownDisposable.dispose();
            keyDownDisposable.dispose();
            scrollDisposable.dispose();
            disposeLsp();
            while (fallbackProviderDisposables.length > 0) {
                const disposable = fallbackProviderDisposables.pop();
                disposable?.dispose?.();
            }
            setTyposDecorations([]);
            if (domNode && onPasteHandler) {
                domNode.removeEventListener('paste', onPasteHandler);
            }
            if (domNode && onContextMenuHandler) {
                domNode.removeEventListener('contextmenu', onContextMenuHandler);
            }
            editor.dispose();
            model.dispose();
        },
    };
}
