/**
 * Shared utilities for markdown rendering across notes.js and markdown.js
 */

import { GetImage } from '../wailsjs/go/main/WApp';
import { BrowserOpenURL } from '../wailsjs/runtime/runtime';
import { marked } from "marked";
import { gfmHeadingId } from "marked-gfm-heading-id";
import hljs from "highlight.js/lib/common";

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
 * Apply syntax highlighting to all code blocks in a container
 * @param {HTMLElement} container - The container element to search for code blocks
 */
export function applySyntaxHighlighting(container) {
    container.querySelectorAll('pre code').forEach((block) => {
        hljs.highlightElement(block);
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
 * Complete markdown processing pipeline - applies all common transformations
 * @param {HTMLElement} container - The container element with rendered markdown
 */
export async function processMarkdownContainer(container) {
    applySyntaxHighlighting(container);
    await processWailsImages(container);
    processLinks(container, { enableBookmarks: true });
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
