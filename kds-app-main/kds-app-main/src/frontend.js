'use strict';
const path = require('path');
const fs = require('fs');
const ipc = require('electron').ipcRenderer;

let frontend = {
    id: 'frontend',
    name: 'frontend',
    serviceinits: false,
    _path: path,
    _fs: fs,
    _ipc: ipc,
    test: () => {
        console.log('test');
    },
    getConfigPath: () => {
        return frontend._path.join(__dirname,'config','kdsconfig.json');
    },
    loadConfig: () => {
        try{                       
            const rawdata = frontend._fs.readFileSync(frontend.getConfigPath());
            return JSON.parse(rawdata);
        }catch(err){
            console.log(err);            
            return false;
        }
    },
    callBacks: {
        service_log_reply_callback: (event, arg) => {
            console.log('cat');
        },
    },
    initIPCallback: () => {
        if (frontend.serviceinits === false) {
            frontend._ipc.on('service-log-reply', (event, arg) => {
                frontend.callBacks.service_log_reply_callback(event,arg);
            });
        }
    },
    serviceStatus: () => {
        frontend._ipc.send('service-log', 'all')
        frontend._ipc.send('service-status', 'all')
    }
};

module.exports = frontend;
