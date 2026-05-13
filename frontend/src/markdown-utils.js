/**
 * Shared utilities for markdown rendering across notes.js and markdown.js
 */

import { GetImage, GetCustomRegexp } from '../wailsjs/go/main/WApp';
import { BrowserOpenURL } from '../wailsjs/runtime/runtime';
import { showFullscreenImageOverlay } from './fullscreen-image-overlay';
import { marked } from "marked";
import { gfmHeadingId } from "marked-gfm-heading-id";
import mermaid from "mermaid";
import hljs from "highlight.js/lib/common";

const hljsLanguageLoaders = import.meta.glob('../node_modules/highlight.js/lib/languages/*.js');

const hljsLanguageAliases = {
    'c++': 'cpp',
    'c#': 'csharp',
    'f#': 'fsharp',
    'objective-c': 'objectivec',
    'obj-c': 'objectivec',
    'sh': 'bash',
    'shell': 'bash',
    'docker': 'dockerfile',
    'yml': 'yaml',
};

function normalizeLanguageName(language) {
    const cleaned = String(language || '').trim().toLowerCase();
    if (!cleaned) {
        return '';
    }
    return hljsLanguageAliases[cleaned] || cleaned;
}

async function ensureHighlightLanguage(language) {
    const normalized = normalizeLanguageName(language);
    if (!normalized) {
        return false;
    }

    if (hljs.getLanguage(normalized)) {
        return true;
    }

    const loaderPath = Object.keys(hljsLanguageLoaders).find((path) => path.endsWith(`/${normalized}.js`));
    if (!loaderPath) {
        return false;
    }

    try {
        const module = await hljsLanguageLoaders[loaderPath]();
        if (module && typeof module.default === 'function') {
            hljs.registerLanguage(normalized, module.default);
            return true;
        }
    } catch (err) {
        console.warn(`Unable to load highlight.js language: ${normalized}`, err);
    }

    return false;
}

function getBlockLanguage(block) {
    const langClass = Array.from(block.classList).find((name) => name.startsWith('language-'));
    if (!langClass) {
        return '';
    }
    return normalizeLanguageName(langClass.slice('language-'.length));
}

// Initialize Mermaid
mermaid.initialize({
    startOnLoad: false,
    theme: 'dark',
    darkMode: true,
    securityLevel: 'loose',
});

// Configure marked with GFM heading IDs
export function configureMarked() {
    marked.use(gfmHeadingId({}));
}

/**
 * Regular expressions for Wails URLs
 */
const rxWailsUrl = /^(wails:\/\/wails\/|http:\/\/localhost:[0-9]+\/|wails:\/\/wails.localhost:[0-9]+\/)/;
const rxBookmark = /^(wails:\/\/wails\/|http:\/\/localhost:[0-9]+\/|wails:\/\/wails.localhost:[0-9]+\/)#/;

/**
 * Render Mermaid diagrams in a container
 * @param {HTMLElement} container - The container element to search for mermaid code blocks
 */
export async function renderMermaidDiagrams(container) {
    const mermaidBlocks = container.querySelectorAll('pre code.language-mermaid');
    
    if (mermaidBlocks.length === 0) {
        return;
    }
    
    // Replace code blocks with pre-rendered mermaid content
    let diagramIndex = 0;
    for (const block of mermaidBlocks) {
        const pre = block.parentElement;
        const mermaidCode = block.textContent;
        
        try {
            // Generate unique ID for this diagram
            const id = `mermaid-diagram-${Date.now()}-${diagramIndex++}`;
            
            // Render the diagram using mermaid.render()
            const { svg } = await mermaid.render(id, mermaidCode);
            
            // Create a container div for the rendered SVG
            const mermaidDiv = document.createElement('div');
            mermaidDiv.className = 'mermaid-diagram';
            mermaidDiv.innerHTML = svg;
            
            // Replace the pre block with the rendered diagram
            pre.replaceWith(mermaidDiv);
        } catch (err) {
            // If rendering fails, show error in place
            console.error('Mermaid rendering error:', err);
            const errorDiv = document.createElement('div');
            errorDiv.className = 'mermaid-error';
            errorDiv.style.color = 'var(--error)';
            errorDiv.style.padding = '10px';
            errorDiv.style.border = '1px solid var(--error)';
            errorDiv.style.borderRadius = '4px';
            errorDiv.style.marginBottom = '10px';
            errorDiv.textContent = `Mermaid diagram error: ${err.message || err}`;
            pre.replaceWith(errorDiv);
        }
    }
}

