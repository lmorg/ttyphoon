import { GetWindowStyle, GetParameters, SendIpc } from '../wailsjs/go/main/WApp';
import { SendVisualInputBox } from '../wailsjs/go/main/WApp';

import { WindowSetSize, Quit } from '../wailsjs/runtime/runtime';

function autoGrow() {
    WindowSetSize(window.innerWidth, document.getElementById('app').clientHeight);
}

function moveCaret(){
    const range = document.createRange();
    range.selectNodeContents(element.input);
    range.collapse(false);
    const sel = window.getSelection();
    sel.removeAllRanges();
    sel.addRange(range);
}

window.autoGrow = autoGrow;

document.querySelector('#app').innerHTML = `
    <h1 class="title" id="title">{{Title}}</h1>
    <div class="input-box">
        <div id="input" class="input" contenteditable="plaintext-only" onkeydown="autoGrow();" onkeyup="autoGrow();"  data-placeholder="Enter your text..."></div>
        <div id="notes-display"><input type="checkbox" id="notes-checkbox" /><label id="notes-label" for="notes-checkbox">Save to TTYphoon Notes</label></div>
        <div class="toolbar">
            <div id="btn-send" class="btn" onclick="send()">Send [ctrl+return]</div>
            <div id="btn-quit" class="btn" onclick="Quit()">Cancel [esc]</div>
            <select id="history" class="history"></select>
        </div>
    </div>
`;

const element = {
    title: document.getElementById("title"),
    input: document.getElementById("input"),
    notesDisplay:  document.getElementById("notes-display"),
    notesCheckbox: document.getElementById("notes-checkbox"),
    dropdown: document.getElementById('history'),
};

element.dropdown.addEventListener('change', (e) => {
    if (e.target.value) {
        element.input.innerText = e.target.value;
        e.target.value = '';
        element.input.focus();
        moveCaret();
        autoGrow();
    }
});

setTimeout(() => element.input.focus(), 0);

GetParameters().then((result) => {
    element.title.innerText = result.title

    if (!result.notesDisplay) element.notesDisplay.style.display = 'none';
    element.notesCheckbox.checked = result.notesDefault;

    if (result.prefill !== "") {
        element.input.innerText = result.prefill;
        moveCaret();
        autoGrow();
    }

    if (result.history.length > 0) {
        element.dropdown.style.visibility = "visible";
        const placeholderOption = document.createElement('option');
        placeholderOption.value = '';
        placeholderOption.textContent = 'History...';
        placeholderOption.disabled = true;
        placeholderOption.selected = true;
        element.dropdown.appendChild(placeholderOption);

        result.history.forEach(option => {
            const optionElement = document.createElement('option');
            optionElement.value = option;
            optionElement.textContent = option;
            element.dropdown.appendChild(optionElement);
        });
    }

    //if (result.placeholder != "") {
    element.input.setAttribute('data-placeholder', result.placeholder);
    //};
});

