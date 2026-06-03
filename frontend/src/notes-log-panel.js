function logLevelColour(levelToken) {
    const normalized = levelToken
        .replace(/^\[/, '')
        .replace(/\]$/, '')
        .replace(/:$/, '')
        .trim()
        .toLowerCase();

    switch (normalized) {
        case 'error':
            return 'var(--red)';
        case 'warn':
            return 'var(--yellow)';
        case 'info':
            return 'var(--cyan)';
        case 'debug':
            return 'var(--green)';
        default:
            return 'var(--magenta)';
    }
}

function appendTextWithTokenHighlight(parentEl, text, baseColour) {
    // Highlight key/value prefixes (key=) and booleans while preserving base colour for other text.
    const tokenPattern = /([-_.a-zA-Z0-9]+=)|\b(true|false)\b/gi;
    let lastIndex = 0;
    let match = tokenPattern.exec(text);

    while (match) {
        if (match.index > lastIndex) {
            const plainSpan = document.createElement('span');
            plainSpan.style.color = baseColour;
            plainSpan.textContent = text.slice(lastIndex, match.index);
            parentEl.appendChild(plainSpan);
        }

        const tokenSpan = document.createElement('span');
        if (match[1]) {
            tokenSpan.style.color = 'var(--yellow)';
        } else {
            tokenSpan.style.color = 'var(--magenta)';
        }
        tokenSpan.textContent = match[0];
        parentEl.appendChild(tokenSpan);

        lastIndex = match.index + match[0].length;
        match = tokenPattern.exec(text);
    }

    if (lastIndex < text.length) {
        const trailingSpan = document.createElement('span');
        trailingSpan.style.color = baseColour;
        trailingSpan.textContent = text.slice(lastIndex);
        parentEl.appendChild(trailingSpan);
    }
}

