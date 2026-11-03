import { GetNotesLayoutSizing, SetNotesLayoutRatio } from './wails-bindings.js';

let sizingPromise;
const selectedRatios = new Map();

export function validLayoutRatio(ratio) {
    return typeof ratio === 'number' && Number.isFinite(ratio) && ratio > 0 && ratio < 1;
}

export async function restoreLayoutRatio(name, fallback) {
    sizingPromise ||= Promise.resolve().then(() => GetNotesLayoutSizing()).catch((err) => {
        console.error('Failed to restore layout sizes:', err);
        return {};
    });
    const sizing = await sizingPromise;
    const ratio = selectedRatios.has(name) ? selectedRatios.get(name) : sizing?.[name];
    return validLayoutRatio(ratio) ? ratio : fallback;
}

export function saveLayoutRatio(name, ratio) {
    selectedRatios.set(name, ratio);
    return Promise.resolve().then(() => SetNotesLayoutRatio(name, ratio)).catch((err) => {
        console.error('Failed to save layout size:', err);
    });
}