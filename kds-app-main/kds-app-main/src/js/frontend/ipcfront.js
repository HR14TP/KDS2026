'use strict';
const ipc = require('electron').ipcRenderer;

let ipcFront = {
	name: 'ipcFront',
	id: 'ipcFront',
    downloadPage: {}
};

ipcFront.downloadPage.init = function(){
    //alert('INIT');
    ipc.on('download-progress' , (event, arg) => {
        //console.log(arg);
        try{            
            let m = JSON.parse(arg);            
            document.getElementById(m.service + '_download').innerHTML = m.total + " %";
            document.getElementById(m.service + '_download').style.width = m.total + "%";
        }catch(e) {
            console.log(e)
        }
    });            
};

module.exports = ipcFront;