/**
 * Apply syntax highlighting to all code blocks in a container
 * @param {HTMLElement} container - The container element to search for code blocks
 */
export async function applySyntaxHighlighting(container) {
    const blocks = Array.from(container.querySelectorAll('pre code'));
    const languages = new Set();

    blocks.forEach((block) => {
        const language = getBlockLanguage(block);
        if (language) {
            languages.add(language);
        }
    });

    await Promise.all(Array.from(languages).map((language) => ensureHighlightLanguage(language)));

    blocks.forEach((block) => {
        const language = getBlockLanguage(block);
        
        // Skip mermaid blocks - they'll be handled separately
        if (language === 'mermaid') {
            return;
        }
        
        if (!language || hljs.getLanguage(language)) {
            hljs.highlightElement(block);
            return;
        }

        const highlighted = hljs.highlightAuto(block.textContent || '');
        block.classList.add('hljs');
        block.innerHTML = highlighted.value;
    });
}

/**
 * Process all images in a container, replacing Wails URLs with actual image data
 * @param {HTMLElement} container - The container element to search for images
 */
export async function processWailsImages(container) {
    const images = container.querySelectorAll('img');
    
    for (const img of images) {
        if (img.src.match(rxWailsUrl)) {
            const path = img.src.replace(rxWailsUrl, '');
            // Extract filename from path and store as data attribute
            const filename = path.split('/').pop() || 'Image';
            img.dataset.originalFilename = filename;
            try {
                const imageData = await GetImage(path);
                if (!imageData.match(/^error: /)) {
                    img.src = imageData;
                } else {
                    console.error('Error loading image:', imageData);
                }
            } catch (err) {
                console.error('Error getting image:', err);
            }
        }
    }
}

export function enableFullscreenImages(container) {
    const images = container.querySelectorAll('img');
    images.forEach((img) => {
        if (img.dataset.fullscreenBound === 'true') {
            return;
        }

        img.dataset.fullscreenBound = 'true';
        img.style.cursor = 'zoom-in';
        img.addEventListener('click', (e) => {
            e.preventDefault();

            const sourceWidth = img.naturalWidth || img.width || 0;
            const sourceHeight = img.naturalHeight || img.height || 0;
            showFullscreenImageOverlay({
                dataURL: img.src,
                sourceWidth,
                sourceHeight,
            });
        });
    });
}

/**
 * Enable fullscreen viewing for Mermaid diagrams
 * @param {HTMLElement} container - The container element to search for mermaid diagrams
 */
export function enableFullscreenMermaidDiagrams(container) {
    const diagrams = container.querySelectorAll('.mermaid-diagram');
    diagrams.forEach((diagram) => {
        if (diagram.dataset.fullscreenBound === 'true') {
            return;
        }

        diagram.dataset.fullscreenBound = 'true';
        diagram.style.cursor = 'zoom-in';
        diagram.addEventListener('click', (e) => {
            e.preventDefault();

            const svg = diagram.querySelector('svg');
            if (!svg) {
                console.error('No SVG found in mermaid diagram');
                return;
            }

            // Get SVG dimensions from viewBox or attributes
            const viewBox = svg.viewBox?.baseVal;
            const width = viewBox?.width || svg.width?.baseVal?.value || parseInt(svg.getAttribute('width')) || 800;
            const height = viewBox?.height || svg.height?.baseVal?.value || parseInt(svg.getAttribute('height')) || 600;

            // Clone SVG and ensure it has proper dimensions
            const clonedSvg = svg.cloneNode(true);
            if (!clonedSvg.getAttribute('width')) {
                clonedSvg.setAttribute('width', width);
            }
            if (!clonedSvg.getAttribute('height')) {
                clonedSvg.setAttribute('height', height);
            }
            // Preserve viewBox
            if (viewBox && !clonedSvg.getAttribute('viewBox')) {
                clonedSvg.setAttribute('viewBox', `${viewBox.x} ${viewBox.y} ${viewBox.width} ${viewBox.height}`);
            }

            showFullscreenImageOverlay({
                svgElement: clonedSvg,
                sourceWidth: Math.round(width),
                sourceHeight: Math.round(height),
            });
        });
    });
}

