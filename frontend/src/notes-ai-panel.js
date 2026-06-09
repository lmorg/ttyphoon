export function initNotesAIPanel(elements) {
    if (!elements?.aiOutput) {
        return;
    }

    const appRoot = elements.appRoot || document.getElementById('notes-pane') || document.getElementById('app') || document.body;
    let isAIMaximized = false;

    const setAIMaximized = (enabled) => {
        const nextState = enabled === true;
        if (nextState === isAIMaximized) {
            return;
        }

        isAIMaximized = nextState;
        appRoot.dataset.aiMaximized = isAIMaximized ? 'true' : 'false';

        if (elements.toolsAIMaximize) {
            elements.toolsAIMaximize.dataset.enabled = isAIMaximized ? 'true' : 'false';
            elements.toolsAIMaximize.textContent = 'Maximize';
        }
    };

    const handleKeyDownCloseMaximized = (event) => {
        if (!isAIMaximized) {
            return;
        }

        if (event.defaultPrevented) {
            return;
        }

        setAIMaximized(false);
    };

    window.addEventListener('keydown', handleKeyDownCloseMaximized, true);

    if (elements.toolsAIMaximize) {
        elements.toolsAIMaximize.addEventListener('click', () => {
            setAIMaximized(!isAIMaximized);
        });

        elements.toolsAIMaximize.dataset.enabled = 'false';
        elements.toolsAIMaximize.textContent = 'Maximize';
    }
}
