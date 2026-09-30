const { ipcRenderer } = require('electron');
const { LocalEvents , LocalRequests , GlobalEvents } = require(__dirname + '/js/libs/globals/events.js');

window.IPCOUT = {
    send: (channel, data) => {
        ipcRenderer.send(channel, data);
    },
    receive: (channel, func) => {
        ipcRenderer.on(channel, (event, ...args) => func(...args));
    },
    request: (channel, data) => {
        return ipcRenderer.sendSync(channel, data);
    },
    saveconfig: (config) => {
        console.log("Sending : " + LocalRequests.REQUEST_CONFIG_SAVE);
        ipcRenderer.send(LocalRequests.REQUEST_CONFIG_SAVE, config);
    }
};

window.KDSAPI = {
    skipRegistration: () => {
        ipcRenderer.send("registration-skip-for-now", "OK");
    },
    playSound: () => {        
        ipcRenderer.send(LocalEvents.EVENT_PLAY_SOUND, "OK");
    }
};

window.addEventListener('DOMContentLoaded', () => {
    const replaceText = (selector, text) => {
        const element = document.getElementById(selector)
        if (element) element.innerText = text
    }

    for (const type of ['chrome', 'node', 'electron']) {
        replaceText(`${type}-version`, process.versions[type])
    }
});