function appendTextWithUrlHighlight(parentEl, text, baseColour) {
    // Match http(s) URLs or common www.* URLs, stopping at whitespace/quotes/backticks.
    const urlPattern = /(https?:\/\/[^\s"'`<>]+|www\.[^\s"'`<>]+)/g;
    let lastIndex = 0;
    let match = urlPattern.exec(text);

    while (match) {
        if (match.index > lastIndex) {
            appendTextWithTokenHighlight(parentEl, text.slice(lastIndex, match.index), baseColour);
        }

        const urlSpan = document.createElement('span');
        urlSpan.style.color = 'var(--link)';
        urlSpan.textContent = match[0];
        parentEl.appendChild(urlSpan);

        lastIndex = match.index + match[0].length;
        match = urlPattern.exec(text);
    }

    if (lastIndex < text.length) {
        appendTextWithTokenHighlight(parentEl, text.slice(lastIndex), baseColour);
    }
}

function appendHighlightedMessage(parentEl, text, defaultColour) {
    const prefixMatch = text.match(/^([-_. a-zA-Z]+: )/);
    if (prefixMatch) {
        const prefixSpan = document.createElement('span');
        prefixSpan.style.color = 'var(--magenta)';
        prefixSpan.textContent = prefixMatch[1];
        parentEl.appendChild(prefixSpan);

        const remainder = text.slice(prefixMatch[1].length);
        appendHighlightedMessage(parentEl, remainder, defaultColour);
        return;
    }

    const pattern = /(`[^`]*`|"(?:\\.|[^"\\])*")/g;
    let lastIndex = 0;
    let match = pattern.exec(text);

    while (match) {
        if (match.index > lastIndex) {
            appendTextWithUrlHighlight(parentEl, text.slice(lastIndex, match.index), defaultColour);
        }

        appendTextWithUrlHighlight(parentEl, match[0], 'var(--cyan)');

        lastIndex = match.index + match[0].length;
        match = pattern.exec(text);
    }

    if (lastIndex < text.length) {
        appendTextWithUrlHighlight(parentEl, text.slice(lastIndex), defaultColour);
    }
}

function appendStyledTimestamp(parentEl, ts) {
    const parts = String(ts).split(/([/:])/g);
    for (const part of parts) {
        if (part === '') {
            continue;
        }

        const span = document.createElement('span');
        if (part === '/' || part === ':') {
            span.className = 'notes-log-ts-punct';
        } else {
            span.className = 'notes-log-ts-text';
        }
        span.textContent = part;
        parentEl.appendChild(span);
    }
}

function appendColourisedLogLine(logOutput, line) {
    const lineEl = document.createElement('div');
    lineEl.className = 'notes-log-line';

    const tsMatch = line.match(/^([0-9]{4}\/[0-9]{2}\/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2})(\s*)(.*)$/);
    if (!tsMatch) {
        lineEl.textContent = line;
        logOutput.appendChild(lineEl);
        return;
    }

    const ts = tsMatch[1];
    const remainder = tsMatch[3];

    // Keep timestamp available on hover but remove it from visible output.
    lineEl.title = ts;

    const tsSpan = document.createElement('span');
    tsSpan.className = 'notes-log-timestamp';
    appendStyledTimestamp(tsSpan, ts);
    tsSpan.appendChild(document.createTextNode(' '));
    lineEl.appendChild(tsSpan);

    const levelMatch = remainder.match(/^(\[[\-_ a-zA-Z0-9]+\]|[A-Za-z0-9_ -]+:)(\s*)(.*)$/);
    if (levelMatch) {
        const levelToken = levelMatch[1];
        const spaceAfterLevel = levelMatch[2];
        const messageText = levelMatch[3];

        const levelSpan = document.createElement('span');
        levelSpan.style.color = logLevelColour(levelToken);
        levelSpan.textContent = levelToken;
        lineEl.appendChild(levelSpan);
        lineEl.appendChild(document.createTextNode(spaceAfterLevel));

        const msgSpan = document.createElement('span');
        appendHighlightedMessage(msgSpan, messageText, 'var(--fg)');
        lineEl.appendChild(msgSpan);
    } else {
        const msgSpan = document.createElement('span');
        appendHighlightedMessage(msgSpan, remainder, 'var(--fg)');
        lineEl.appendChild(msgSpan);
    }

    logOutput.appendChild(lineEl);
}

function appendColourisedLogMessage(logOutput, message) {
    const lines = String(message ?? '').split(/\r?\n/);
    for (let i = 0; i < lines.length; i++) {
        if (i === lines.length - 1 && lines[i] === '') {
            continue;
        }
        appendColourisedLogLine(logOutput, lines[i]);
    }
    logOutput.scrollTop = logOutput.scrollHeight;
}

export function initNotesLogPanel(elements, eventsOn) {
    if (!elements?.logOutput) {
        return;
    }

    if (elements.toolsLogClear) {
        elements.toolsLogClear.addEventListener('click', () => {
            elements.logOutput.textContent = '';
        });
    }

    if (elements.toolsLogWordwrap) {
        let isWordWrapped = false;

        const updateWrapButtonState = () => {
            elements.toolsLogWordwrap.dataset.wrapped = isWordWrapped ? 'true' : 'false';
        };

        elements.toolsLogWordwrap.addEventListener('click', () => {
            isWordWrapped = !isWordWrapped;
            if (isWordWrapped) {
                elements.logOutput.style.whiteSpace = 'pre-wrap';
                elements.logOutput.style.overflowWrap = 'break-word';
            } else {
                elements.logOutput.style.whiteSpace = 'pre';
                elements.logOutput.style.overflowWrap = 'normal';
            }

            updateWrapButtonState();
        });

        // No-wrap by default
        updateWrapButtonState();
    }

    if (elements.toolsLogTimestamp && elements.logOutput) {
        let isTimestampVisible = false;

        const updateTimestampButtonState = () => {
            elements.toolsLogTimestamp.dataset.enabled = isTimestampVisible ? 'true' : 'false';
            elements.logOutput.dataset.showTimestamp = isTimestampVisible ? 'true' : 'false';
        };

        elements.toolsLogTimestamp.addEventListener('click', () => {
            isTimestampVisible = !isTimestampVisible;
            updateTimestampButtonState();
        });

        // Timestamps are hidden by default.
        updateTimestampButtonState();
    }

    eventsOn('notesLog', (message) => {
        appendColourisedLogMessage(elements.logOutput, message);
    });
}
