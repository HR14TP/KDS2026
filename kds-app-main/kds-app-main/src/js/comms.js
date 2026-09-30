'use strict';

const electron = require('electron')
const ipc = electron.ipcRenderer

const kdscom = {
	name: 'kdscom',
	id: 'kdscom'	
};

function updateservice() {
    ipc.send('service-update', 'all')    
};

function querykdstcpserver() {
    ipc.send('service-log', 'kdstcpserver')
    ipc.send('service-status', 'kdstcpserver')
};
function queryservices() {
    ipc.send('service-log', 'all')
    ipc.send('service-status', 'all')
};
function restartservices() {
    ipc.send('service-restart', 'all')    
};
function autostart() {
    ipc.send('service-autostart', 'all')
};
ipc.on('service-restart-reply', (event, arg) => {
    const message = `Asynchronous message reply: ${arg}`
    //console.log(message);
});
ipc.on('service-log-reply', (event, arg) => {
    //console.log(arg);    
    document.getElementById('kdstcplog').innerHTML = '';
    document.getElementById('kdspollerlog').innerHTML = '';
    document.getElementById('kdsnatslog').innerHTML = '';
    document.getElementById('kdsweblog').innerHTML = '';
    let logs = JSON.parse(arg);  
    if (logs.kdstcpserver.length > 0) {
        for (let i = 0; i < logs.kdstcpserver.length; i++) {
            var node = document.createElement("LI");
            var textnode = document.createTextNode(logs.kdstcpserver[i]);
            node.appendChild(textnode);
            document.getElementById('kdstcplog').appendChild(node);
        }
    };
    if (logs.kdspoller.length > 0) {
        for (let i = 0; i < logs.kdspoller.length; i++) {
            var node = document.createElement("LI");
            var textnode = document.createTextNode(logs.kdspoller[i]);
            node.appendChild(textnode);
            document.getElementById('kdspollerlog').appendChild(node);
        }
    };
    if (logs.kdsnats.length > 0) {
        for (let i = 0; i < logs.kdsnats.length; i++) {
            var node = document.createElement("LI");
            var textnode = document.createTextNode(logs.kdsnats[i]);
            node.appendChild(textnode);
            document.getElementById('kdsnatslog').appendChild(node);
        }
    };
    if (logs.kdsweb.length > 0) {
        for (let i = 0; i < logs.kdsweb.length; i++) {
            var node = document.createElement("LI");
            var textnode = document.createTextNode(logs.kdsweb[i]);
            node.appendChild(textnode);
            document.getElementById('kdsweblog').appendChild(node);
        }
    };
});
// Sets both the label and the status-running/status-stopped class the
// admin page's status-pill CSS keys off of, so the badge is actually
// color-coded (green/red) rather than just differently-worded gray text.
function setServiceStatus(elementId, isRunning) {
    let el = document.getElementById(elementId);
    el.classList.remove('status-running', 'status-stopped');
    if (isRunning === true) {
        el.value = "RUNNING";
        el.classList.add('status-running');
    } else if (isRunning === false) {
        el.value = "STOPPED";
        el.classList.add('status-stopped');
    }
}
ipc.on('service-status-reply', (event, arg) => {
    let status = JSON.parse(arg);
    setServiceStatus('kdstcpstatus', status.kdstcpserver);
    setServiceStatus('kdspollerstatus', status.kdspoller);
    setServiceStatus('kdsnatsstatus', status.kdsnats);
    setServiceStatus('kdswebstatus', status.kdsweb);
});
