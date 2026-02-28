import { GetWindowStyle } from '../wailsjs/go/main/WApp';
import { GetParameters, GetMarkdown, GetImage } from '../wailsjs/go/main/WApp';

import { EventsOn, BrowserOpenURL } from '../wailsjs/runtime/runtime';

import { marked } from "marked";
import { gfmHeadingId } from "marked-gfm-heading-id";
import hljs from "highlight.js/lib/common";

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
    style.textContent = `
        ::selection {
            background-color: rgb(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue});
        }
        h1, h2, h3, h4, h5, h6 {
            color: rgb(${result.colors.yellow.Red}, ${result.colors.yellow.Green}, ${result.colors.yellow.Blue});
        }
        a {
            text-decoration: none;
            color: rgb(${result.colors.link.Red}, ${result.colors.link.Green}, ${result.colors.link.Blue});
        }
        a:hover {
            text-decoration: underline;
        }
        div {
            font-size: ${result.fontSize}px;
            font-family: ${result.fontFamily};
        }
        pre, code {
            color: rgb(${result.colors.green.Red}, ${result.colors.green.Green}, ${result.colors.green.Blue});
        }
        pre {
            border: 0px;
            border-left: 2px;
            border-style: solid;
            margin: 0px;
            padding: 10px;
            padding-left: 20px;
        }
        blockquote {
            border: 0px;
            border-left: 2px;
            border-style: solid;
            margin: 0px;
            padding: 1px;
            padding-left: 20px;
            color: rgb(${result.colors.magenta.Red}, ${result.colors.magenta.Green}, ${result.colors.magenta.Blue});
        }
        details {
            opacity: 50%;
            width: 100%;
            border-radius: 0px;
            border-width: 2px;
            border-style: solid;
            padding: 5px;
            margin-top: 5px;
        }
        summary {
            cursor: pointer;
        }

        /* Highlight.js syntax highlighting theme */
        pre code.hljs {
            display: block;
            overflow-x: auto;
            background: transparent;
            color: rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue});
        }
        .hljs-comment, .hljs-quote {
            color: rgb(${result.colors.blueBright.Red}, ${result.colors.blueBright.Green}, ${result.colors.blueBright.Blue});
            font-style: italic;
        }
        .hljs-keyword, .hljs-selector-tag, .hljs-subst {
            color: rgb(${result.colors.magenta.Red}, ${result.colors.magenta.Green}, ${result.colors.magenta.Blue});
            font-weight: bold;
        }
        .hljs-string, .hljs-title, .hljs-name, .hljs-type, .hljs-attribute, .hljs-symbol, .hljs-bullet, .hljs-addition, .hljs-built_in {
            color: rgb(${result.colors.green.Red}, ${result.colors.green.Green}, ${result.colors.green.Blue});
        }
        .hljs-number, .hljs-literal, .hljs-variable, .hljs-template-variable {
            color: rgb(${result.colors.yellow.Red}, ${result.colors.yellow.Green}, ${result.colors.yellow.Blue});
        }
        .hljs-section, .hljs-meta, .hljs-function, .hljs-class, .hljs-title.class_ {
            color: rgb(${result.colors.cyan.Red}, ${result.colors.cyan.Green}, ${result.colors.cyan.Blue});
        }
        .hljs-deletion, .hljs-regexp, .hljs-link {
            color: rgb(${result.colors.red.Red}, ${result.colors.red.Green}, ${result.colors.red.Blue});
        }
        .hljs-punctuation, .hljs-tag {
            color: rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue});
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
    const options = {};
    marked.use(gfmHeadingId(options));
    document.getElementById('ttyphoon-markdown').innerHTML = marked.parse(doc);

    // Apply syntax highlighting to all code blocks
    document.querySelectorAll('pre code').forEach((block) => {
        hljs.highlightElement(block);
    });

    let rxWailsUrl = /^(wails:\/\/wails\/|http:\/\/localhost:[0-9]+\/|wails:\/\/wails.localhost:[0-9]+\/)/;

    document.querySelectorAll('img').forEach(img => {
        //console.log(img.src);
        
        if (img.src.match(rxWailsUrl)) {
            let path = img.src.replace(rxWailsUrl, '')
            GetImage(path).then((image) => {
                if (image.match(/^error: /)) {
                    console.log(image);
                    //document.getElementById('markdown').innerText = image;
                } else {
                    //console.log(image);
                    img.src = image;
                }
            })
        }
    
    });

    let rxBookmark = /^(wails:\/\/wails\/|http:\/\/localhost:[0-9]+\/|wails:\/\/wails.localhost:[0-9]+\/)#/;

    document.querySelectorAll('a').forEach(a => {
        if (!a.href.match(rxWailsUrl)) {
            a.addEventListener('click', (e) => {
                e.preventDefault();
                BrowserOpenURL(a.href);
            });
        }

        if (!a.href.match(rxBookmark)) {
            /*let id = a.href.replace(rxBookmark, '');
            console.log(id);
            //a.href = "#"+id;
            a.addEventListener("click", () => {
                document.getElementById(id).scrollIntoView();
            });*/
        }
    });

    GetWindowStyle().then((result) => {
        document.querySelectorAll('div').forEach(div => {
            div.style.fontFamily = result.fontFamily;
            div.style.fontSize   = result.fontSize;
        });

    });
};
