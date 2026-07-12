import { EventsOn } from '../wailsjs/runtime/runtime';
import {
    CommandPaletteSelect,
    FilterStrings,
    TerminalMenuHighlight,
    TerminalMenuSelect,
    TerminalMenuCancel,
    TerminalRequestRedraw,
} from '../wailsjs/go/main/WApp';

const LISTBOX_ROOT_ID = 'ttyphoon-listbox-menu';
const PASSIVE_MENU_ROOT_ID = 'ttyphoon-passive-menu';

// Registry for pure-JS context menus with negative IDs (never forwarded to Go).
const _localCallbacks = new Map();
let _localNextId = -1;
let _showListMenuFn = null;
let _setAnchorFn = null;
let _menuOperationInProgress = false;
let _localMenuReturnFocus = null;
let _listMenuTransitionSeq = 0;
let _goFilterSeq = 0;
let _passiveMenuRoot = null;
let _passiveMenuTitle = null;
let _passiveMenuBody = null;
let _passiveMenuVisible = false;
let _passiveMenuRows = [];
let _passiveMenuHighlightIndex = -1;
let _passiveMenuOnSelect = null;
let _passiveMenuOnHighlight = null;
let _passiveMenuOnCancel = null;
let _passiveMenuShowNextToMouse = true;
let _passiveMenuAnchorX = 8;
let _passiveMenuAnchorY = 8;
let _passiveMenuVisibleRows = 12;
let _passiveMenuRootClassName = '';

