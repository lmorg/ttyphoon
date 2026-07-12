import { defineConfig } from 'vite';

export default defineConfig({
    build: {
        chunkSizeWarningLimit: 2600,
        rollupOptions: {
            output: {
                manualChunks(id) {
                    if (!id.includes('node_modules')) {
                        return undefined;
                    }

                    if (id.includes('monaco-editor')) {
                        return 'vendor-monaco';
                    }

                    if (id.includes('mermaid')) {
                        return 'vendor-mermaid';
                    }

                    if (id.includes('highlight.js')
                        || id.includes('highlight-js-murex')
                        || id.includes('highlight-js-terraform')) {
                        return 'vendor-highlight';
                    }

                    if (id.includes('katex')) {
                        return 'vendor-katex';
                    }

                    if (id.includes('cytoscape')) {
                        return 'vendor-cytoscape';
                    }

                    return undefined;
                },
            },
        },
    },
});
