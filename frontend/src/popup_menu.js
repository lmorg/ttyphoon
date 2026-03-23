import { EventsOn } from '../wailsjs/runtime/runtime';
import { TerminalMenuHighlight, TerminalMenuSelect, TerminalMenuCancel, TerminalRequestRedraw } from '../wailsjs/go/main/WApp';

const LISTBOX_ROOT_ID = 'ttyphoon-listbox-menu';

// Registry for pure-JS context menus with negative IDs (never forwarded to Go).
const _localCallbacks = new Map();
let _localNextId = -1;
let _showListMenuFn = null;
let _setAnchorFn = null;
let _menuOperationInProgress = false;
let _localMenuReturnFocus = null;

function menuHighlight(id, index) {
    if (id < 0) {
        _localCallbacks.get(id)?.highlight?.(index);
        return;
    }
    TerminalMenuHighlight(id, index).catch(() => {});
    TerminalRequestRedraw().catch(() => {});
}

function menuSelect(id, index) {
    if (id < 0) {
        const returnTo = _localMenuReturnFocus;
        _localMenuReturnFocus = null;
        _localCallbacks.get(id)?.select?.(index);
        _localCallbacks.delete(id);
        if (returnTo) returnTo.focus();
        return;
    }
    TerminalMenuSelect(id, index).catch(() => {});
}

function menuCancel(id, index) {
    if (id < 0) {
        const returnTo = _localMenuReturnFocus;
        _localMenuReturnFocus = null;
        _localCallbacks.get(id)?.cancel?.(index);
        _localCallbacks.delete(id);
        if (returnTo) returnTo.focus();
        return;
    }
    TerminalMenuCancel(id, index).catch(() => {});
}

function normalizeMenuPayload(payload) {
    if (Array.isArray(payload?.[0])) {
        return payload[0];
    }
    if (Array.isArray(payload)) {
        return payload[0] || payload;
    }
    return payload;
}

function toIconText(icon) {
    if (!Number.isFinite(icon) || icon <= 0) {
        return '';
    }

    try {
        return String.fromCodePoint(icon);
    } catch {
        return '';
    }
}

function isSeparatorTitle(title) {
    return title === '-';
}

function measureIdealWidth(items, title, withIcons) {
    const c = document.createElement('canvas');
    const ctx = c.getContext('2d');
    const rootStyle = getComputedStyle(document.documentElement);
    const fontFamily = rootStyle.getPropertyValue('--terminal-menu-font').trim()
        || 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace';
    const fontSizeVar = rootStyle.getPropertyValue('--terminal-menu-font-size').trim();
    const parsedFontSize = Number.parseFloat(fontSizeVar);
    const fontSize = Number.isFinite(parsedFontSize) && parsedFontSize > 0 ? parsedFontSize : 14;
    ctx.font = `${fontSize}px ${fontFamily}`;

    let maxTextW = 0;
    for (const item of items) {
        if (!item.separator) {
            const w = ctx.measureText(item.title).width;
            if (w > maxTextW) maxTextW = w;
        }
    }

    // row padding 8px*2, list padding 6px*2, border 1px*2, scrollbar reserve ~12px
    const rowOverhead = 16 + 12 + 2 + 12 + (withIcons ? 26 : 0);
    const itemWidth = Math.ceil(maxTextW) + rowOverhead;

    // title uses padding 8px 10px (20px horizontal) + border 1px*2 + list padding 6px*2
    let titleWidth = 0;
    if (title) {
        titleWidth = Math.ceil(ctx.measureText(title).width) + 20 + 2 + 12;
    }

    return Math.max(itemWidth, titleWidth, 300);
}

function tokenizeQuery(q) {
    return (q || '').toLowerCase().trim().split(/\s+/).filter(Boolean);
}

function buildFilteredItems(items, query) {
    const tokens = tokenizeQuery(query);

    const raw = items.map((item) => {
        if (item.separator) {
            return { ...item, visible: true };
        }

        if (tokens.length === 0) {
            return { ...item, visible: true };
        }

        const value = item.title.toLowerCase();
        const visible = tokens.every((token) => value.includes(token));
        return { ...item, visible };
    });

    // Remove separators with no visible selectable items around them.
    const result = [];
    for (let i = 0; i < raw.length; i++) {
        const item = raw[i];
        if (!item.visible) {
            continue;
        }

        if (!item.separator) {
            result.push(item);
            continue;
        }

        let hasBefore = false;
        for (let j = result.length - 1; j >= 0; j--) {
            if (!result[j].separator) {
                hasBefore = true;
                break;
            }
        }

        let hasAfter = false;
        for (let j = i + 1; j < raw.length; j++) {
            if (raw[j].visible && !raw[j].separator) {
                hasAfter = true;
                break;
            }
        }

        if (hasBefore && hasAfter) {
            result.push(item);
        }
    }

    return result;
}

