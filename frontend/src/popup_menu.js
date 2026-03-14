import { EventsOn } from '../wailsjs/runtime/runtime';
import { TerminalMenuHighlight, TerminalMenuSelect, TerminalMenuCancel } from '../wailsjs/go/main/WApp';

const MENU_ROOT_ID = 'ttyphoon-popup-menu';

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

function normalizeMenuPayload(payload) {
    if (Array.isArray(payload?.[0])) {
        return payload[0];
    }
    if (Array.isArray(payload)) {
        return payload[0] || payload;
    }
    return payload;
}

function isSeparator(item) {
    return item?.title === '-';
}

export function initTerminalPopupMenu(canvas) {
    if (!canvas) {
        return;
    }

    let mouseX = 8;
    let mouseY = 8;
    let activeMenuId = null;

    const root = document.createElement('div');
    root.id = MENU_ROOT_ID;
    root.style.position = 'fixed';
    root.style.zIndex = '10000';
    root.style.minWidth = '280px';
    root.style.maxWidth = '420px';
    root.style.background = '#1a1a1a';
    root.style.border = '1px solid #3c3c3c';
    root.style.borderRadius = '8px';
    root.style.boxShadow = '0 8px 28px rgba(0,0,0,0.45)';
    root.style.padding = '8px';
    root.style.display = 'none';
    root.style.userSelect = 'none';
    root.style.fontFamily = 'Hasklig, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace';
    root.style.fontSize = '13px';

    const title = document.createElement('div');
    title.style.padding = '6px 8px';
    title.style.color = '#a8a8a8';
    title.style.fontWeight = '600';
    title.style.borderBottom = '1px solid #2f2f2f';
    title.style.marginBottom = '6px';

    const list = document.createElement('div');
    list.style.display = 'flex';
    list.style.flexDirection = 'column';
    list.style.gap = '2px';

    root.appendChild(title);
    root.appendChild(list);
    document.body.appendChild(root);

    function hideMenu(cancel = true) {
        if (activeMenuId !== null && cancel) {
            TerminalMenuCancel(activeMenuId, -1).catch(() => {});
        }

        activeMenuId = null;
        root.style.display = 'none';
        list.replaceChildren();
    }

    function positionMenu() {
        const vw = window.innerWidth;
        const vh = window.innerHeight;
        const rect = root.getBoundingClientRect();

        let x = mouseX;
        let y = mouseY;

        if (x+rect.width > vw - 8) {
            x = Math.max(8, vw - rect.width - 8);
        }
        if (y+rect.height > vh - 8) {
            y = Math.max(8, vh - rect.height - 8);
        }

        root.style.left = `${x}px`;
        root.style.top = `${y}px`;
    }

    function showMenu(menu) {
        activeMenuId = menu.menuId;

        title.textContent = menu.title || 'Menu';
        title.style.display = menu.title ? 'block' : 'none';

        const items = (menu.options || []).map((opt, i) => ({
            title: opt,
            icon: menu.icons?.[i],
            index: i,
        }));

        list.replaceChildren();

        for (const item of items) {
            if (isSeparator(item)) {
                const hr = document.createElement('div');
                hr.style.height = '1px';
                hr.style.margin = '4px 6px';
                hr.style.background = '#2f2f2f';
                list.appendChild(hr);
                continue;
            }

            const row = document.createElement('button');
            row.type = 'button';
            row.style.display = 'flex';
            row.style.alignItems = 'center';
            row.style.width = '100%';
            row.style.border = '0';
            row.style.background = 'transparent';
            row.style.color = '#e6e6e6';
            row.style.padding = '7px 8px';
            row.style.borderRadius = '6px';
            row.style.textAlign = 'left';
            row.style.cursor = 'default';

            const icon = document.createElement('span');
            icon.textContent = toIconText(item.icon);
            icon.style.display = 'inline-block';
            icon.style.width = '18px';
            icon.style.marginRight = '8px';
            icon.style.opacity = icon.textContent ? '0.9' : '0';

            const text = document.createElement('span');
            text.textContent = item.title;

            row.appendChild(icon);
            row.appendChild(text);

            row.addEventListener('mouseenter', () => {
                row.style.background = '#2d4f87';
                if (activeMenuId !== null) {
                    TerminalMenuHighlight(activeMenuId, item.index).catch(() => {});
                }
            });

            row.addEventListener('mouseleave', () => {
                row.style.background = 'transparent';
            });

            row.addEventListener('click', () => {
                if (activeMenuId !== null) {
                    TerminalMenuSelect(activeMenuId, item.index).catch(() => {});
                }
                hideMenu(false);
            });

            list.appendChild(row);
        }

        root.style.display = 'block';
        positionMenu();
    }

    window.addEventListener('mousemove', (event) => {
        mouseX = event.clientX;
        mouseY = event.clientY;
    });

    window.addEventListener('mousedown', (event) => {
        if (root.style.display === 'none') {
            return;
        }

        if (!root.contains(event.target)) {
            hideMenu(true);
        }
    });

    window.addEventListener('keydown', (event) => {
        if (root.style.display === 'none') {
            return;
        }

        if (event.key === 'Escape') {
            event.preventDefault();
            hideMenu(true);
        }
    });

    window.addEventListener('blur', () => {
        if (root.style.display !== 'none') {
            hideMenu(true);
        }
    });

    EventsOn('terminalMenu', (payload) => {
        const menu = normalizeMenuPayload(payload);
        if (!menu || !Array.isArray(menu.options) || menu.options.length === 0) {
            return;
        }

        showMenu(menu);
    });
}
