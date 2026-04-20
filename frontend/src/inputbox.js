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
                <div class="inputbox-hint">Return to confirm &nbsp;&nbsp; Escape to cancel</div>
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
    const inputboxOkBtn = document.getElementById('inputbox-ok');
    const inputboxCancel = document.getElementById('inputbox-cancel');

    if (!inputboxOverlay || !inputboxInputContainer || !inputboxHistoryBtn || !inputboxTitle || !inputboxOkBtn || !inputboxCancel) {
        return;
    }

    let inputboxId = null;
    let inputboxInput = null;
    let inputboxHistoryItems = [];

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
        if (!inputboxInput || inputboxHistoryItems.length === 0) {
            return;
        }

        const rect = inputboxHistoryBtn.getBoundingClientRect();
        showLocalMenu({
            title: 'History',
            options: inputboxHistoryItems,
            x: rect.left,
            y: rect.bottom,
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
    });

    // Clicks on the backdrop (outside the dialog) cancel.
    inputboxOverlay.addEventListener('click', (e) => {
        if (e.target === inputboxOverlay) {
            inputboxSubmit(false);
        }
    });

    EventsOn('terminalInputBox', (payload) => {
        const p = Array.isArray(payload?.[0]) ? payload[0] : payload;
        if (!p) {
            return;
        }

        inputboxId = p.id;
        inputboxTitle.textContent = p.title ?? '';
        inputboxInputContainer.innerHTML = '';

        if (p.multiline) {
            inputboxInput = document.createElement('textarea');
            inputboxInput.className = 'inputbox-input';
            inputboxInput.rows = 2;
            inputboxInput.value = p.defaultValue ?? '';
            inputboxInput.placeholder = p.placeholder ?? '';
            inputboxInput.setAttribute('autocomplete', 'off');
            inputboxInput.setAttribute('spellcheck', 'false');
            inputboxInput.style.resize = 'none';
            inputboxInput.addEventListener('input', () => autoGrowTextarea(inputboxInput));
            setTimeout(() => autoGrowTextarea(inputboxInput), 0);
            inputboxInput.addEventListener('keydown', (e) => {
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
            inputboxInput.setAttribute('spellcheck', 'false');
            inputboxInput.addEventListener('keydown', (e) => {
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
