import { TerminalInputBoxSubmit } from '../wailsjs/go/main/WApp';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { showLocalMenu } from './popup_menu';
import { initLineNavigationKeys } from './line-navigation.js';
import { attachVimMode } from './vim-mode';
import './inputbox.css';

initLineNavigationKeys(document);

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
                    <span>Escape for Vim keys</span>
                    <span>Escape again to close</span>
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

    window.ttyphoonInputboxOpen = false;

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
    let inputboxVariableGetters = [];
    let inputboxPreviousTerminalFocusedState = true;
    let backdropPointerDown = false;
    let inputboxVimHandle = null;

    function setSharedTerminalFocusState(focused, options = {}) {
        if (typeof window.ttyphoonSetTerminalFocusState === 'function') {
            window.ttyphoonSetTerminalFocusState(Boolean(focused), options);
            return;
        }

        window.terminalFocusedState = Boolean(focused);
    }

    function normalizeVariableType(rawType) {
        const value = String(rawType || '').trim().toLowerCase();
        switch (value) {
            case 'string':
            case 'str':
                return 'string';
            case 'int':
            case 'integer':
                return 'integer';
            case 'bool':
            case 'boolean':
                return 'boolean';
            case 'list':
                return 'list';
            default:
                return 'string';
        }
    }

    function parseBooleanDefault(value) {
        const normalized = String(value || '').trim().toLowerCase();
        return normalized === 'true' || normalized === '1' || normalized === 'yes' || normalized === 'on';
    }

    function parseListOptions(value) {
        const options = String(value || '')
            .split(',')
            .map((item) => item.trim())
            .filter((item) => item.length > 0);

        if (options.length === 0) {
            return [''];
        }

        return options;
    }

    function parseDefinitionOptions(definition) {
        if (!definition || !Array.isArray(definition.options)) {
            return [];
        }

        return definition.options
            .map((item) => String(item ?? '').trim())
            .filter((item) => item.length > 0);
    }

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

    function isInputboxVisible() {
        return inputboxId !== null && inputboxOverlay.style.display !== 'none';
    }

    function getInputboxFocusableElements() {
        const selectors = [
            'input:not([type="hidden"])',
            'textarea',
            'select',
            '[tabindex]:not([tabindex="-1"])',
        ];

        return Array.from(inputboxOverlay.querySelectorAll(selectors.join(', ')))
            .filter((el) => {
                // Exclude buttons (they have hotkeys, not part of tab cycle)
                if (el.tagName === 'BUTTON') {
                    return false;
                }
                if (el.disabled || el.getAttribute('aria-hidden') === 'true') {
                    return false;
                }
                if (el.hidden) {
                    return false;
                }
                if (el.style && el.style.display === 'none') {
                    return false;
                }
                return true;
            });
    }

    function cycleInputboxFocus(reverse) {
        const focusables = getInputboxFocusableElements();
        if (focusables.length === 0) {
            return;
        }

        const active = document.activeElement;
        const currentIndex = focusables.indexOf(active);
        if (currentIndex === -1) {
            const fallback = reverse ? focusables[focusables.length - 1] : focusables[0];
            fallback.focus();
            return;
        }

        const nextIndex = reverse
            ? (currentIndex - 1 + focusables.length) % focusables.length
            : (currentIndex + 1) % focusables.length;
        focusables[nextIndex].focus();
    }

    function focusFirstInputboxElement() {
        const focusables = getInputboxFocusableElements();
        if (focusables.length === 0) {
            return;
        }

        focusables[0].focus();
    }

    function addVariableField(definition) {
        if (!definition || !definition.name) {
            return;
        }

        const fieldName = String(definition.name);
        const fieldType = normalizeVariableType(definition.type);
        const row = document.createElement('div');
        row.className = 'inputbox-variable-row';

        const line = document.createElement('div');
        line.className = 'inputbox-variable-line';
        row.appendChild(line);

        const label = document.createElement('label');
        label.className = 'inputbox-variable-label';
        label.textContent = String(definition.label || fieldName);
        if (definition.description) {
            const tooltip = String(definition.description);
            label.setAttribute('aria-label', `${label.textContent}: ${tooltip}`);

            const tooltipEl = document.createElement('span');
            tooltipEl.className = 'inputbox-variable-tooltip';
            tooltipEl.textContent = tooltip;
            tooltipEl.setAttribute('role', 'tooltip');
            label.appendChild(tooltipEl);
        }
        line.appendChild(label);

        let control = null;
        let getter = () => '';

        if (fieldType === 'boolean') {
            const checkboxWrap = document.createElement('label');
            checkboxWrap.className = 'inputbox-variable-checkbox-wrap';
            control = document.createElement('input');
            control.type = 'checkbox';
            control.className = 'inputbox-variable-checkbox';
            control.checked = parseBooleanDefault(definition.default);
            checkboxWrap.appendChild(control);
            line.appendChild(checkboxWrap);

            getter = () => Boolean(control.checked);
        } else if (fieldType === 'integer') {
            control = document.createElement('input');
            control.type = 'number';
            control.step = '1';
            control.className = 'inputbox-input';
            const defaultValue = String(definition.default ?? '').trim();
            control.value = defaultValue;
            const placeholderOptions = parseDefinitionOptions(definition);
            if (placeholderOptions.length > 0) {
                control.placeholder = placeholderOptions[0];
            }
            line.appendChild(control);

            getter = () => {
                const parsed = Number.parseInt(String(control.value || '').trim(), 10);
                return Number.isFinite(parsed) ? parsed : 0;
            };
        } else if (fieldType === 'list') {
            let options = parseDefinitionOptions(definition);
            if (options.length === 0) {
                // Backward compatibility for existing definitions that still pass
                // comma-separated choices via default.
                options = parseListOptions(definition.default);
            }

            let selected = options[0];
            const defaultValue = String(definition.default ?? '').trim();
            if (defaultValue.length > 0 && options.includes(defaultValue)) {
                selected = defaultValue;
            }

            control = document.createElement('button');
            control.type = 'button';
            control.className = 'inputbox-input inputbox-variable-list-btn';
            control.textContent = selected;
            control.addEventListener('click', () => {
                const rect = control.getBoundingClientRect();
                showLocalMenu({
                    title: String(definition.label || fieldName),
                    options,
                    x: rect.left,
                    y: rect.bottom,
                    showNextToMouseCursor: true,
                    onSelect: (index) => {
                        const value = options[index];
                        if (typeof value !== 'string') {
                            return;
                        }
                        selected = value;
                        control.textContent = value;
                        control.focus();
                    },
                });
            });
            line.appendChild(control);

            getter = () => selected;
        } else {
            control = document.createElement('input');
            control.type = 'text';
            control.className = 'inputbox-input';
            control.value = String(definition.default ?? '');
            const placeholderOptions = parseDefinitionOptions(definition);
            if (placeholderOptions.length > 0) {
                control.placeholder = placeholderOptions[0];
            }
            control.setAttribute('autocomplete', 'off');
            control.setAttribute('autocorrect', 'off');
            control.setAttribute('autocapitalize', 'off');
            control.setAttribute('spellcheck', 'false');
            line.appendChild(control);

            getter = () => String(control.value || '');
        }

        if (control && typeof control.addEventListener === 'function') {
            control.addEventListener('keydown', (e) => {
                if (e.key === 'Escape') {
                    e.preventDefault();
                    inputboxSubmit(false);
                }

                if (e.key === 'Enter') {
                    const isListButton = control.classList.contains('inputbox-variable-list-btn');
                    if (!isListButton) {
                        e.preventDefault();
                        inputboxSubmit(true);
                    }
                }

                e.stopPropagation();
            });
        }

        inputboxInputContainer.appendChild(row);
        inputboxVariableGetters.push({ name: fieldName, getter });
    }

    function inputboxSubmit(isOk) {
        if (inputboxId === null || !inputboxInput) {
            return;
        }

        const value = inputboxInput.value;
        const variables = {};
        for (const field of inputboxVariableGetters) {
            if (!field || !field.name || typeof field.getter !== 'function') {
                continue;
            }
            variables[field.name] = field.getter();
        }

        const id = inputboxId;
        inputboxId = null;
        inputboxVariableGetters = [];

        inputboxVimHandle?.detach();
        inputboxVimHandle = null;

        inputboxOverlay.style.display = 'none';
        window.ttyphoonInputboxOpen = false;
        setSharedTerminalFocusState(inputboxPreviousTerminalFocusedState, {
            focusVisible: false,
            force: true,
        });

        TerminalInputBoxSubmit(id, { value, variables }, isOk).catch(() => {});
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

    document.addEventListener('keydown', (e) => {
        if (e.key !== 'Tab' || !isInputboxVisible()) {
            return;
        }

        e.preventDefault();
        e.stopPropagation();

        if (e.ctrlKey || e.metaKey || e.altKey) {
            return;
        }

        cycleInputboxFocus(e.shiftKey);
    }, true);

    document.addEventListener('focusin', (e) => {
        if (!isInputboxVisible()) {
            return;
        }

        const target = e.target;
        if (!(target instanceof Element)) {
            return;
        }

        if (inputboxOverlay.contains(target)) {
            return;
        }

        focusFirstInputboxElement();
    }, true);

    EventsOn('terminalInputBox', (payload) => {
        const p = Array.isArray(payload?.[0]) ? payload[0] : payload;
        if (!p) {
            return;
        }

        inputboxId = p.id;
        backdropPointerDown = false;
        window.ttyphoonInputboxOpen = true;
        inputboxPreviousTerminalFocusedState = window.terminalFocusedState === true;
        setSharedTerminalFocusState(false, { focusVisible: false, force: true });
        inputboxTitle.textContent = p.title ?? '';
        inputboxConfirmHint.textContent = p.multiline
            ? 'Ctrl+Return to confirm'
            : 'Return to confirm';
        inputboxInputContainer.innerHTML = '';
        inputboxVariableGetters = [];

        if (p.multiline) {
            inputboxInput = document.createElement('textarea');
            inputboxInput.className = 'inputbox-input';
            inputboxInput.rows = 2;
            inputboxInput.value = p.defaultValue ?? '';
            inputboxInput.placeholder = p.placeholder ?? '';
            inputboxInput.setAttribute('autocomplete', 'off');
            inputboxInput.setAttribute('autocorrect', 'off');
            inputboxInput.setAttribute('autocapitalize', 'off');
            inputboxInput.setAttribute('spellcheck', 'false');
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
                    // First Esc transitions vim INSERT→NORMAL; second Esc closes.
                    if (!inputboxVimHandle || inputboxVimHandle.getMode() === 'normal') {
                        e.preventDefault();
                        inputboxSubmit(false);
                    }
                    // else: vim mode (registered later on same element) handles it
                }
                e.stopPropagation();
            });
            inputboxVimHandle = attachVimMode(inputboxInput);
        } else {
            inputboxInput = document.createElement('input');
            inputboxInput.className = 'inputbox-input';
            inputboxInput.type = 'text';
            inputboxInput.value = p.defaultValue ?? '';
            inputboxInput.placeholder = p.placeholder ?? '';
            inputboxInput.setAttribute('autocomplete', 'off');
            inputboxInput.setAttribute('autocorrect', 'off');
            inputboxInput.setAttribute('autocapitalize', 'off');
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

        const variables = Array.isArray(p.variables) ? p.variables : [];
        for (const variable of variables) {
            addVariableField(variable);
        }

        inputboxOverlay.style.display = 'flex';
        setTimeout(() => {
            inputboxInput.focus();
            if (typeof inputboxInput.select === 'function') {
                inputboxInput.select();
            }
        }, 0);
    });
}
