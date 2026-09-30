'use-strict';

class IPC_Client {
    ipc_main = null;
    ipc_window = null;
    constructor(ipc , ipc_window) {
        this.ipc_main = ipc; 
        this.ipc_window = ipc_window;       
    };
    register(channel , callback) {
        this.ipc_main.on(channel, (event, ...args) => callback(event, ...args));
    };
    send(channel, data) {
        console.log(this.ipc_window);
        this.ipc_window.send(channel, data);
    };
};

module.exports = { IPC_Client };