export function initTerminalPopupMenu(canvas) {
    if (!canvas) {
        return;
    }

    let mouseX = 8;
    let mouseY = 8;
    let anchorX = 8;
    let anchorY = 8;

    let activeListMenuId = null;

    let listItems = [];
    let filteredItems = [];
    let highlightVisibleIndex = -1;
    let hasIcons = false;
    let query = '';

    const listRoot = document.createElement('div');
    listRoot.id = LISTBOX_ROOT_ID;
    listRoot.className = 'tty-menu tty-listbox';
    listRoot.style.display = 'none';

    const listTitle = document.createElement('div');
    listTitle.className = 'tty-menu-title';

    const listSearchWrap = document.createElement('div');
    listSearchWrap.className = 'tty-listbox-search';
    listSearchWrap.style.display = 'none';

    const listSearchInput = document.createElement('input');
    listSearchInput.type = 'text';
    listSearchInput.className = 'tty-listbox-search-input';
    listSearchInput.placeholder = 'Filter...';
    listSearchWrap.appendChild(listSearchInput);

    const listBody = document.createElement('div');
    listBody.className = 'tty-menu-list';

    listRoot.appendChild(listTitle);
    listRoot.appendChild(listSearchWrap);
    listRoot.appendChild(listBody);
    document.body.appendChild(listRoot);

    function menuConstraints() {
        const rect = canvas.getBoundingClientRect();
        return {
            maxWidth: Math.max(280, Math.floor(rect.width * 0.92)),
            maxHeight: Math.max(160, Math.floor(rect.height * 0.78)),
        };
    }

    function positionMenu(root) {
        const vw = window.innerWidth;
        const vh = window.innerHeight;
        const rect = root.getBoundingClientRect();

        let x = anchorX;
        let y = anchorY;

        if (x + rect.width > vw - 8) {
            x = Math.max(8, vw - rect.width - 8);
        }
        if (y + rect.height > vh - 8) {
            y = Math.max(8, vh - rect.height - 8);
        }

        root.style.left = `${x}px`;
        root.style.top = `${y}px`;
    }

    function applyMenuSizing(root, listEl, reserveHeaderPx = 0, idealWidth = null) {
        const { maxWidth, maxHeight } = menuConstraints();
        const targetWidth = idealWidth !== null
            ? Math.min(Math.max(300, idealWidth), maxWidth)
            : Math.min(Math.max(300, maxWidth * 0.66), maxWidth);
        root.style.maxWidth = `${maxWidth}px`;
        root.style.width = `${targetWidth}px`;
        listEl.style.maxHeight = `${Math.max(80, maxHeight - reserveHeaderPx)}px`;
    }

    function hideListMenu(cancel = true) {
        if (activeListMenuId !== null && cancel) {
            menuCancel(activeListMenuId, -1);
        }

        activeListMenuId = null;
        listItems = [];
        filteredItems = [];
        highlightVisibleIndex = -1;
        hasIcons = false;
        query = '';
        listSearchInput.value = '';
        listSearchWrap.style.display = 'none';
        listRoot.style.display = 'none';
        listBody.replaceChildren();
    }

    function hideMenus(cancel = true) {
        hideListMenu(cancel);
    }

    function visibleSelectableIndexes() {
        const indexes = [];
        for (let i = 0; i < filteredItems.length; i++) {
            if (!filteredItems[i].separator) {
                indexes.push(i);
            }
        }
        return indexes;
    }

    function ensureValidHighlight() {
        const selectable = visibleSelectableIndexes();
        if (selectable.length === 0) {
            highlightVisibleIndex = -1;
            return;
        }

        if (!selectable.includes(highlightVisibleIndex)) {
            highlightVisibleIndex = selectable[0];
        }
    }

    function setHighlightByVisibleIndex(visibleIndex) {
        if (visibleIndex < 0 || visibleIndex >= filteredItems.length) {
            return;
        }
        if (filteredItems[visibleIndex].separator) {
            return;
        }

        highlightVisibleIndex = visibleIndex;
        const item = filteredItems[visibleIndex];

        if (activeListMenuId !== null) {
            menuHighlight(activeListMenuId, item.index);
        }

        const row = listBody.querySelector(`[data-visible-index="${visibleIndex}"]`);
        if (row) {
            row.scrollIntoView({ block: 'nearest' });
        }
    }

    function cycleHighlight(direction) {
        const selectable = visibleSelectableIndexes();
        if (selectable.length === 0) {
            return;
        }

        if (highlightVisibleIndex === -1) {
            setHighlightByVisibleIndex(direction > 0 ? selectable[0] : selectable[selectable.length - 1]);
            return;
        }

        const current = selectable.indexOf(highlightVisibleIndex);
        const next = (current + direction + selectable.length) % selectable.length;
        setHighlightByVisibleIndex(selectable[next]);
    }

    function renderListbox() {
        filteredItems = buildFilteredItems(listItems, query);
        ensureValidHighlight();

        listBody.replaceChildren();

        for (let i = 0; i < filteredItems.length; i++) {
            const item = filteredItems[i];

            if (item.separator) {
                const hr = document.createElement('div');
                hr.className = 'tty-menu-separator';
                listBody.appendChild(hr);
                continue;
            }

            const row = document.createElement('button');
            row.type = 'button';
            row.className = 'tty-menu-row';
            row.dataset.visibleIndex = String(i);
            row.title = item.title;

            if (hasIcons) {
                const icon = document.createElement('span');
                icon.className = 'tty-menu-row-icon';
                icon.textContent = toIconText(item.icon);
                icon.style.opacity = icon.textContent ? '0.9' : '0';
                icon.style.fontFamily = '"Font Awesome Solid", "Font Awesome';
                icon.style.fontWeight = '900';
                row.appendChild(icon);
            }

            const text = document.createElement('span');
            text.className = 'tty-menu-row-label';
            text.textContent = item.title;
            row.appendChild(text);

            if (i === highlightVisibleIndex) {
                row.classList.add('is-active');
            }

            row.addEventListener('mouseenter', () => {
                const prev = listBody.querySelector('.tty-menu-row.is-active');
                if (prev) prev.classList.remove('is-active');
                row.classList.add('is-active');
                highlightVisibleIndex = i;
                if (activeListMenuId !== null) {
                    menuHighlight(activeListMenuId, item.index);
                }
            });

            row.addEventListener('click', (e) => {
                e.stopPropagation();
                e.preventDefault();
                _menuOperationInProgress = true;
                if (activeListMenuId !== null) {
                    menuSelect(activeListMenuId, item.index);
                }
                hideListMenu(false);
                // Allow async clipboard/IO operations to complete, then clear flag
                setTimeout(() => {
                    _menuOperationInProgress = false;
                }, 500);
            });

            listBody.appendChild(row);
        }

        const reserveHeader = 78 + (listSearchWrap.style.display === 'none' ? 0 : 44);
        const idealWidth = measureIdealWidth(filteredItems, listTitle.textContent, hasIcons);
        applyMenuSizing(listRoot, listBody, reserveHeader, idealWidth);
        listRoot.style.display = 'block';
        positionMenu(listRoot);
    }

    function showListMenu(menu) {
        anchorX = mouseX;
        anchorY = mouseY;
        activeListMenuId = menu.menuId;
        listTitle.textContent = menu.title || 'Select an item';
        listTitle.style.display = menu.title ? 'block' : 'none';

        hasIcons = Array.isArray(menu.icons) && menu.icons.length > 0;

        listItems = (menu.options || []).map((title, index) => ({
            title,
            index,
            icon: menu.icons?.[index],
            separator: isSeparatorTitle(title),
        }));

        const firstSelectable = listItems.find((item) => !item.separator);
        if (firstSelectable && activeListMenuId !== null) {
            menuHighlight(activeListMenuId, firstSelectable.index);
        }

        query = '';
        listSearchInput.value = '';
        listSearchWrap.style.display = 'none';
        highlightVisibleIndex = -1;

        renderListbox();
    }

    function showContextMenu(menu) {
        showListMenu(menu);
    }

    window.addEventListener('mousemove', (event) => {
        mouseX = event.clientX;
        mouseY = event.clientY;
    });

    window.addEventListener('mousedown', (event) => {
        mouseX = event.clientX;
        mouseY = event.clientY;

        if (listRoot.style.display === 'none') {
            return;
        }

        if (!listRoot.contains(event.target)) {
            hideMenus(true);
        }
    });

    window.addEventListener('keydown', (event) => {
        if (listRoot.style.display !== 'none') {
            event.preventDefault();
            event.stopPropagation();
            event.stopImmediatePropagation();

            if (event.key === 'Escape') {
                hideListMenu(true);
                return;
            }

            if (event.key === 'Enter') {
                if (highlightVisibleIndex >= 0 && highlightVisibleIndex < filteredItems.length) {
                    const item = filteredItems[highlightVisibleIndex];
                    if (!item.separator && activeListMenuId !== null) {
                        menuSelect(activeListMenuId, item.index);
                        hideListMenu(false);
                    }
                }
                return;
            }

            if (event.key === 'ArrowDown' || (event.key === 'Tab' && !event.shiftKey)) {
                cycleHighlight(1);
                renderListbox();
                return;
            }

            if (event.key === 'ArrowUp' || (event.key === 'Tab' && event.shiftKey)) {
                cycleHighlight(-1);
                renderListbox();
                return;
            }

            if (event.ctrlKey && !event.altKey && !event.metaKey && event.key.toLowerCase() === 'u') {
                query = '';
                listSearchInput.value = '';
                listSearchWrap.style.display = 'none';
                renderListbox();
                return;
            }

            const isTypeable = event.key.length === 1 && !event.ctrlKey && !event.altKey && !event.metaKey;
            if (isTypeable) {
                query += event.key;
                listSearchInput.value = query;
                listSearchWrap.style.display = 'block';
                renderListbox();
                return;
            }

            if (event.key === 'Backspace') {
                query = query.slice(0, -1);
                listSearchInput.value = query;
                listSearchWrap.style.display = query.length > 0 ? 'block' : 'none';
                renderListbox();
            }

            return;
        }
    }, true);

    window.addEventListener('keypress', (event) => {
        if (listRoot.style.display === 'none') {
            return;
        }

        event.preventDefault();
        event.stopPropagation();
        event.stopImmediatePropagation();
    }, true);

    window.addEventListener('keyup', (event) => {
        if (listRoot.style.display === 'none') {
            return;
        }

        event.preventDefault();
        event.stopPropagation();
        event.stopImmediatePropagation();
    }, true);

    window.addEventListener('blur', () => {
        if (listRoot.style.display !== 'none') {
            hideMenus(true);
        }
    });

    _showListMenuFn = showListMenu;
    _setAnchorFn = (x, y) => { mouseX = x; mouseY = y; };

    EventsOn('terminalListMenu', (payload) => {
        const menu = normalizeMenuPayload(payload);
        if (!menu || !Array.isArray(menu.options) || menu.options.length === 0) {
            return;
        }
        showListMenu(menu);
    });

    EventsOn('terminalContextMenu', (payload) => {
        const menu = normalizeMenuPayload(payload);
        if (!menu || !Array.isArray(menu.options) || menu.options.length === 0) {
            return;
        }
        showContextMenu(menu);
    });

    // Suppress native context menus during menu operations
    document.addEventListener('contextmenu', (e) => {
        if (_menuOperationInProgress) {
            e.preventDefault();
            e.stopPropagation();
        }
    }, true); // Use capture phase to intercept early
}

/**
 * Show the terminal popup menu with pure-JS callbacks.
 * Uses negative menu IDs so no Go backend calls are ever made.
 *
 * @param {object} options
 * @param {string|null} [options.title]          - Optional header title
 * @param {string[]}     options.options          - Item labels; '-' produces a separator
 * @param {number[]}    [options.icons]           - Optional icon codepoints
 * @param {number}       options.x               - Client X anchor
 * @param {number}       options.y               - Client Y anchor
 * @param {function(number):void} [options.onSelect]    - Called with item index on selection
 * @param {function(number):void} [options.onHighlight] - Called with item index on hover
 * @param {function(number):void} [options.onCancel]    - Called on dismiss
 */
export function showLocalMenu({ title = null, options = [], icons = [], x = 8, y = 8, onSelect, onHighlight, onCancel } = {}) {
    if (!_showListMenuFn || !_setAnchorFn || options.length === 0) {
        return;
    }

    _localMenuReturnFocus = document.activeElement || null;

    const id = _localNextId--;
    _localCallbacks.set(id, {
        select: onSelect || null,
        highlight: onHighlight || null,
        cancel: onCancel || null,
    });

    _setAnchorFn(x, y);
    _showListMenuFn({ menuId: id, title: title || '', options, icons });
}