GetWindowStyle().then((result) => {    
    const style = document.createElement('style');
    style.textContent = `
        :root {
            --bg: rgb(${result.colors.bg.Red}, ${result.colors.bg.Green}, ${result.colors.bg.Blue});
            --fg: rgb(${result.colors.fg.Red}, ${result.colors.fg.Green}, ${result.colors.fg.Blue});
            --accent: rgb(${result.colors.yellow.Red}, ${result.colors.yellow.Green}, ${result.colors.yellow.Blue});
            --link: rgb(${result.colors.link.Red}, ${result.colors.link.Green}, ${result.colors.link.Blue});
            --red: rgb(${result.colors.red.Red}, ${result.colors.red.Green}, ${result.colors.red.Blue});
            --green: rgb(${result.colors.green.Red}, ${result.colors.green.Green}, ${result.colors.green.Blue});
            --yellow: rgb(${result.colors.yellow.Red}, ${result.colors.yellow.Green}, ${result.colors.yellow.Blue});
            --blue: rgb(${result.colors.blue.Red}, ${result.colors.blue.Green}, ${result.colors.blue.Blue});
            --magenta: rgb(${result.colors.magenta.Red}, ${result.colors.magenta.Green}, ${result.colors.magenta.Blue});
            --cyan: rgb(${result.colors.cyan.Red}, ${result.colors.cyan.Green}, ${result.colors.cyan.Blue});
            --red-bright: rgb(${result.colors.redBright.Red}, ${result.colors.redBright.Green}, ${result.colors.redBright.Blue});
            --green-bright: rgb(${result.colors.greenBright.Red}, ${result.colors.greenBright.Green}, ${result.colors.greenBright.Blue});
            --yellow-bright: rgb(${result.colors.yellowBright.Red}, ${result.colors.yellowBright.Green}, ${result.colors.yellowBright.Blue});
            --blue-bright: rgb(${result.colors.blueBright.Red}, ${result.colors.blueBright.Green}, ${result.colors.blueBright.Blue});
            --magenta-bright: rgb(${result.colors.magentaBright.Red}, ${result.colors.magentaBright.Green}, ${result.colors.magentaBright.Blue});
            --cyan-bright: rgb(${result.colors.cyanBright.Red}, ${result.colors.cyanBright.Green}, ${result.colors.cyanBright.Blue});
            --selection: rgb(${result.colors.selection.Red}, ${result.colors.selection.Green}, ${result.colors.selection.Blue});
            --error: rgb(${result.colors.error.Red}, ${result.colors.error.Green}, ${result.colors.error.Blue});
        }

        ::-webkit-scrollbar { display: none; }
        * { -ms-overflow-style: none; overflow-y: scroll; }

        ::selection {
            background-color: var(--selection);
        }

        body {
            background: var(--bg);
            color: var(--fg);
        }
        
        h1, h2, h3, h4, h5, h6 {
            color: var(--accent);
        }
        a {
            text-decoration: none;
            color: var(--link);
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
            color: var(--fg);
            visibility: hidden;
        }
        select:hover {
            border-color: var(--selection);
        }
        pre, code {
            color: var(--green);
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
            color: var(--magenta);
        }

        #input {
            background: var(--bg);
            color: var(--fg);
            border: 2px;
            border-style: solid;
            border-color: var(--fg);
        }

        #input:empty::before {
            content: attr(data-placeholder);
            color: var(--fg);
            opacity: 0.5;
            pointer-events: none;
        }

        #notes-display {
            margin-top: 15px;
            margin-left: 0px;
            display: flex;
            align-items: center;
        }

        input[type="checkbox"] {
            appearance: none;
            -webkit-appearance: none;
            -moz-appearance: none;
            cursor: pointer;
            margin-right: 5px;
            margin-left: 0px;
            width: ${result.fontSize}px;
            height: ${result.fontSize}px;
            border: 2px solid var(--red);
            background: transparent;
            position: relative;
            vertical-align: middle;
            flex-shrink: 0;
        }

        input[type="checkbox"]:hover {
            border-color: var(--accent);
        }

        input[type="checkbox"]:checked:hover {
            border-color: var(--accent);
            background: var(--accent);
        }

        input[type="checkbox"]:checked {
            background: var(--green);
            border-color: var(--green);
        }

        input[type="checkbox"]:checked::after {
            content: '✓';
            position: absolute;
            color: var(--bg);
            font-size: ${result.fontSize}px;
            font-weight: bold;
            left: 50%;
            top: 50%;
            transform: translate(-50%, -50%);
        }

        #notes-label {
            cursor: pointer;
            vertical-align: middle;
        }

        #btn-send:hover {
            border-color: var(--green);
            color: var(--green);
        }

        #btn-quit:hover {
            border-color: var(--error);
            color: var(--error);
        }

    `;
    document.head.appendChild(style);
})

element.input.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
        Quit();
    } else if (e.key === "Enter" && e.ctrlKey) {
        e.preventDefault();
        window.send();
    };
});

element.input.addEventListener("keyup", (e) => {
    SendIpc("keyPress", { value: element.input.innerText });
});

window.Quit = Quit
window.send = function () {
    try {
        SendVisualInputBox(element.input.innerText, element.notesCheckbox.checked);
    } catch (err) {
        console.error(err);
    }
};

setTimeout(window.autoGrow, 1);
