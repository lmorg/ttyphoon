let preparation = null;
let clearPrintDocument = () => {};

export function discardPrintDocument() {
    clearPrintDocument();
}

async function waitForAssets(container) {
    const assets = [...container.querySelectorAll('img')].map(async (image) => {
        image.loading = 'eager';
        if (image.decode) await image.decode().catch(() => {});
    });
    if (document.fonts?.ready) assets.push(document.fonts.ready);
    let timer;
    try {
        await Promise.race([
            Promise.all(assets),
            new Promise((resolve) => { timer = setTimeout(resolve, 10000); }),
        ]);
    } finally {
        clearTimeout(timer);
    }
}

export function printDocument({ source, filename = '', renderMarkdown, dismissMenu, print }) {
    if (preparation) return preparation;
    preparation = (async () => {
        clearPrintDocument();
        await dismissMenu();
        const menuAnimations = [...document.querySelectorAll('.tty-menu')]
            .flatMap((menu) => menu.getAnimations?.() || []);
        await Promise.allSettled(menuAnimations.map((animation) => animation.finished));
        const container = document.createElement('article');
        container.id = 'notes-print-root';
        container.className = 'markdown-body';
        container.setAttribute('aria-hidden', 'true');
        document.body.appendChild(container);
        let operationActive = false;
        const cleanup = () => {
            container.remove();
            document.documentElement.classList.remove('notes-print-ready');
            window.removeEventListener('afterprint', afterPrint);
        };
        const afterPrint = () => {
            if (!operationActive) cleanup();
        };
        clearPrintDocument = cleanup;
        try {
            if (renderMarkdown) {
                await renderMarkdown(container, source);
            } else {
                if (filename) {
                    const heading = document.createElement('div');
                    heading.className = 'notes-print-filename';
                    heading.textContent = filename;
                    container.appendChild(heading);
                }
                const block = document.createElement('pre');
                const code = document.createElement('code');
                code.textContent = source;
                block.appendChild(code);
                container.appendChild(block);
            }
            container.querySelectorAll('details').forEach((details) => { details.open = true; });
            await waitForAssets(container);
            const fontSizes = [container, ...container.querySelectorAll('*')]
                .map((element) => [element, getComputedStyle(element).fontSize]);
            for (const [element, fontSize] of fontSizes) {
                if (fontSize) element.style.setProperty('--notes-print-font-size', fontSize);
            }
            await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)));
            document.documentElement.classList.add('notes-print-ready');
            window.addEventListener('afterprint', afterPrint);
            operationActive = true;
            try {
                await print();
            } finally {
                operationActive = false;
            }
        } catch (error) {
            cleanup();
            throw error;
        }
    })().finally(() => { preparation = null; });
    return preparation;
}