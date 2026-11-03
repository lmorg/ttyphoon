export function disableNativeTextAssistance(element) {
    for (const [attribute, value] of [
        ['autocomplete', 'off'],
        ['autocorrect', 'off'],
        ['autocapitalize', 'off'],
        ['spellcheck', 'false'],
        ['writingsuggestions', 'false'],
        ['data-gramm', 'false'],
        ['data-gramm_editor', 'false'],
        ['data-enable-grammarly', 'false'],
    ]) element.setAttribute(attribute, value);
}

export function disableNativeTextAssistanceIn(container) {
    container.querySelectorAll('textarea, input:not([type]), input[type="text"], input[type="search"], input[type="url"], input[type="email"], input[type="password"], [contenteditable="true"]')
        .forEach(disableNativeTextAssistance);
}