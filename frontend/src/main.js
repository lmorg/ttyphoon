import './style.css';
import './app.css';

//import logo from './assets/images/logo-universal.png';
import { GetPayload, GetWindowStyle, GetParameters } from '../wailsjs/go/main/WApp';
import { VisualInputBox } from '../wailsjs/go/main/WApp';

document.querySelector('#app').innerHTML = `
    <!--<img id="logo" class="logo">-->
    <div class="title" id="title">stuff</div>
      <div class="input-box" id="input">
        <input class="input" id="name" type="text" autocomplete="off" />
        <button class="btn" onclick="send()">Send</button>
      </div>
    </div>
`;

let nameElement = document.getElementById("name");
nameElement.focus();
let titleElement = document.getElementById("title");

GetWindowStyle().then((result) => {
    document.body.style.backgroundColor = `rgb(${result.bg.Red}, ${result.bg.Green}, ${result.bg.Blue})`;
})

GetParameters().then((result) => {
    titleElement.innerText = result['title']
})

//document.getElementById('logo').src = logo;



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
                //resultElement.innerText = result;
            })
            .catch((err) => {
                console.error(err);
            });
    } catch (err) {
        console.error(err);
    }
};
