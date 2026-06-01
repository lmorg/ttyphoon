import { afterEach } from 'vitest';

const localStorageState = new Map();
const localStorageMock = {
    getItem(key) {
        return localStorageState.has(key) ? localStorageState.get(key) : null;
    },
    setItem(key, value) {
        localStorageState.set(String(key), String(value));
    },
    removeItem(key) {
        localStorageState.delete(String(key));
    },
    clear() {
        localStorageState.clear();
    },
    key(index) {
        return Array.from(localStorageState.keys())[index] ?? null;
    },
    get length() {
        return localStorageState.size;
    },
};

Object.defineProperty(globalThis, 'localStorage', {
    value: localStorageMock,
    configurable: true,
});

if (typeof window !== 'undefined') {
    Object.defineProperty(window, 'localStorage', {
        value: localStorageMock,
        configurable: true,
    });
}

if (typeof Element !== 'undefined' && typeof Element.prototype.scrollIntoView !== 'function') {
    Object.defineProperty(Element.prototype, 'scrollIntoView', {
        value: () => {},
        configurable: true,
        writable: true,
    });
}

afterEach(() => {
    document.body.innerHTML = '';
    localStorageMock.clear();
});