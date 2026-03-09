import { GetWindowStyle } from '../wailsjs/go/main/WApp';
import { GetParameters, GetMarkdown, SendIpc } from '../wailsjs/go/main/WApp';

import { EventsOn, Quit } from '../wailsjs/runtime/runtime';

import { marked } from "marked";
import { configureMarked, processMarkdownContainer } from './markdown-utils.js';
import { getAllMarkdownStyles, getMarkdownBaseTextSizeStyles } from './style-utils.js';

document.querySelector('#app').innerHTML = `
    <div id="ttyphoon-error"></div>
    <div id="ttyphoon-markdown"></div>
`;

let errorElement = document.getElementById('ttyphoon-error')

GetWindowStyle().then((result) => {
    document.body.style.color           = `rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue})`;
    document.body.style.backgroundColor = `rgb(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue})`;
    errorElement.style.color = `rgb(${result.colors.error.Red}, ${result.colors.error.Green}, ${result.colors.error.Blue})`;

    const style = document.createElement('style');
    // Use shared markdown styles utility
    style.textContent = getAllMarkdownStyles(result, {
        classPrefix: '',      // No class prefix for global styles
        useCssVars: false,    // Use explicit colors, not CSS variables
        includeCheckboxes: false
    }) + `
        html, body {
            height: 100%;
            margin: 0;
            padding: 0;
            overflow: hidden;
        }

        #app {
            height: 100%;
            display: flex;
            flex-direction: column;
            overflow: hidden;
        }

        #ttyphoon-markdown {
            flex: 1;
            overflow-y: auto;
            overflow-x: hidden;
            padding-left: 16px;
        }

        ${getMarkdownBaseTextSizeStyles('#ttyphoon-markdown', result.fontSize)}

        #ttyphoon-error {
            padding: 16px;
        }

        div {
            font-size: ${result.fontSize}px;
            font-family: ${result.fontFamily};
        }
    `;
    document.head.appendChild(style);
});

GetParameters().then((result) => {
    GetMarkdown(result.path).then((doc) => {
        markdown(doc);
    });
});

EventsOn("markdownOpen", params => {
    GetMarkdown(params.path).then((doc) => {
        markdown(doc);
    })
});

function markdown(doc) {
    configureMarked();
    const container = document.getElementById('ttyphoon-markdown');
    container.innerHTML = marked.parse(doc);

    // Apply all common markdown processing
    processMarkdownContainer(container);

    GetWindowStyle().then((result) => {
        document.querySelectorAll('div').forEach(div => {
            div.style.fontFamily = result.fontFamily;
            div.style.fontSize   = result.fontSize;
        });
    });
};

document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        SendIpc("focus", {});
        Quit();
    }
});