import { defineConfig } from 'vitest/config';

export default defineConfig({
    test: {
        environment: 'jsdom',
        include: ['src/**/*.test.js'],
        setupFiles: ['./vitest.setup.js'],
        clearMocks: true,
        restoreMocks: true,
    },
});