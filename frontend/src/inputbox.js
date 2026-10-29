import { GetClipboardData, TerminalInputBoxHistoryImages, TerminalInputBoxSubmit } from '../wailsjs/go/main/WApp';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { showLocalMenu } from './popup_menu';
import { initLineNavigationKeys } from './line-navigation.js';
import { attachVimMode } from './vim-mode';
import { attachSpellCheck } from './spellcheck';
import { disableNativeTextAssistance, disableNativeTextAssistanceIn } from './text-input.js';
import { bindSharedTooltipMouseTracking, closeSharedTooltip, showSharedTooltip } from './shared_tooltip';
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
                <div class="inputbox-title markdown-body" id="inputbox-title"><h3></h3></div>
                <div id="inputbox-input-container"></div>
                <div id="inputbox-images" class="inputbox-images" hidden></div>
                <div id="inputbox-image-error" class="inputbox-image-error" role="alert" hidden></div>
                <div class="inputbox-hint" id="inputbox-hint">
                    <span id="inputbox-confirm-hint">Return to confirm</span>
                    <span id="inputbox-vim-hint">Escape for Vim keys</span>
                    <span id="inputbox-escape-hint">Escape again to close</span>
                </div>
                <div class="inputbox-buttons">
                    <button class="inputbox-btn inputbox-ok" id="inputbox-ok">OK</button>
                    <button class="inputbox-btn inputbox-draft" id="inputbox-draft" data-tooltip="Close, but store draft" aria-label="Draft" hidden>&#xe698;</button>
                    <button class="inputbox-btn inputbox-history-btn" id="inputbox-history-btn" data-tooltip="History" aria-label="History" style="display:none">&#xf1da;</button>
                    <button class="inputbox-btn inputbox-cancel" id="inputbox-cancel">Cancel</button>
                </div>
            </div>
        </div>
    `;
    root.appendChild(wrapper.firstElementChild);
}

export function initInputBox(canvas) {
    ensureInputBoxDom();
    bindSharedTooltipMouseTracking();

    window.ttyphoonInputboxOpen = false;

    const inputboxOverlay = document.getElementById('terminal-inputbox');
    const inputboxInputContainer = document.getElementById('inputbox-input-container');
    const inputboxHistoryBtn = document.getElementById('inputbox-history-btn');
    const inputboxTitle = document.getElementById('inputbox-title');
    const inputboxConfirmHint = document.getElementById('inputbox-confirm-hint');
    const inputboxVimHint = document.getElementById('inputbox-vim-hint');
    const inputboxEscapeHint = document.getElementById('inputbox-escape-hint');
    const inputboxOkBtn = document.getElementById('inputbox-ok');
    const inputboxCancel = document.getElementById('inputbox-cancel');
    const inputboxDraftBtn = document.getElementById('inputbox-draft');
    const inputboxImagesContainer = document.getElementById('inputbox-images');
    const inputboxImageError = document.getElementById('inputbox-image-error');

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
    let inputboxSpellCheckHandle = null;
    let inputboxConfirmation = false;
    let inputboxCanSaveDraft = false;
    let inputboxSavingDraft = false;
    let inputboxAllowImagePaste = false;
    let inputboxImages = [];
    let inputboxPendingImages = 0;
    let inputboxRestoringHistoryImages = false;
    let inputboxGeneration = 0;
    const maxImages = 6;
    const maxImageBytes = 8 * 1024 * 1024;
    const imageTypes = new Set(['image/png', 'image/jpeg', 'image/gif', 'image/webp']);

    function showInputboxTooltip(event) {
        if (!isInputboxVisible()) return;
        const target = event.target instanceof Element ? event.target.closest('[data-tooltip]') : null;
        if (target && inputboxOverlay.contains(target)) {
            void showSharedTooltip(target.dataset.tooltip);
        }
    }

    function hideInputboxTooltip(event) {
        if (!isInputboxVisible()) return;
        const target = event.target instanceof Element ? event.target.closest('[data-tooltip]') : null;
        if (target && !(event.relatedTarget instanceof Node && target.contains(event.relatedTarget))) {
            closeSharedTooltip();
        }
    }

    inputboxOverlay.addEventListener('mouseover', showInputboxTooltip);
    inputboxOverlay.addEventListener('focusin', showInputboxTooltip);
    inputboxOverlay.addEventListener('mouseout', hideInputboxTooltip);
    inputboxOverlay.addEventListener('focusout', hideInputboxTooltip);
    inputboxOverlay.addEventListener('mouseleave', () => { if (isInputboxVisible()) closeSharedTooltip(); });
    inputboxOverlay.addEventListener('click', () => { if (isInputboxVisible()) closeSharedTooltip(); });

    function setImageError(message = '') {
        inputboxImageError.textContent = message;
        inputboxImageError.hidden = !message;
    }

    function renderInputboxImages() {
        inputboxImagesContainer.replaceChildren();
        inputboxImagesContainer.hidden = inputboxImages.length === 0 && inputboxPendingImages === 0;
        inputboxImagesContainer.setAttribute('aria-busy', String(inputboxPendingImages > 0));
        inputboxOkBtn.disabled = inputboxPendingImages > 0 || inputboxSavingDraft;
        inputboxDraftBtn.disabled = inputboxPendingImages > 0 || inputboxSavingDraft;
        inputboxCancel.disabled = inputboxSavingDraft;
        inputboxImages.forEach((attachment, index) => {
            const item = document.createElement('div');
            item.className = 'inputbox-image';
            const image = document.createElement('img');
            image.src = `data:${attachment.mimeType};base64,${attachment.base64}`;
            image.alt = `Attached image ${index + 1}`;
            const remove = document.createElement('button');
            remove.type = 'button';
            remove.className = 'inputbox-image-remove';
            remove.innerHTML = '&#xf00d;';
            remove.dataset.tooltip = `Remove image ${index + 1}`;
            remove.setAttribute('aria-label', remove.dataset.tooltip);
            remove.addEventListener('click', () => {
                inputboxImages.splice(index, 1);
                setImageError();
                renderInputboxImages();
                inputboxInput?.focus();
            });
            remove.addEventListener('keydown', (event) => event.stopPropagation());
            item.append(image, remove);
            inputboxImagesContainer.appendChild(item);
        });
        if (inputboxPendingImages > 0) {
            const loading = document.createElement('span');
            loading.className = 'inputbox-image-loading';
            loading.innerHTML = '&#xf110;';
            loading.setAttribute('role', 'status');
            loading.setAttribute('aria-label', 'Loading image');
            inputboxImagesContainer.appendChild(loading);
        }
    }

    function readImageFile(file) {
        if (!imageTypes.has(file.type)) {
            return Promise.reject(new Error('Use a PNG, JPEG, GIF, or WebP image.'));
        }
        if (file.size > maxImageBytes) {
            return Promise.reject(new Error('Each image must be at most 8 MiB.'));
        }
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onload = () => resolve({ mimeType: file.type, base64: String(reader.result).split(',')[1] });
            reader.onerror = () => reject(new Error('Could not read the pasted image.'));
            reader.onabort = () => reject(new Error('Image read was cancelled.'));
            reader.readAsDataURL(file);
        });
    }

    async function pasteInputboxImage(readImage, target = inputboxInput) {
        if (inputboxRestoringHistoryImages) {
            setImageError('Wait for saved images to finish loading.');
            return;
        }
        if (inputboxImages.length + inputboxPendingImages >= maxImages) {
            setImageError('Attach at most 6 images.');
            return;
        }
        const generation = inputboxGeneration;
        inputboxPendingImages++;
        setImageError();
        renderInputboxImages();
        try {
            const result = await readImage();
            if (generation !== inputboxGeneration || !isInputboxVisible()) return;
            if (result?.text != null && !result.image && !result.base64) {
                if (target.selectionStart != null) {
                    target.setRangeText(result.text, target.selectionStart, target.selectionEnd, 'end');
                } else {
                    target.value = result.text;
                }
                target.dispatchEvent(new Event('input', { bubbles: true }));
                return;
            }
            const attachment = result?.image ? { mimeType: 'image/png', base64: result.image } : result;
            if (!attachment?.base64) return;
            if (!imageTypes.has(attachment.mimeType) || attachment.base64.length > Math.ceil(maxImageBytes / 3) * 4) {
                throw new Error('Use a PNG, JPEG, GIF, or WebP image of at most 8 MiB.');
            }
            inputboxImages.push(attachment);
        } catch (error) {
            if (generation === inputboxGeneration) setImageError(error.message || 'Could not paste image.');
        } finally {
            if (generation === inputboxGeneration) {
                inputboxPendingImages--;
                renderInputboxImages();
            }
        }
    }

    inputboxOverlay.addEventListener('paste', (event) => {
        if (!isInputboxVisible() || !inputboxAllowImagePaste) return;
        const items = Array.from(event.clipboardData?.items || []);
        const files = items.filter((item) => item.kind === 'file' && item.type.startsWith('image/'))
            .map((item) => item.getAsFile()).filter(Boolean);
        if (files.length > 0) {
            event.preventDefault();
            event.stopPropagation();
            for (const file of files) void pasteInputboxImage(() => readImageFile(file));
        } else if (items.length === 0 && !event.clipboardData?.getData('text/plain')) {
            event.preventDefault();
            event.stopPropagation();
            const target = event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement
                ? event.target : inputboxInput;
            void pasteInputboxImage(() => GetClipboardData(), target);
        }
    });

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

    async function restoreHistoryImages(index, entry) {
        const generation = ++inputboxGeneration;
        inputboxRestoringHistoryImages = false;
        inputboxImages = [];
        inputboxPendingImages = 0;
        setImageError();
        renderInputboxImages();
        if (!inputboxAllowImagePaste || !Array.isArray(entry.images) || entry.images.length === 0) return;
        inputboxRestoringHistoryImages = true;
        inputboxPendingImages = 1;
        renderInputboxImages();
        try {
            const images = await TerminalInputBoxHistoryImages(inputboxId, index);
            if (generation !== inputboxGeneration || !isInputboxVisible()) return;
            if (!Array.isArray(images) || images.length > maxImages || images.some((image) =>
                !imageTypes.has(image.mimeType) || !image.base64 || image.base64.length > Math.ceil(maxImageBytes / 3) * 4)) {
                throw new Error('Saved images are invalid or exceed the attachment limits.');
            }
            inputboxImages = images;
        } catch (error) {
            if (generation === inputboxGeneration) {
                setImageError(`Could not restore saved images: ${error.message || error}`);
            }
        } finally {
            if (generation === inputboxGeneration) {
                inputboxRestoringHistoryImages = false;
                inputboxPendingImages = 0;
                renderInputboxImages();
            }
        }
    }

    function openInputboxHistoryMenu(x, y) {
        if (!inputboxInput || inputboxHistoryItems.length === 0) {
            return;
        }

        const generation = inputboxGeneration;
        showLocalMenu({
            title: 'History',
            options: inputboxHistoryItems.map((entry) => {
                const variables = Object.entries(entry.variables || {})
                    .map(([name, value]) => `${name}: ${value === '' ? '""' : String(value)}`).join(', ');
                const text = entry.value || '(empty)';
                const label = variables ? `${text} | ${variables}` : text;
                return entry.images?.length ? `${label} | ${entry.images.length} image(s)` : label;
            }),
            x,
            y,
            showNextToMouseCursor: true,
            onSelect: (index) => {
                const entry = inputboxHistoryItems[index];
                if (!entry || !inputboxInput || generation !== inputboxGeneration || !isInputboxVisible()) {
                    return;
                }

                inputboxInput.value = entry.value;
                void restoreHistoryImages(index, entry);
                if (entry.variables) {
                    for (const field of inputboxVariableGetters) {
                        if (Object.prototype.hasOwnProperty.call(entry.variables, field.name)) {
                            field.setter(entry.variables[field.name]);
                        } else {
                            field.reset();
                        }
                    }
                }
                inputboxInput.dispatchEvent(new Event('input', { bubbles: true }));
                inputboxInput.focus();

                if (inputboxInput.tagName === 'TEXTAREA') {
                    autoGrowTextarea(inputboxInput);
                } else if (typeof inputboxInput.select === 'function') {
                    inputboxInput.select();
                }
            },
        });
    }

    function openFieldHistoryMenu(control, name = null, setter = (value) => { control.value = value; }) {
        const values = [];
        const seen = new Set();
        for (const entry of inputboxHistoryItems) {
            if (name !== null && !Object.prototype.hasOwnProperty.call(entry.variables || {}, name)) continue;
            const value = String(name === null ? entry.value : entry.variables[name]);
            if (!seen.has(value)) {
                seen.add(value);
                values.push(value);
            }
        }
        if (values.length === 0) return false;
        const rect = control.getBoundingClientRect();
        showLocalMenu({
            title: 'History', options: values.map((value) => value || '(empty)'),
            x: rect.left, y: rect.bottom, showNextToMouseCursor: true,
            onSelect: (index) => {
                if (index < 0 || index >= values.length) return;
                setter(values[index]);
                control.dispatchEvent(new Event('input', { bubbles: true }));
                control.focus();
                if (control.tagName === 'TEXTAREA') autoGrowTextarea(control);
                else control.select?.();
            },
        });
        return true;
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

        if (shouldOpenHistoryUpArrow(e) && openFieldHistoryMenu(inputboxInput)) {
            e.preventDefault();
            return true;
        }
        if (shouldOpenHistoryHotkey(e)) {
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
        return inputboxId !== null && inputboxOverlay.isConnected && inputboxOverlay.style.display !== 'none';
    }

    function getInputboxFocusableElements() {
        if (inputboxConfirmation) {
            return [inputboxOkBtn, inputboxCancel].filter((button) => !button.disabled);
        }
        const selectors = [
            'input:not([type="hidden"])',
            'textarea',
            'select',
            'button.inputbox-image-remove',
            '[tabindex]:not([tabindex="-1"])',
        ];

        return Array.from(inputboxOverlay.querySelectorAll(selectors.join(', ')))
            .filter((el) => {
                // Exclude buttons (they have hotkeys, not part of tab cycle)
                if (el.tagName === 'BUTTON' && !el.classList.contains('inputbox-image-remove')) {
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
        if (inputboxConfirmation) {
            inputboxCancel.focus();
            return;
        }
        const focusables = getInputboxFocusableElements();
        if (focusables.length === 0) {
            return;
        }

        focusables[0].focus();
    }

    function addVariableField(definition, draftVariables = {}) {
        if (!definition || !definition.name) {
            return;
        }

        const fieldName = String(definition.name);
        const initialValue = Object.prototype.hasOwnProperty.call(draftVariables, fieldName)
            ? draftVariables[fieldName] : definition.default;
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
            label.dataset.tooltip = tooltip;
        }
        line.appendChild(label);

        let control = null;
        let getter = () => '';
        let setter = () => {};

        if (fieldType === 'boolean') {
            const checkboxWrap = document.createElement('label');
            checkboxWrap.className = 'inputbox-variable-checkbox-wrap';
            control = document.createElement('input');
            control.type = 'checkbox';
            control.className = 'inputbox-variable-checkbox';
            control.checked = parseBooleanDefault(initialValue);
            checkboxWrap.appendChild(control);
            line.appendChild(checkboxWrap);

            getter = () => Boolean(control.checked);
            setter = (value) => { control.checked = parseBooleanDefault(value); };
        } else if (fieldType === 'integer') {
            control = document.createElement('input');
            control.type = 'number';
            control.step = '1';
            control.className = 'inputbox-input';
            const defaultValue = String(initialValue ?? '').trim();
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
            setter = (value) => { control.value = String(value ?? ''); };
        } else if (fieldType === 'list') {
            let options = parseDefinitionOptions(definition);
            if (options.length === 0) {
                // Backward compatibility for existing definitions that still pass
                // comma-separated choices via default.
                options = parseListOptions(definition.default);
            }

            let selected = options[0];
            const defaultValue = String(initialValue ?? '').trim();
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
            setter = (value) => {
                selected = options.includes(String(value)) ? String(value) : options[0];
                control.textContent = selected;
            };
        } else {
            control = document.createElement('input');
            control.type = 'text';
            control.className = 'inputbox-input';
            control.value = String(initialValue ?? '');
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
            setter = (value) => { control.value = String(value ?? ''); };
        }

        disableNativeTextAssistanceIn(line);
        if (control && typeof control.addEventListener === 'function') {
            control.addEventListener('keydown', (e) => {
                if (fieldType === 'string' && e.key === 'ArrowUp' && control.value.length === 0 &&
                    openFieldHistoryMenu(control, fieldName, setter)) {
                    e.preventDefault();
                    e.stopPropagation();
                    return;
                }
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
        inputboxVariableGetters.push({ name: fieldName, getter, setter, reset: () => setter(definition.default) });
    }

    async function inputboxSubmit(isOk, saveDraft = false) {
        if (inputboxId === null || inputboxSavingDraft || ((isOk || saveDraft) && inputboxPendingImages > 0)) {
            return;
        }
        if (saveDraft && !inputboxCanSaveDraft) {
            return;
        }

        const value = inputboxInput?.value ?? '';
        const variables = {};
        for (const field of inputboxVariableGetters) {
            if (!field || !field.name || typeof field.getter !== 'function') {
                continue;
            }
            variables[field.name] = field.getter();
        }

        const id = inputboxId;
        const images = (isOk || saveDraft) && inputboxAllowImagePaste ? inputboxImages : [];
        const result = { value, variables };
        if (images.length > 0) result.images = images;
        if (saveDraft) {
            result.saveDraft = true;
            const generation = inputboxGeneration;
            inputboxSavingDraft = true;
            renderInputboxImages();
            try {
                await TerminalInputBoxSubmit(id, result, false);
            } catch {
                if (generation === inputboxGeneration) {
                    inputboxSavingDraft = false;
                    renderInputboxImages();
                    setImageError('Could not save draft. Please try again.');
                }
                return;
            }
            if (generation !== inputboxGeneration) return;
        }
        inputboxSavingDraft = false;
        inputboxGeneration++;
        inputboxId = null;
        inputboxVariableGetters = [];
        inputboxImages = [];
        inputboxPendingImages = 0;
        inputboxRestoringHistoryImages = false;
        renderInputboxImages();
        setImageError();

        inputboxVimHandle?.detach();
        inputboxVimHandle = null;
        inputboxSpellCheckHandle?.detach();
        inputboxSpellCheckHandle = null;

        inputboxOverlay.style.display = 'none';
        closeSharedTooltip();
        window.ttyphoonInputboxOpen = false;
        setSharedTerminalFocusState(inputboxPreviousTerminalFocusedState, {
            focusVisible: false,
            force: true,
        });

        if (!saveDraft) TerminalInputBoxSubmit(id, result, isOk).catch(() => {});
    }

    inputboxDraftBtn.addEventListener('click', () => inputboxSubmit(false, true));
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

    function handleConfirmationKeyboard(e) {
        if (!isInputboxVisible() || !inputboxConfirmation) {
            return;
        }
        e.preventDefault();
        e.stopImmediatePropagation();
        if (e.type !== 'keydown') {
            return;
        }
        if (e.key === 'Escape') {
            inputboxSubmit(false);
        } else if (e.key === 'Tab' && !e.ctrlKey && !e.metaKey && !e.altKey) {
            cycleInputboxFocus(e.shiftKey);
        } else if (e.key === 'Enter' || e.key === ' ' || e.code === 'Space') {
            if (document.activeElement === inputboxOkBtn || document.activeElement === inputboxCancel) {
                document.activeElement.click();
            }
        }
    }
    for (const eventName of ['keydown', 'keypress', 'keyup']) {
        window.addEventListener(eventName, handleConfirmationKeyboard, true);
    }

    document.addEventListener('keydown', (e) => {
        if (!isInputboxVisible()) {
            return;
        }

        if (e.key !== 'Tab') {
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
        closeSharedTooltip();
        backdropPointerDown = false;
        window.ttyphoonInputboxOpen = true;
        inputboxPreviousTerminalFocusedState = window.terminalFocusedState === true;
        setSharedTerminalFocusState(false, { focusVisible: false, force: true });
        inputboxTitle.querySelector('h3').textContent = p.title ?? '';
        inputboxConfirmation = p.confirmation === true;
        inputboxCanSaveDraft = p.canSaveDraft === true && !inputboxConfirmation;
        inputboxSavingDraft = false;
        inputboxDraftBtn.hidden = !inputboxCanSaveDraft;
        inputboxDraftBtn.style.display = inputboxCanSaveDraft ? '' : 'none';
        const draft = inputboxCanSaveDraft ? p.draft : null;
        const initialValue = draft?.value ?? p.defaultValue ?? '';
        inputboxGeneration++;
        inputboxAllowImagePaste = p.allowImagePaste === true && !inputboxConfirmation;
        inputboxImages = inputboxAllowImagePaste && Array.isArray(draft?.images) ? draft.images.slice(0, maxImages) : [];
        inputboxPendingImages = 0;
        inputboxRestoringHistoryImages = false;
        renderInputboxImages();
        setImageError();
        inputboxConfirmHint.hidden = inputboxConfirmation;
        inputboxConfirmHint.textContent = p.multiline
            ? 'Ctrl+Return to confirm' : 'Return to confirm';
        inputboxVimHint.hidden = inputboxConfirmation;
        inputboxEscapeHint.textContent = inputboxConfirmation ? 'Escape to cancel' : 'Escape again to close';
        inputboxEscapeHint.hidden = false;
        inputboxOkBtn.textContent = p.confirmLabel || 'OK';
        inputboxCancel.textContent = p.cancelLabel || 'Cancel';
        inputboxOkBtn.classList.toggle('inputbox-danger', p.confirmIsDanger === true);
        inputboxInputContainer.innerHTML = '';
        inputboxVariableGetters = [];

        if (inputboxConfirmation) {
            inputboxInput = null;
            inputboxHistoryItems = [];
            inputboxHistoryBtn.style.display = 'none';
            const message = document.createElement('div');
            message.className = 'inputbox-confirmation-message';
            message.textContent = p.message ?? '';
            inputboxInputContainer.appendChild(message);
            inputboxOverlay.style.display = 'flex';
            setTimeout(() => inputboxCancel.focus(), 0);
            return;
        }

        inputboxVimHint.hidden = false;

        if (p.multiline) {
            inputboxInput = document.createElement('textarea');
            inputboxInput.className = 'inputbox-input';
            inputboxInput.rows = 2;
            inputboxInput.value = initialValue;
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
            inputboxInput.value = initialValue;
            inputboxInput.placeholder = p.placeholder ?? '';
            inputboxInput.setAttribute('autocomplete', 'off');
            inputboxInput.setAttribute('autocorrect', 'off');
            inputboxInput.setAttribute('autocapitalize', 'off');
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

        const history = Array.isArray(p.historyEntries) ? p.historyEntries : (Array.isArray(p.history) ? p.history : []);
        inputboxHistoryItems = history.map((entry) => typeof entry === 'string' ? { value: entry } : entry)
            .filter((entry) => entry && typeof entry.value === 'string');
        inputboxHistoryBtn.style.display = inputboxHistoryItems.length > 0 ? 'inline-flex' : 'none';

        disableNativeTextAssistance(inputboxInput);
        inputboxInputContainer.appendChild(inputboxInput);
        if (p.multiline) {
            inputboxSpellCheckHandle = attachSpellCheck(inputboxInput);
        }

        const variables = Array.isArray(p.variables) ? p.variables : [];
        for (const variable of variables) {
            addVariableField(variable, draft?.variables || {});
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
