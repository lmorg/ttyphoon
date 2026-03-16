import { TerminalInputBoxSubmit } from '../wailsjs/go/main/WApp';
import { EventsOn } from '../wailsjs/runtime/runtime';
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
                <select id="inputbox-history" class="inputbox-history"></select>
                <div class="inputbox-buttons">
                    <button class="inputbox-btn inputbox-ok" id="inputbox-ok">OK</button>
                    <button class="inputbox-btn inputbox-cancel" id="inputbox-cancel">Cancel</button>
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
    const inputboxHistory = document.getElementById('inputbox-history');
    const inputboxTitle = document.getElementById('inputbox-title');
    const inputboxOkBtn = document.getElementById('inputbox-ok');
    const inputboxCancel = document.getElementById('inputbox-cancel');

    if (!inputboxOverlay || !inputboxInputContainer || !inputboxHistory || !inputboxTitle || !inputboxOkBtn || !inputboxCancel) {
        return;
    }

    let inputboxId = null;
    let inputboxInput = null;

    function autoGrowTextarea(textarea) {
        textarea.style.height = 'auto';
        textarea.style.height = `${textarea.scrollHeight + 2}px`;
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

    // Clicks on the backdrop (outside the dialog) cancel.
    inputboxOverlay.addEventListener('click', (e) => {
        if (e.target === inputboxOverlay) {
            inputboxSubmit(false);
        }
    });

    inputboxHistory.addEventListener('change', (e) => {
        if (!inputboxInput || !e.target.value) {
            return;
        }

        inputboxInput.value = e.target.value;
        e.target.value = '';
        inputboxInput.focus();

        if (inputboxInput.tagName === 'TEXTAREA') {
            autoGrowTextarea(inputboxInput);
        } else if (typeof inputboxInput.select === 'function') {
            inputboxInput.select();
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

        inputboxHistory.innerHTML = '';
        const history = Array.isArray(p.history) ? p.history : [];
        if (history.length > 0) {
            const placeholderOption = document.createElement('option');
            placeholderOption.value = '';
            placeholderOption.textContent = 'History...';
            placeholderOption.disabled = true;
            placeholderOption.selected = true;
            inputboxHistory.appendChild(placeholderOption);

            history.forEach((item) => {
                const option = document.createElement('option');
                option.value = item;
                option.textContent = item;
                inputboxHistory.appendChild(option);
            });

            inputboxHistory.style.display = 'block';
        } else {
            inputboxHistory.style.display = 'none';
        }

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
