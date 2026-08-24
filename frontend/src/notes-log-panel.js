function logLevelColour(levelToken) {
    const normalized = levelToken
        .replace(/^\[/, '')
        .replace(/\]$/, '')
        .replace(/:$/, '')
        .trim()
        .toLowerCase();

    switch (normalized) {
        case 'trace':
            return 'var(--blue)';
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

        const normalizedLevel = levelToken
            .replace(/^\[/, '')
            .replace(/\]$/, '')
            .replace(/:$/, '')
            .trim()
            .toLowerCase();
        if (normalizedLevel) {
            lineEl.dataset.level = normalizedLevel;
        }

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

const DEFAULT_MAX_LOG_LINES = 1000;

function normalizeMaxLogLines(value) {
    const parsed = Number.parseInt(value, 10);
    if (!Number.isFinite(parsed) || parsed <= 0) {
        return DEFAULT_MAX_LOG_LINES;
    }
    return parsed;
}

function trimLogLines(logOutput, maxLogLines) {
    while (logOutput.childElementCount > maxLogLines) {
        logOutput.removeChild(logOutput.firstElementChild);
    }
}

function appendColourisedLogMessage(logOutput, message, maxLogLines, stickToBottom = true) {
    const lines = String(message ?? '').split(/\r?\n/);
    for (let i = 0; i < lines.length; i++) {
        if (i === lines.length - 1 && lines[i] === '') {
            continue;
        }
        appendColourisedLogLine(logOutput, lines[i]);
    }
    trimLogLines(logOutput, maxLogLines);
    if (stickToBottom) {
        logOutput.scrollTop = logOutput.scrollHeight;
    }
}

const LOG_BOTTOM_THRESHOLD_PX = 24;

export function initNotesLogPanel(elements, eventsOn, maxLogLines = DEFAULT_MAX_LOG_LINES) {
    if (!elements?.logOutput) {
        return;
    }

    const appRoot = elements.appRoot || document.getElementById('notes-pane') || document.getElementById('app') || document.body;
    let isWordWrapped = false;
    let isTimestampVisible = false;
    let isTraceVisible = false;
    let isLogMaximized = false;
    let restoreWordWrapOnClose = false;
    let restoreTimestampOnClose = false;
    let lineLimit = normalizeMaxLogLines(maxLogLines);

    const isLogOutputNearBottom = () => {
        const remaining = elements.logOutput.scrollHeight - (elements.logOutput.scrollTop + elements.logOutput.clientHeight);
        return remaining <= LOG_BOTTOM_THRESHOLD_PX;
    };

    const updateLogScrollBottomButton = () => {
        if (!elements.logScrollBottom) {
            return;
        }
        elements.logScrollBottom.dataset.visible = isLogOutputNearBottom() ? 'false' : 'true';
    };

    if (elements.logScrollBottom) {
        elements.logOutput.addEventListener('scroll', updateLogScrollBottomButton);

        elements.logScrollBottom.addEventListener('click', () => {
            elements.logOutput.scrollTop = elements.logOutput.scrollHeight;
            updateLogScrollBottomButton();
        });
    }

    const setWordWrap = (enabled) => {
        isWordWrapped = enabled === true;
        if (isWordWrapped) {
            elements.logOutput.style.whiteSpace = 'pre-wrap';
            elements.logOutput.style.overflowWrap = 'break-word';
        } else {
            elements.logOutput.style.whiteSpace = 'pre';
            elements.logOutput.style.overflowWrap = 'normal';
        }

        elements.logOutput.dataset.wordwrap = isWordWrapped ? 'true' : 'false';

        if (elements.toolsLogWordwrap) {
            elements.toolsLogWordwrap.dataset.wrapped = isWordWrapped ? 'true' : 'false';
        }
    };

    const setTimestampVisible = (enabled) => {
        isTimestampVisible = enabled === true;
        if (elements.toolsLogTimestamp) {
            elements.toolsLogTimestamp.dataset.enabled = isTimestampVisible ? 'true' : 'false';
        }
        elements.logOutput.dataset.showTimestamp = isTimestampVisible ? 'true' : 'false';
    };

    const setTraceVisible = (enabled) => {
        isTraceVisible = enabled === true;
        if (elements.toolsLogTrace) {
            elements.toolsLogTrace.dataset.enabled = isTraceVisible ? 'true' : 'false';
        }
        elements.logOutput.dataset.showTrace = isTraceVisible ? 'true' : 'false';
    };

    const setLogMaximized = (enabled) => {
        const nextState = enabled === true;
        if (nextState === isLogMaximized) {
            return;
        }

        isLogMaximized = nextState;
        appRoot.dataset.logMaximized = isLogMaximized ? 'true' : 'false';

        if (elements.toolsLogMaximize) {
            elements.toolsLogMaximize.dataset.enabled = isLogMaximized ? 'true' : 'false';
            elements.toolsLogMaximize.textContent = 'Maximize';
        }

        if (isLogMaximized) {
            restoreWordWrapOnClose = isWordWrapped;
            restoreTimestampOnClose = isTimestampVisible;
            setWordWrap(true);
            setTimestampVisible(true);
        } else {
            setWordWrap(restoreWordWrapOnClose);
            setTimestampVisible(restoreTimestampOnClose);
        }
    };

    const handleKeyDownCloseMaximized = (event) => {
        if (!isLogMaximized) {
            return;
        }

        if (event.defaultPrevented) {
            return;
        }

        setLogMaximized(false);
    };

    window.addEventListener('keydown', handleKeyDownCloseMaximized, true);

    if (elements.toolsLogClear) {
        elements.toolsLogClear.addEventListener('click', () => {
            elements.logOutput.textContent = '';
        });
    }

    if (elements.logOutput) {
        elements.logOutput.addEventListener('click', (event) => {
            const line = event.target.closest('.notes-log-line');
            if (!line || !elements.logOutput.contains(line)) {
                return;
            }

            line.dataset.selected = line.dataset.selected === 'true' ? 'false' : 'true';
        });
    }

    if (elements.toolsLogDeselect) {
        elements.toolsLogDeselect.addEventListener('click', () => {
            const selectedLines = elements.logOutput.querySelectorAll('.notes-log-line[data-selected="true"]');
            for (const line of selectedLines) {
                line.dataset.selected = 'false';
            }
        });
    }

    if (elements.toolsLogCopy) {
        elements.toolsLogCopy.addEventListener('click', async () => {
            const lines = Array.from(elements.logOutput.querySelectorAll('.notes-log-line'));
            const selectedLines = lines.filter((line) => line.dataset.selected === 'true');
            const linesToCopy = selectedLines.length > 0 ? selectedLines : lines;
            const text = linesToCopy.length > 0
                ? linesToCopy.map((line) => String(line.textContent || '')).join('\n')
                : String(elements.logOutput.textContent || '');

            const triggerCopyFeedback = () => {
                elements.toolsLogCopy.dataset.copied = 'false';
                // Force reflow so repeated clicks replay the transition.
                void elements.toolsLogCopy.offsetWidth;
                elements.toolsLogCopy.dataset.copied = 'true';
                window.setTimeout(() => {
                    elements.toolsLogCopy.dataset.copied = 'false';
                }, 520);
            };

            try {
                await navigator.clipboard.writeText(text);
                triggerCopyFeedback();
            } catch (err) {
                try {
                    const textarea = document.createElement('textarea');
                    textarea.value = text;
                    textarea.setAttribute('readonly', 'true');
                    textarea.style.position = 'fixed';
                    textarea.style.opacity = '0';
                    textarea.style.pointerEvents = 'none';
                    document.body.appendChild(textarea);
                    textarea.select();
                    const ok = document.execCommand('copy');
                    document.body.removeChild(textarea);
                    if (!ok) {
                        throw new Error('document.execCommand(copy) returned false');
                    }

                    triggerCopyFeedback();
                } catch (fallbackErr) {
                    console.error('Failed to copy log output:', fallbackErr || err);
                }
            }
        });
    }

    if (elements.toolsLogWordwrap) {
        elements.toolsLogWordwrap.addEventListener('click', () => {
            setWordWrap(!isWordWrapped);
        });
    }

    if (elements.toolsLogTimestamp && elements.logOutput) {
        elements.toolsLogTimestamp.addEventListener('click', () => {
            setTimestampVisible(!isTimestampVisible);
        });
    }

    if (elements.toolsLogTrace && elements.logOutput) {
        elements.toolsLogTrace.addEventListener('click', () => {
            setTraceVisible(!isTraceVisible);
        });
    }

    if (elements.toolsLogMaximize) {
        elements.toolsLogMaximize.addEventListener('click', () => {
            setLogMaximized(!isLogMaximized);
        });
    }

    // No-wrap and hidden timestamps by default. Trace lines are hidden by default too.
    setWordWrap(false);
    setTimestampVisible(false);
    setTraceVisible(false);

    if (elements.toolsLogMaximize) {
        elements.toolsLogMaximize.dataset.enabled = 'false';
        elements.toolsLogMaximize.textContent = 'Maximize';
    }

    eventsOn('notesLog', (message) => {
        // Only auto-scroll when already at the bottom; otherwise leave the
        // scroll position untouched and let the Latest button reveal itself.
        const stickToBottom = isLogOutputNearBottom();
        appendColourisedLogMessage(elements.logOutput, message, lineLimit, stickToBottom);
        updateLogScrollBottomButton();
    });

    return {
        setMaxLogLines(nextMaxLogLines) {
            lineLimit = normalizeMaxLogLines(nextMaxLogLines);
            trimLogLines(elements.logOutput, lineLimit);
        },
    };
}
