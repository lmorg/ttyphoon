import { TerminalInputBoxSubmit } from '../wailsjs/go/main/WApp';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { showLocalMenu } from './popup_menu';
import './inputbox.css';

function ensureInputBoxDom() {
    if (document.getElementById('terminal-inputbox')) {
        return;
    }

    const root = document.getElementById('terminal-app') || document.getElementById('terminal-pane') || document.querySelector('#app');
    if (!root) {
        return;
    }

    const wrapper = document.createElement('div');
    wrapper.innerHTML = `
        <div id="terminal-inputbox" class="inputbox-overlay" style="display:none">
            <div class="inputbox-dialog">
                <div class="inputbox-title" id="inputbox-title"></div>
                <div id="inputbox-input-container"></div>
                <div class="inputbox-hint" id="inputbox-hint">
                    <span id="inputbox-confirm-hint">Return to confirm</span>
                    <span>Escape to cancel</span>
                </div>
                <div class="inputbox-buttons">
                    <button class="inputbox-btn inputbox-ok" id="inputbox-ok">OK</button>
                    <button class="inputbox-btn inputbox-cancel" id="inputbox-cancel">Cancel</button>
                    <button class="inputbox-btn inputbox-history-btn" id="inputbox-history-btn" title="History" aria-label="History" style="display:none">&#xf141;</button>
                </div>
            </div>
        </div>
    `;
    root.appendChild(wrapper.firstElementChild);
}