function menuHighlight(id, index) {
    if (id < 0) {
        _localCallbacks.get(id)?.highlight?.(index);
        return;
    }
    TerminalMenuHighlight(id, index).catch(() => { });
    TerminalRequestRedraw().catch(() => { });
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
    TerminalMenuSelect(id, index).catch(() => { });
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
    TerminalMenuCancel(id, index).catch(() => { });
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

function normalizeColumnAlign(value) {
    const align = String(value || '').toLowerCase();
    if (align === 'center' || align === 'right') {
        return align;
    }
    return 'left';
}

function normalizePassiveRows(rows = []) {
    return rows.map((row, rowIndex) => {
        if (row === '-' || row?.separator === true) {
            return {
                id: rowIndex,
                separator: true,
                selectable: false,
                columns: [],
            };
        }

        if (typeof row === 'string') {
            return {
                id: rowIndex,
                separator: false,
                selectable: true,
                deprecated: false,
                columns: [{
                    text: row,
                    align: 'left',
                    color: '',
                    className: '',
                    grow: true,
                }],
            };
        }

        const columns = Array.isArray(row?.columns) && row.columns.length > 0
            ? row.columns
            : [{ text: String(row?.title || ''), grow: true }];

        return {
            id: rowIndex,
            separator: false,
            selectable: row?.selectable !== false,
            deprecated: row?.deprecated === true,
            rowClassName: String(row?.rowClassName || ''),
            columns: columns.map((column) => ({
                text: String(column?.text || ''),
                align: normalizeColumnAlign(column?.align),
                color: String(column?.color || ''),
                className: String(column?.className || ''),
                grow: column?.grow !== false,
                title: String(column?.title || ''),
            })),
        };
    });
}

function firstSelectablePassiveRowIndex() {
    for (let i = 0; i < _passiveMenuRows.length; i += 1) {
        const row = _passiveMenuRows[i];
        if (!row.separator && row.selectable) {
            return i;
        }
    }
    return -1;
}

function clampPassiveMenuPosition() {
    if (!_passiveMenuRoot) {
        return;
    }

    const rect = _passiveMenuRoot.getBoundingClientRect();
    const vw = window.innerWidth;
    const vh = window.innerHeight;

    let x = _passiveMenuAnchorX;
    let y = _passiveMenuAnchorY;

    if (!_passiveMenuShowNextToMouse) {
        x = Math.round((vw - rect.width) / 2);
        y = 14;
    }

    if (x + rect.width > vw - 8) {
        x = Math.max(8, vw - rect.width - 8);
    }
    if (y + rect.height > vh - 8) {
        y = Math.max(8, vh - rect.height - 8);
    }

    _passiveMenuRoot.style.left = `${x}px`;
    _passiveMenuRoot.style.top = `${y}px`;
}

function ensurePassiveMenuDom() {
    if (_passiveMenuRoot && _passiveMenuRoot.isConnected) {
        return;
    }

    _passiveMenuRoot = document.createElement('div');
    _passiveMenuRoot.id = PASSIVE_MENU_ROOT_ID;
    _passiveMenuRoot.className = 'tty-menu tty-passive-menu';
    _passiveMenuRoot.style.display = 'none';

    _passiveMenuTitle = document.createElement('div');
    _passiveMenuTitle.className = 'tty-menu-title';

    _passiveMenuBody = document.createElement('div');
    _passiveMenuBody.className = 'tty-menu-list';

    _passiveMenuRoot.appendChild(_passiveMenuTitle);
    _passiveMenuRoot.appendChild(_passiveMenuBody);
    document.body.appendChild(_passiveMenuRoot);

    _passiveMenuRoot.addEventListener('mousedown', (event) => {
        const rowElement = event.target instanceof Element
            ? event.target.closest('[data-passive-menu-index]')
            : null;
        if (!rowElement) {
            return;
        }

        event.preventDefault();
        event.stopPropagation();

        const rowIndex = Number.parseInt(rowElement.dataset.passiveMenuIndex || '-1', 10);
        if (!Number.isInteger(rowIndex) || rowIndex < 0 || rowIndex >= _passiveMenuRows.length) {
            return;
        }

        const row = _passiveMenuRows[rowIndex];
        if (!row || row.separator || !row.selectable) {
            return;
        }

        _passiveMenuHighlightIndex = rowIndex;
        _passiveMenuOnSelect?.(rowIndex);
        hidePassiveLocalMenu(false);
    });

    _passiveMenuRoot.addEventListener('mousemove', (event) => {
        const rowElement = event.target instanceof Element
            ? event.target.closest('[data-passive-menu-index]')
            : null;
        if (!rowElement) {
            return;
        }

        const rowIndex = Number.parseInt(rowElement.dataset.passiveMenuIndex || '-1', 10);
        if (!Number.isInteger(rowIndex) || rowIndex < 0 || rowIndex >= _passiveMenuRows.length) {
            return;
        }

        const row = _passiveMenuRows[rowIndex];
        if (!row || row.separator || !row.selectable || rowIndex === _passiveMenuHighlightIndex) {
            return;
        }

        _passiveMenuHighlightIndex = rowIndex;
        _passiveMenuOnHighlight?.(rowIndex);
        renderPassiveLocalMenu();
    });
}

function renderPassiveLocalMenu() {
    if (!_passiveMenuRoot || !_passiveMenuBody || !_passiveMenuTitle) {
        return;
    }

    _passiveMenuBody.replaceChildren();

    _passiveMenuRows.forEach((row, rowIndex) => {
        if (row.separator) {
            const hr = document.createElement('div');
            hr.className = 'tty-menu-separator';
            _passiveMenuBody.appendChild(hr);
            return;
        }

        const item = document.createElement('button');
        item.type = 'button';
        item.className = 'tty-menu-row tty-menu-rich-row';
        item.dataset.passiveMenuIndex = String(rowIndex);
        item.classList.toggle('is-active', rowIndex === _passiveMenuHighlightIndex);
        item.classList.toggle('is-deprecated', row.deprecated === true);
        if (row.rowClassName) {
            item.classList.add(...row.rowClassName.split(/\s+/).filter(Boolean));
        }

        if (!row.selectable) {
            item.disabled = true;
        }

        row.columns.forEach((column) => {
            const col = document.createElement('span');
            col.className = `tty-menu-rich-col align-${column.align}`;
            if (column.className) {
                col.classList.add(...column.className.split(/\s+/).filter(Boolean));
            }

            if (column.grow) {
                col.classList.add('is-grow');
            }

            if (column.color) {
                col.style.color = column.color;
            }

            if (column.title) {
                col.title = column.title;
            }

            col.textContent = column.text;
            item.appendChild(col);
        });

        _passiveMenuBody.appendChild(item);
    });

    _passiveMenuBody.style.maxHeight = `calc(${Math.max(4, _passiveMenuVisibleRows)} * var(--terminal-menu-font-size) + 16px)`;

    const hasTitle = _passiveMenuTitle.textContent.trim().length > 0;
    _passiveMenuTitle.style.display = hasTitle ? 'block' : 'none';

    _passiveMenuRoot.classList.remove('hide');
    _passiveMenuRoot.classList.add('show');
    _passiveMenuRoot.style.display = 'block';
    clampPassiveMenuPosition();

    const activeRow = _passiveMenuBody.querySelector(`[data-passive-menu-index="${_passiveMenuHighlightIndex}"]`);
    if (activeRow && typeof activeRow.scrollIntoView === 'function') {
        activeRow.scrollIntoView({ block: 'nearest' });
    }
}

function measureIdealWidth(items, title, withIcons) {
    const c = document.createElement('canvas');
    const ctx = c.getContext('2d');
    const rootStyle = getComputedStyle(document.documentElement);
    const fontFamily = rootStyle.getPropertyValue('--terminal-menu-font').trim()
        || rootStyle.getPropertyValue('--font-family').trim()
        || getComputedStyle(document.body).fontFamily;
    const fontSizeVar = rootStyle.getPropertyValue('--terminal-menu-font-size').trim();
    const parsedFontSize = Number.parseFloat(fontSizeVar);
    const fontSize = (Number.isFinite(parsedFontSize) && parsedFontSize > 0 ? parsedFontSize : 14);
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

const TOP_CENTER_MENU_OFFSET_Y = 14;

function tokenizeQuery(q) {
    return (q || '').toLowerCase().trim().split(/\s+/).filter(Boolean);
}

export function buildFilteredItems(items, query, visibleSet = null) {
    const tokens = visibleSet === null ? tokenizeQuery(query) : null;

    const raw = items.map((item) => {
        if (item.separator) {
            return { ...item, visible: true };
        }

        if (visibleSet !== null) {
            return { ...item, visible: visibleSet.has(item.title) };
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

    // Collapse runs of separators into a single separator.
    const deduped = [];
    for (let i = 0; i < result.length; i++) {
        const item = result[i];
        if (item.separator && deduped.length > 0 && deduped[deduped.length - 1].separator) {
            continue;
        }
        deduped.push(item);
    }

    return deduped;
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
    let showSearch = false;
    let hideItemsUntilQuery = false;
    let showNextToMouseCursor = false;
    let mouseHighlightEnabled = true;

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
    listSearchInput.setAttribute('autocomplete', 'off');
    listSearchInput.setAttribute('autocorrect', 'off');
    listSearchInput.setAttribute('autocapitalize', 'off');
    listSearchInput.setAttribute('spellcheck', 'false');
    listSearchWrap.appendChild(listSearchInput);

    const listBody = document.createElement('div');
    listBody.className = 'tty-menu-list';

    listRoot.appendChild(listTitle);
    listRoot.appendChild(listSearchWrap);
    listRoot.appendChild(listBody);
    document.body.appendChild(listRoot);

    function syncMouseHighlightState() {
        listRoot.dataset.hoverHighlight = mouseHighlightEnabled ? 'true' : 'false';
    }

    syncMouseHighlightState();

    function menuConstraints() {
        let rect = canvas?.getBoundingClientRect();

        // When notes are embedded, the terminal canvas can be hidden (0x0).
        // Fall back to the visible host so menu sizing does not collapse.
        if (!rect || rect.width < 120 || rect.height < 120) {
            const embeddedNotesHost = document.getElementById('terminal-jupyter-host');
            const hostVisible = embeddedNotesHost && embeddedNotesHost.style.display !== 'none';
            const fallbackRoot = hostVisible
                ? embeddedNotesHost
                : (document.getElementById('terminal-viewport') || document.getElementById('notes-pane') || document.body);
            rect = fallbackRoot.getBoundingClientRect();
        }

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

        if (!showNextToMouseCursor) {
            x = Math.round((vw - rect.width) / 2);
            y = TOP_CENTER_MENU_OFFSET_Y;
        }

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
        const transitionSeq = ++_listMenuTransitionSeq;
        _goFilterSeq++; // cancel any pending async filter render

        if (activeListMenuId !== null && cancel) {
            menuCancel(activeListMenuId, -1);
        }

        activeListMenuId = null;
        listItems = [];
        filteredItems = [];
        highlightVisibleIndex = -1;
        hasIcons = false;
        query = '';
        showSearch = false;
        hideItemsUntilQuery = false;
        showNextToMouseCursor = false;
        mouseHighlightEnabled = true;
        syncMouseHighlightState();
        listSearchInput.value = '';
        listSearchWrap.style.display = 'none';

        listRoot.classList.remove('show');
        listRoot.classList.add('hide');

        const onAnimationEnd = () => {
            if (transitionSeq !== _listMenuTransitionSeq) {
                return;
            }

            listRoot.removeEventListener('animationend', onAnimationEnd);
            listRoot.classList.remove('hide');
            listRoot.style.display = 'none';
            listBody.replaceChildren();
        };

        listRoot.addEventListener('animationend', onAnimationEnd, { once: true });
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

    function setHighlightFromMouse(visibleIndex) {
        if (!mouseHighlightEnabled) {
            return;
        }

        if (highlightVisibleIndex === visibleIndex) {
            return;
        }

        setHighlightByVisibleIndex(visibleIndex);
        renderDOM();
    }

    function selectMenuItem(item) {
        _menuOperationInProgress = true;
        if (activeListMenuId !== null) {
            menuSelect(activeListMenuId, item.index);
        }
        hideListMenu(false);
        // Allow async clipboard/IO operations to complete, then clear flag
        setTimeout(() => {
            _menuOperationInProgress = false;
        }, 500);
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

    function renderDOM() {
        ensureValidHighlight();

        if (highlightVisibleIndex >= 0 && activeListMenuId !== null) {
            const item = filteredItems[highlightVisibleIndex];
            if (item && !item.separator) {
                menuHighlight(activeListMenuId, item.index);
            }
        }

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

            row.addEventListener('mousemove', () => {
                setHighlightFromMouse(i);
            });

            row.addEventListener('click', (e) => {
                e.stopPropagation();
                e.preventDefault();
                if (e.button !== 0) {
                    return;
                }
                selectMenuItem(item);
            });

            row.addEventListener('mousedown', (e) => {
                if (e.button !== 2) {
                    return;
                }

                e.stopPropagation();
                e.preventDefault();
                selectMenuItem(item);
            });

            listBody.appendChild(row);
        }

        const reserveHeader = 78 + (listSearchWrap.style.display === 'none' ? 0 : 44);
        const idealWidth = measureIdealWidth(filteredItems, listTitle.textContent, hasIcons);
        applyMenuSizing(listRoot, listBody, reserveHeader, idealWidth);
        listRoot.classList.remove('hide');
        listRoot.classList.add('show');
        listRoot.style.display = 'block';
        positionMenu(listRoot);
    }

    async function renderListbox() {
        const seq = ++_goFilterSeq;

        if (hideItemsUntilQuery && query.length === 0) {
            filteredItems = [];
        } else if (query.trim()) {
            try {
                const titles = listItems
                    .filter((item) => !item.separator)
                    .map((item) => item.title);
                // FilterStrings now returns { List, Error }. An Error (malformed
                // query or zero matches) yields an empty match set.
                const result = await FilterStrings(query, titles);
                if (seq !== _goFilterSeq) {
                    return;
                }
                const matched = result && Array.isArray(result.List) ? result.List : [];
                filteredItems = buildFilteredItems(listItems, query, new Set(matched));
            } catch {
                filteredItems = buildFilteredItems(listItems, query);
            }
        } else {
            filteredItems = buildFilteredItems(listItems, query);
        }

        if (seq !== _goFilterSeq) {
            return;
        }
        renderDOM();
    }

    function showListMenu(menu) {
        // Invalidate any pending hide callback from a previous menu instance.
        _listMenuTransitionSeq++;

        anchorX = mouseX;
        anchorY = mouseY;
        activeListMenuId = menu.menuId;
        listTitle.textContent = menu.title || 'Select an item';
        listTitle.style.display = menu.title ? 'block' : 'none';

        hasIcons = Array.isArray(menu.icons) && menu.icons.length > 0;
        showSearch = Boolean(menu.showSearch);
        hideItemsUntilQuery = Boolean(menu.hideItemsUntilQuery);
        showNextToMouseCursor = Boolean(menu.showNextToMouseCursor);

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
        listSearchWrap.style.display = showSearch ? 'block' : 'none';
        highlightVisibleIndex = -1;
        mouseHighlightEnabled = true;
        syncMouseHighlightState();

        renderListbox();

        if (showSearch) {
            listSearchInput.focus();
        }
    }

    function showContextMenu(menu) {
        showListMenu(menu);
    }

    window.addEventListener('mousemove', (event) => {
        if (!listRoot.isConnected) {
            return;
        }

        mouseX = event.clientX;
        mouseY = event.clientY;

        if (listRoot.style.display !== 'none') {
            mouseHighlightEnabled = true;
            syncMouseHighlightState();
        }
    });

    window.addEventListener('mousedown', (event) => {
        if (!listRoot.isConnected) {
            return;
        }

        mouseX = event.clientX;
        mouseY = event.clientY;

        if (listRoot.style.display === 'none') {
            return;
        }

        if (!listRoot.contains(event.target)) {
            event.preventDefault();
            event.stopPropagation();
            event.stopImmediatePropagation();
            hideMenus(true);

            // Swallow the mouseup and click that will follow this mousedown so
            // they don't activate whatever is underneath the menu.
            const swallow = (e) => {
                e.stopPropagation();
                e.stopImmediatePropagation();
            };
            window.addEventListener('mouseup', swallow, { capture: true, once: true });
            window.addEventListener('click', swallow, { capture: true, once: true });
        }
    }, true);

    window.addEventListener('keydown', (event) => {
        if (listRoot.isConnected && listRoot.style.display !== 'none') {
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
                mouseHighlightEnabled = false;
                syncMouseHighlightState();
                cycleHighlight(1);
                renderDOM();
                return;
            }

            if (event.key === 'ArrowUp' || (event.key === 'Tab' && event.shiftKey)) {
                mouseHighlightEnabled = false;
                syncMouseHighlightState();
                cycleHighlight(-1);
                renderDOM();
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
        if (!listRoot.isConnected || listRoot.style.display === 'none') {
            return;
        }

        event.preventDefault();
        event.stopPropagation();
        event.stopImmediatePropagation();
    }, true);

    window.addEventListener('keyup', (event) => {
        if (!listRoot.isConnected || listRoot.style.display === 'none') {
            return;
        }

        event.preventDefault();
        event.stopPropagation();
        event.stopImmediatePropagation();
    }, true);

    window.addEventListener('blur', () => {
        if (!listRoot.isConnected) {
            return;
        }

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
        menu.showNextToMouseCursor = true;
        showContextMenu(menu);
    });

    EventsOn('commandPaletteOpen', (payload) => {
        const menu = normalizeMenuPayload(payload);
        const options = Array.isArray(menu?.options) ? menu.options : [];
        if (options.length === 0) {
            return;
        }

        showLocalMenu({
            title: menu?.title || 'Command Palette',
            options: options.map((item) => item?.title ?? ''),
            icons: options.map((item) => item?.icon ?? 0),
            x: Math.round(window.innerWidth / 2),
            y: 14,
            showSearch: true,
            hideItemsUntilQuery: true,
            onSelect: (index) => {
                CommandPaletteSelect(index).catch(() => { });
            },
        });
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
 * @param {boolean} [options.showSearch]                - Force search field visible on open
 * @param {boolean} [options.hideItemsUntilQuery]       - Keep list empty until search query is typed
 * @param {boolean} [options.showNextToMouseCursor]     - When true, anchor near cursor; otherwise top-center
 */
export function showLocalMenu({
    title = null,
    options = [],
    icons = [],
    x = 8,
    y = 8,
    onSelect,
    onHighlight,
    onCancel,
    showSearch = false,
    hideItemsUntilQuery = false,
    showNextToMouseCursor = false,
} = {}) {
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
    _showListMenuFn({
        menuId: id,
        title: title || '',
        options,
        icons,
        showSearch,
        hideItemsUntilQuery,
        showNextToMouseCursor,
    });
}

function isSelectablePassiveMenuIndex(index) {
    if (!Number.isInteger(index) || index < 0 || index >= _passiveMenuRows.length) {
        return false;
    }

    const row = _passiveMenuRows[index];
    return Boolean(row && !row.separator && row.selectable);
}

function ensurePassiveHighlightIndex(index = -1) {
    if (isSelectablePassiveMenuIndex(index)) {
        _passiveMenuHighlightIndex = index;
        return;
    }

    _passiveMenuHighlightIndex = firstSelectablePassiveRowIndex();
}

export function isPassiveLocalMenuVisible() {
    return _passiveMenuVisible === true;
}

export function hidePassiveLocalMenu(cancel = true) {
    if (!_passiveMenuVisible) {
        return;
    }

    _passiveMenuVisible = false;

    if (_passiveMenuRoot) {
        _passiveMenuRoot.classList.remove('show');
        _passiveMenuRoot.classList.add('hide');
        _passiveMenuRoot.style.display = 'none';
    }

    _passiveMenuRootClassName = '';

    if (cancel) {
        _passiveMenuOnCancel?.(_passiveMenuHighlightIndex);
    }
}

export function setPassiveLocalMenuHighlight(index) {
    if (!_passiveMenuVisible) {
        return false;
    }

    if (!isSelectablePassiveMenuIndex(index)) {
        return false;
    }

    _passiveMenuHighlightIndex = index;
    _passiveMenuOnHighlight?.(index);
    renderPassiveLocalMenu();
    return true;
}

export function selectPassiveLocalMenu(index = _passiveMenuHighlightIndex) {
    if (!_passiveMenuVisible || !isSelectablePassiveMenuIndex(index)) {
        return false;
    }

    _passiveMenuHighlightIndex = index;
    _passiveMenuOnSelect?.(index);
    hidePassiveLocalMenu(false);
    return true;
}

export function showPassiveLocalMenu({
    title = null,
    rows = [],
    x = 8,
    y = 8,
    onSelect,
    onHighlight,
    onCancel,
    highlightIndex = -1,
    visibleRows = 12,
    showNextToMouseCursor = true,
    rootClassName = '',
} = {}) {
    if (!Array.isArray(rows) || rows.length === 0) {
        hidePassiveLocalMenu(true);
        return;
    }

    ensurePassiveMenuDom();

    _passiveMenuRows = normalizePassiveRows(rows);
    _passiveMenuOnSelect = typeof onSelect === 'function' ? onSelect : null;
    _passiveMenuOnHighlight = typeof onHighlight === 'function' ? onHighlight : null;
    _passiveMenuOnCancel = typeof onCancel === 'function' ? onCancel : null;
    _passiveMenuShowNextToMouse = showNextToMouseCursor === true;
    _passiveMenuAnchorX = Number.isFinite(x) ? x : 8;
    _passiveMenuAnchorY = Number.isFinite(y) ? y : 8;
    _passiveMenuVisibleRows = Number.isFinite(visibleRows) ? Math.max(4, Math.floor(visibleRows)) : 12;
    _passiveMenuRootClassName = String(rootClassName || '').trim();
    _passiveMenuTitle.textContent = title ? String(title) : '';
    _passiveMenuVisible = true;

    if (_passiveMenuRoot) {
        _passiveMenuRoot.className = ['tty-menu', 'tty-passive-menu', _passiveMenuRootClassName]
            .filter(Boolean)
            .join(' ');
    }

    ensurePassiveHighlightIndex(Number.isFinite(highlightIndex) ? Math.floor(highlightIndex) : -1);
    if (_passiveMenuHighlightIndex >= 0) {
        _passiveMenuOnHighlight?.(_passiveMenuHighlightIndex);
    }
    renderPassiveLocalMenu();
}

export function updatePassiveLocalMenu({
    title,
    rows,
    x,
    y,
    highlightIndex,
    visibleRows,
} = {}) {
    if (!_passiveMenuVisible) {
        return false;
    }

    if (typeof title !== 'undefined' && _passiveMenuTitle) {
        _passiveMenuTitle.textContent = title ? String(title) : '';
    }

    if (Array.isArray(rows)) {
        _passiveMenuRows = normalizePassiveRows(rows);
    }

    if (Number.isFinite(x)) {
        _passiveMenuAnchorX = x;
    }

    if (Number.isFinite(y)) {
        _passiveMenuAnchorY = y;
    }

    if (Number.isFinite(visibleRows)) {
        _passiveMenuVisibleRows = Math.max(4, Math.floor(visibleRows));
    }

    if (Number.isFinite(highlightIndex)) {
        ensurePassiveHighlightIndex(Math.floor(highlightIndex));
    } else if (!isSelectablePassiveMenuIndex(_passiveMenuHighlightIndex)) {
        ensurePassiveHighlightIndex(-1);
    }

    renderPassiveLocalMenu();
    return true;
}

window.addEventListener('mousedown', (event) => {
    if (!_passiveMenuVisible || !_passiveMenuRoot) {
        return;
    }

    if (_passiveMenuRoot.contains(event.target)) {
        return;
    }

    hidePassiveLocalMenu(true);
}, true);

window.addEventListener('resize', () => {
    if (!_passiveMenuVisible) {
        return;
    }
    clampPassiveMenuPosition();
});

window.addEventListener('blur', () => {
    if (_passiveMenuVisible) {
        hidePassiveLocalMenu(true);
    }
});