/**
 * Process all links in a container, handling external links and bookmarks
 * @param {HTMLElement} container - The container element to search for links
 * @param {Object} options - Link handling options
 * @param {boolean} options.enableBookmarks - Enable in-document bookmark scrolling
 */
export function processLinks(container, options = {}) {
    const { enableBookmarks = false } = options;

    container.querySelectorAll('a').forEach(a => {
        const rawHref = a.getAttribute('href') || '';
        const isHashOnly = rawHref.startsWith('#');
        const isBookmark = isHashOnly || a.href.match(rxBookmark);

        if (enableBookmarks && isBookmark) {
            const id = isHashOnly ? rawHref.slice(1) : a.href.replace(rxBookmark, '');
            if (!id) {
                return;
            }

            a.addEventListener('click', (e) => {
                e.preventDefault();
                const safeId = typeof CSS !== 'undefined' && CSS.escape ? CSS.escape(id) : id;
                const target = container.querySelector(`#${safeId}`);
                if (target) {
                    target.scrollIntoView({ behavior: 'smooth', block: 'start' });
                }
            });
            return;
        }

        if (!a.href.match(rxWailsUrl)) {
            // External link - open in browser
            a.addEventListener('click', (e) => {
                e.preventDefault();
                BrowserOpenURL(a.href);
            });
        }
    });
}

/**
 * Apply custom regex hyperlinking to text nodes in the container
 * @param {HTMLElement} container - The container element to process
 */
export async function autoHyperlink(container) {
    const customRegexps = await GetCustomRegexp?.() || [];

    if (!customRegexps || customRegexps.length === 0) {
        return;
    }

    for (const custom of customRegexps) {
        if (!custom.pattern || !custom.link) {
            continue;
        }

        let regex;
        try {
            regex = new RegExp(custom.pattern, 'g');
        } catch (err) {
            console.warn('Invalid custom regexp:', custom.pattern, err);
            continue;
        }

        const walker = document.createTreeWalker(
            container,
            NodeFilter.SHOW_TEXT,
            null,
            false
        );

        const nodesToProcess = [];
        let node;
        while ((node = walker.nextNode())) {
            let parent = node.parentNode;
            let insideLink = false;
            while (parent) {
                if (parent.tagName === 'A') {
                    insideLink = true;
                    break;
                }
                parent = parent.parentNode;
            }

            if (!insideLink && regex.test(node.textContent)) {
                regex.lastIndex = 0;
                nodesToProcess.push(node);
            }
        }

        nodesToProcess.forEach((textNode) => {
            const text = textNode.textContent;
            const parts = [];
            let lastIndex = 0;
            let match;

            regex.lastIndex = 0;
            while ((match = regex.exec(text)) !== null) {
                if (match.index > lastIndex) {
                    parts.push(document.createTextNode(text.substring(lastIndex, match.index)));
                }

                const matchedText = match[0];
                const link = matchedText.replace(new RegExp(custom.pattern), custom.link);
                const a = document.createElement('a');
                a.href = link;
                a.textContent = matchedText;
                a.addEventListener('click', (e) => {
                    e.preventDefault();
                    BrowserOpenURL(a.href);
                });
                parts.push(a);

                lastIndex = regex.lastIndex;
            }

            if (lastIndex < text.length) {
                parts.push(document.createTextNode(text.substring(lastIndex)));
            }

            const fragment = document.createDocumentFragment();
            parts.forEach(part => fragment.appendChild(part));
            textNode.parentNode.replaceChild(fragment, textNode);
        });
    }
}

/**
 * Complete markdown processing pipeline - applies all common transformations
 * @param {HTMLElement} container - The container element with rendered markdown
 */
export async function processMarkdownContainer(container) {
    await renderMermaidDiagrams(container);
    enableFullscreenMermaidDiagrams(container);
    await applySyntaxHighlighting(container);
    await processWailsImages(container);
    enableFullscreenImages(container);
    processLinks(container, { enableBookmarks: true });
    await autoHyperlink(container);
}

/**
 * Parse markdown and apply all processing
 * @param {string} markdown - The markdown text to parse
 * @param {HTMLElement} container - The container element to render into
 */
export async function renderMarkdownWithProcessing(markdown, container) {
    configureMarked();
    container.innerHTML = marked.parse(markdown);
    await processMarkdownContainer(container);
}
