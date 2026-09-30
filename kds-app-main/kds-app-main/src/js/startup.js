'use strict';

const fs = require('fs');
const ipc = require('electron').ipcRenderer;
const frontend = require('./frontend.js');

let configFailed = false;
let kdsnatsFailed = false;
let kdswebFailed = false;
let kdspollFailed = false;
let kdstcpFailed = false;

function startupCheck() {
    frontend.callBacks.service_log_reply_callback = (event,arg) => {                
        console.log(event);
        console.log(arg);
    }; 
    frontend.initIPCallback();
    frontend.serviceStatus();

    setTimeout(function(){        
        checkConfig();
        checkServices();
    }, 3000);        
}
function checkConfig() {        
    try {
        if (fs.existsSync(frontend.getConfigPath())) {
            configFailed = false;            
        } else {
            configFailed = true;            
        }
    } catch(err) {
        configFailed = true;        
    }
}
function checkServices(){    
    ipc.send('service-status', 'all');
}
ipc.on('service-status-reply', (event, arg) => {    
    let status = JSON.parse(arg);    
    if (status.kdsnats === false) {
        kdsnatsFailed = true;
    } 
    if (status.kdsweb === false) {
        kdswebFailed = true;        
    }    
    if (status.kdstcpserver === false) {
        kdstcpFailed = true;      
    } 
    if (status.kdspoller === false) {
        kdspollFailed = true;     
    }    
    if (!kdsnatsFailed && !kdspollFailed && !kdstcpFailed && !kdswebFailed && !configFailed) {
        window.location.href = "index.html";        
    } else {
        document.getElementById("statusfield").innerHTML = "KDS Error - Redirecting to Admin Config...<br />";
        setTimeout(function(){
            window.location.href = "admin.html";
        }, 3000);
    }
});
