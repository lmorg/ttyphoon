import { GetWindowStyle } from '../wailsjs/go/main/WApp';
import { GetParameters, GetMarkdown, GetImage } from '../wailsjs/go/main/WApp';

import { BrowserOpenURL } from '../wailsjs/runtime/runtime';

import { marked } from "marked";
import { gfmHeadingId } from "marked-gfm-heading-id";

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
        a {
            text-decoration: none;
            color: rgb(${result.colors.link.Red}, ${result.colors.link.Green}, ${result.colors.link.Blue});
        }
        a:hover {
            text-decoration: underline;
        }
    `;
    document.head.appendChild(style);
});

GetParameters().then((result) => {
    GetMarkdown(result.path).then((doc) => {
        markdown(doc);
    });
});

function markdown(doc) {
    const options = {};
    marked.use(gfmHeadingId(options));
    document.getElementById('ttyphoon-markdown').innerHTML = marked.parse(doc);

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
        document.querySelectorAll('a').forEach(a => {
            /*a.style.color = `rgb(${result.colors.link.Red}, ${result.colors.link.Green}, ${result.colors.link.Blue})`;
            a.style.textDecoration = "none";

                const style = a.createElement('style');
                style.textContent = `:hover {
                    border-width: 1px;
                    border-style: solid;
                    border-color: rgb(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue});
                }`;
                a.head.appendChild(style);*/
        });
    });
};

/*GetPayload().then((result) => {
    document.getElementById('output').innerHTML = result;
})*/
