import './style.css';
import './app.css';

//import logo from './assets/images/logo-universal.png';
import { VisualInputBox } from '../wailsjs/go/main/WApp';
//import { main } from "../wailsjs/go/models";



document.querySelector('#app').innerHTML = `
    <img id="logo" class="logo">
      <div class="result" id="result">bob</div>
      <div class="input-box" id="input">
        <input class="input" id="name" type="text" autocomplete="off" />
        <button class="btn" onclick="send()">Send</button>
      </div>
    </div>
`;

//document.body.style.backgroundColor = `rgb(${payload.window.bg.red}, ${payload.window.bg.green}, ${payload.window.bg.blue})`;
//document.getElementById('logo').src = logo;

let nameElement = document.getElementById("name");
nameElement.focus();
let resultElement = document.getElementById("result");

// Setup the greet function
window.send = function () {
    // Get name
    let name = nameElement.value;

    // Check if the input is empty
    //if (name === "") return;

    // Call App.Greet(name)
    try {
        VisualInputBox(name)
            .then((result) => {
                // Update result with data back from App.Greet()
                resultElement.innerText = result;
            })
            .catch((err) => {
                console.error(err);
            });
    } catch (err) {
        console.error(err);
    }
};
