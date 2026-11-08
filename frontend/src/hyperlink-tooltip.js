export function createHyperlinkTooltip({ getTooltip, showTooltip, closeTooltip, reportError }) {
    const cache = new WeakMap();
    let active = null;

    return {
        async show(anchor, href, fallback) {
            if (active?.anchor === anchor && active.href === href) return;
            const request = { anchor, href };
            active = request;
            closeTooltip();
            if (!/^(https?|file):/i.test(href)) {
                await showTooltip(fallback);
                return;
            }
            let entry = cache.get(anchor);
            if (!entry || entry.href !== href) {
                entry = { href, promise: Promise.resolve().then(() => getTooltip(href)) };
                cache.set(anchor, entry);
            }
            try {
                const html = await entry.promise;
                if (active === request && anchor.isConnected) {
                    await showTooltip(html);
                }
            } catch (error) {
                if (cache.get(anchor) === entry) cache.delete(anchor);
                reportError(error);
                if (active === request) {
                    active = null;
                    closeTooltip();
                }
            }
        },
        close() {
            active = null;
            closeTooltip();
        },
    };
}
