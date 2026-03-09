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
 */
export function processLinks(container) {
    container.querySelectorAll('a').forEach(a => {
        if (!a.href.match(rxWailsUrl)) {
            // External link - open in browser
            a.addEventListener('click', (e) => {
                e.preventDefault();
                BrowserOpenURL(a.href);
            });
        }

        if (!a.href.match(rxBookmark)) {
            // Could add bookmark handling here if needed
            // const id = a.href.replace(rxBookmark, '');
            // a.addEventListener("click", () => {
            //     document.getElementById(id).scrollIntoView();
            // });
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
    processLinks(container);
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
