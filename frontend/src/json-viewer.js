/**
 * Generic JSON viewer with expandable/collapsible nodes.
 * Can be reused to render any JSON string or object.
 */

function escapeHtml(value) {
    return String(value)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

function formatPrimitive(value) {
    if (value === null) {
        return '<span class="json-value-null">null</span>';
    }

    if (typeof value === 'string') {
        return `<span class="json-value-string">"${escapeHtml(value)}"</span>`;
    }

    if (typeof value === 'number') {
        return `<span class="json-value-number">${value}</span>`;
    }

    if (typeof value === 'boolean') {
        return `<span class="json-value-boolean">${value}</span>`;
    }

    return `<span class="json-value-string">"${escapeHtml(String(value))}"</span>`;
}

function buildNode(value, key, depth) {
    const indent = depth * 18;
    const keyPrefix = key !== null
        ? `<span class="json-key">"${escapeHtml(key)}"</span><span class="json-colon">: </span>`
        : '';

    if (Array.isArray(value)) {
        const children = value.map((item, index) => buildNode(item, String(index), depth + 1)).join('');
        return `
            <div class="json-node" data-node-type="array" data-expanded="true">
                <div class="json-row" style="padding-left: ${indent}px;">
                    <button type="button" class="json-toggle" aria-label="Collapse node" aria-expanded="true"></button>
                    ${keyPrefix}<span class="json-brace">[</span><span class="json-meta">${value.length} item${value.length === 1 ? '' : 's'}</span><span class="json-brace">]</span>
                </div>
                <div class="json-children">${children}</div>
            </div>
        `;
    }

    if (value && typeof value === 'object') {
        const entries = Object.entries(value);
        const children = entries.map(([childKey, childValue]) => buildNode(childValue, childKey, depth + 1)).join('');
        return `
            <div class="json-node" data-node-type="object" data-expanded="true">
                <div class="json-row" style="padding-left: ${indent}px;">
                    <button type="button" class="json-toggle" aria-label="Collapse node" aria-expanded="true"></button>
                    ${keyPrefix}<span class="json-brace">{</span><span class="json-meta">${entries.length} propert${entries.length === 1 ? 'y' : 'ies'}</span><span class="json-brace">}</span>
                </div>
                <div class="json-children">${children}</div>
            </div>
        `;
    }

    return `
        <div class="json-node json-node-leaf" data-node-type="leaf">
            <div class="json-row" style="padding-left: ${indent}px;">
                <span class="json-toggle-placeholder"></span>
                ${keyPrefix}${formatPrimitive(value)}
            </div>
        </div>
    `;
}

function parseInput(input) {
    if (typeof input === 'string') {
        return JSON.parse(input);
    }

    return input;
}

export function renderJsonViewer(container, input) {
    if (!container) {
        return;
    }

    try {
        const parsed = parseInput(input);
        const rootHtml = buildNode(parsed, null, 0);

        container.innerHTML = `<div class="json-viewer-root">${rootHtml}</div>`;

        const toggles = container.querySelectorAll('.json-toggle');
        toggles.forEach((toggle) => {
            toggle.addEventListener('click', () => {
                const node = toggle.closest('.json-node');
                if (!node) {
                    return;
                }

                const isExpanded = node.getAttribute('data-expanded') !== 'false';
                const nextExpanded = !isExpanded;
                node.setAttribute('data-expanded', nextExpanded ? 'true' : 'false');
                toggle.setAttribute('aria-expanded', nextExpanded ? 'true' : 'false');
                toggle.setAttribute('aria-label', nextExpanded ? 'Collapse node' : 'Expand node');
            });
        });
    } catch (err) {
        container.innerHTML = `
            <div class="json-viewer-error">
                Invalid JSON: ${escapeHtml(err.message || 'Unable to parse JSON')}
            </div>
        `;
    }
}
