import { GetWindowStyle, GetParameters, SendIpc } from '../wailsjs/go/main/WApp';
import { VisualInputBox } from '../wailsjs/go/main/WApp';

import { EventsOn, WindowSetSize, Quit } from '../wailsjs/runtime/runtime';

function autoGrow() {
    WindowSetSize(window.innerWidth, document.getElementById('app').clientHeight);
}

function moveCaret(){
    const range = document.createRange();
    range.selectNodeContents(inputElement);
    range.collapse(false);
    const sel = window.getSelection();
    sel.removeAllRanges();
    sel.addRange(range);
}

window.autoGrow = autoGrow;

document.querySelector('#app').innerHTML = `
    <h1 class="title" id="title">{{Title}}</h1>
    <div class="input-box">
        <div class="input" id="input" contenteditable="plaintext-only" onkeydown="autoGrow();" onkeyup="autoGrow();"></div>
        <div class="toolbar">
            <div class="btn" onclick="send()">Send [ctrl+return]</div>
            <div class="btn" onclick="Quit()">Cancel [esc]</div>
            <select class="history" id="history"></select>
        </div>
    </div>
`;

const inputElement = document.getElementById("input");
const titleElement = document.getElementById("title");
const dropdown = document.getElementById('history');

dropdown.addEventListener('change', (e) => {
    if (e.target.value) {
        inputElement.innerText = e.target.value;
        e.target.value = '';
        inputElement.focus();
        moveCaret();
        autoGrow();
    }
});

const scrollbarStyle = document.createElement('style');
scrollbarStyle.textContent = `
    ::-webkit-scrollbar { display: none; }
    * { -ms-overflow-style: none; overflow-y: scroll; }
`;
document.head.appendChild(scrollbarStyle);

setTimeout(() => inputElement.focus(), 0);

GetParameters().then((result) => {
    titleElement.innerText = result.title

    if (result.prefill !== "") {
        inputElement.innerText = result.prefill;
        moveCaret();
        autoGrow();
    }

    if (result.history.length > 0) {
        dropdown.style.visibility = "visible";
        const placeholderOption = document.createElement('option');
        placeholderOption.value = '';
        placeholderOption.textContent = 'History...';
        placeholderOption.disabled = true;
        placeholderOption.selected = true;
        dropdown.appendChild(placeholderOption);

        result.history.forEach(option => {
            const optionElement = document.createElement('option');
            optionElement.value = option;
            optionElement.textContent = option;
            dropdown.appendChild(optionElement);
        });
    }
});

GetWindowStyle().then((result) => {
    document.body.style.color           = `rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue})`;
    document.body.style.backgroundColor = `rgb(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue})`;
    inputElement.style.color            = `rgb(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue})`;
    inputElement.style.backgroundColor  = `rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue})`;
    
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
        select {
            font-size: ${result.fontSize}px;
            font-family: ${result.fontFamily};
            color: rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue});
            visibility: hidden;
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
    `;
    document.head.appendChild(style);
})

inputElement.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
        Quit();
    } else if (e.key === "Enter" && e.ctrlKey) {
        e.preventDefault();
        window.send();
    };
});

inputElement.addEventListener("keyup", (e) => {
    SendIpc("keyPress", { value: inputElement.innerText });
});

window.Quit = Quit
window.send = function () {
    try {
        VisualInputBox(inputElement.innerText)
    } catch (err) {
        console.error(err);
    }
};

setTimeout(window.autoGrow, 1);
