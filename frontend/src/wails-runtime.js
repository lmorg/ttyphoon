import { Browser, Clipboard, Events, Screens, Window } from '@wailsio/runtime';

export function EventsOn(eventName, callback) {
    return Events.On(eventName, (event) => callback(event.data));
}

export function EventsOff(...eventNames) {
    Events.Off(...eventNames);
}

export function ClipboardSetText(text) {
    return Clipboard.SetText(text);
}

export function BrowserOpenURL(url) {
    return Browser.OpenURL(url);
}

export function ScreenGetAll() {
    return Promise.all([
        Screens.GetAll(),
        Window.GetScreen().catch(() => null),
    ]).then(([screens, current]) => screens.map((screen) => ({
        x: screen.WorkArea.X,
        y: screen.WorkArea.Y,
        width: screen.WorkArea.Width,
        height: screen.WorkArea.Height,
        isCurrent: screen.ID === current?.ID,
        isPrimary: screen.IsPrimary,
    })));
}

export function WindowGetPosition() {
    return Window.Position();
}

export async function WindowGetSize() {
    const size = await Window.Size();
    return { w: size.width, h: size.height };
}

export function WindowIsMaximised() {
    return Window.IsMaximised();
}

export function WindowMaximise() {
    return Window.Maximise();
}

export function WindowUnmaximise() {
    return Window.UnMaximise();
}

export function WindowSetPosition(x, y) {
    return Window.SetPosition(x, y);
}

export function WindowSetSize(width, height) {
    return Window.SetSize(width, height);
}