export function initInputBox(canvas) {
    ensureInputBoxDom();

    const inputboxOverlay = document.getElementById('terminal-inputbox');
    const inputboxInputContainer = document.getElementById('inputbox-input-container');
    const inputboxHistoryBtn = document.getElementById('inputbox-history-btn');
    const inputboxTitle = document.getElementById('inputbox-title');
    const inputboxConfirmHint = document.getElementById('inputbox-confirm-hint');
    const inputboxOkBtn = document.getElementById('inputbox-ok');
    const inputboxCancel = document.getElementById('inputbox-cancel');

    if (!inputboxOverlay || !inputboxInputContainer || !inputboxHistoryBtn || !inputboxTitle || !inputboxConfirmHint || !inputboxOkBtn || !inputboxCancel) {
        return;
    }

    let inputboxId = null;
    let inputboxInput = null;
    let inputboxHistoryItems = [];
    let backdropPointerDown = false;

    function openInputboxHistoryMenu(x, y) {
        if (!inputboxInput || inputboxHistoryItems.length === 0) {
            return;
        }

        showLocalMenu({
            title: 'History',
            options: inputboxHistoryItems,
            x,
            y,
            showNextToMouseCursor: true,
            onSelect: (index) => {
                const value = inputboxHistoryItems[index];
                if (!value || !inputboxInput) {
                    return;
                }

                inputboxInput.value = value;
                inputboxInput.focus();

                if (inputboxInput.tagName === 'TEXTAREA') {
                    autoGrowTextarea(inputboxInput);
                } else if (typeof inputboxInput.select === 'function') {
                    inputboxInput.select();
                }
            },
        });
    }

    function shouldOpenHistoryHotkey(e) {
        return e.ctrlKey && !e.altKey && !e.metaKey && !e.shiftKey && e.key.toLowerCase() === 'h';
    }

    function shouldOpenHistoryUpArrow(e) {
        return e.key === 'ArrowUp' && inputboxInput && inputboxInput.value.length === 0;
    }

    function shouldClearInputbox(e) {
        return e.ctrlKey && !e.altKey && !e.metaKey && !e.shiftKey && e.key.toLowerCase() === 'u';
    }

    function handleHistoryHotkeys(e) {
        if (!inputboxInput || inputboxHistoryItems.length === 0) {
            return false;
        }

        if (shouldOpenHistoryHotkey(e) || shouldOpenHistoryUpArrow(e)) {
            e.preventDefault();
            const rect = inputboxInput.getBoundingClientRect();
            openInputboxHistoryMenu(rect.left, rect.bottom);
            return true;
        }

        return false;
    }

    function handleInputboxHotkeys(e) {
        if (!inputboxInput) {
            return false;
        }

        if (shouldClearInputbox(e)) {
            e.preventDefault();
            inputboxInput.value = '';
            if (inputboxInput.tagName === 'TEXTAREA') {
                autoGrowTextarea(inputboxInput);
            }
            return true;
        }

        return handleHistoryHotkeys(e);
    }

    function autoGrowTextarea(textarea) {
        textarea.style.height = 'auto';
        const maxHeight = Math.max(120, window.innerHeight - 220);
        const nextHeight = Math.min(textarea.scrollHeight + 2, maxHeight);
        textarea.style.maxHeight = `${maxHeight}px`;
        textarea.style.height = `${nextHeight}px`;
        textarea.style.overflowY = textarea.scrollHeight + 2 > maxHeight ? 'auto' : 'hidden';
    }

    function inputboxSubmit(isOk) {
        if (inputboxId === null || !inputboxInput) {
            return;
        }

        const value = inputboxInput.value;
        const id = inputboxId;
        inputboxId = null;

        inputboxOverlay.style.display = 'none';
        if (canvas) {
            canvas.focus();
        }

        TerminalInputBoxSubmit(id, value, isOk).catch(() => {});
    }

    inputboxOkBtn.addEventListener('click', () => inputboxSubmit(true));
    inputboxCancel.addEventListener('click', () => inputboxSubmit(false));

    inputboxHistoryBtn.addEventListener('click', () => {
        const rect = inputboxHistoryBtn.getBoundingClientRect();
        openInputboxHistoryMenu(rect.left, rect.bottom);
    });

    // Only close when both pointer down and pointer up happen on the backdrop.
    // This avoids accidental dismiss when selecting text and releasing outside.
    inputboxOverlay.addEventListener('pointerdown', (e) => {
        backdropPointerDown = e.target === inputboxOverlay;
    });

    inputboxOverlay.addEventListener('pointerup', (e) => {
        const shouldClose = backdropPointerDown && e.target === inputboxOverlay;
        backdropPointerDown = false;
        if (shouldClose) {
            inputboxSubmit(false);
        }
    });

    inputboxOverlay.addEventListener('pointercancel', () => {
        backdropPointerDown = false;
    });

    EventsOn('terminalInputBox', (payload) => {
        const p = Array.isArray(payload?.[0]) ? payload[0] : payload;
        if (!p) {
            return;
        }

        inputboxId = p.id;
        backdropPointerDown = false;
        inputboxTitle.textContent = p.title ?? '';
        inputboxConfirmHint.textContent = p.multiline
            ? 'Ctrl+Return to confirm'
            : 'Return to confirm';
        inputboxInputContainer.innerHTML = '';

        if (p.multiline) {
            inputboxInput = document.createElement('textarea');
            inputboxInput.className = 'inputbox-input';
            inputboxInput.rows = 2;
            inputboxInput.value = p.defaultValue ?? '';
            inputboxInput.placeholder = p.placeholder ?? '';
            inputboxInput.setAttribute('autocomplete', 'off');
            //inputboxInput.setAttribute('spellcheck', 'false');
            inputboxInput.style.resize = 'none';
            inputboxInput.addEventListener('input', () => autoGrowTextarea(inputboxInput));
            setTimeout(() => autoGrowTextarea(inputboxInput), 0);
            inputboxInput.addEventListener('keydown', (e) => {
                if (handleInputboxHotkeys(e)) {
                    e.stopPropagation();
                    return;
                }

                if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
                    e.preventDefault();
                    inputboxSubmit(true);
                }
                if (e.key === 'Escape') {
                    e.preventDefault();
                    inputboxSubmit(false);
                }
                e.stopPropagation();
            });
        } else {
            inputboxInput = document.createElement('input');
            inputboxInput.className = 'inputbox-input';
            inputboxInput.type = 'text';
            inputboxInput.value = p.defaultValue ?? '';
            inputboxInput.placeholder = p.placeholder ?? '';
            inputboxInput.setAttribute('autocomplete', 'off');
            //inputboxInput.setAttribute('spellcheck', 'false');
            inputboxInput.addEventListener('keydown', (e) => {
                if (handleInputboxHotkeys(e)) {
                    e.stopPropagation();
                    return;
                }

                if (e.key === 'Enter') {
                    e.preventDefault();
                    inputboxSubmit(true);
                }
                if (e.key === 'Escape') {
                    e.preventDefault();
                    inputboxSubmit(false);
                }
                e.stopPropagation();
            });
        }

        inputboxHistoryItems = Array.isArray(p.history) ? p.history : [];
        inputboxHistoryBtn.style.display = inputboxHistoryItems.length > 0 ? 'inline-flex' : 'none';

        inputboxInputContainer.appendChild(inputboxInput);
        inputboxOverlay.style.display = 'flex';
        setTimeout(() => {
            inputboxInput.focus();
            if (typeof inputboxInput.select === 'function') {
                inputboxInput.select();
            }
        }, 0);
    });
}
