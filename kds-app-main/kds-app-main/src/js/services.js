'use strict';
var child_process = require('child_process');
const electron = require('electron')
const ipc = electron.ipcRenderer
const https = require('https');
var kill  = require('tree-kill');
const fs = require('fs');
const theapp = require('electron').app
const find = require('find-process');
const si = require('systeminformation');
let platform = "linux";
let arch = "x64";
const path = require('path');


function getosfilename(filename){
    if (platform === "linux") {
        return filename;
    } else if (platform === "win32"){
        return filename + ".exe";
    } else if (platform === "win64"){
        return filename + ".exe";
    } else if (platform === "darwin"){
        return filename;
    } else {
        return filename;
    }                    
};
function getarchfilename(){
    if (platform === "win32" || platform === "win64") {
        return "windows";
    } else if (arch === "x64" && platform === "linux") {
        return "linux";
    } else if (arch === "x64" && platform === "darwin") {
        return "osx";
    } else {
        return "arm7";
    }                    
};
function getnatsfilename(){
    if (platform === "win32" || platform === "win64") {
        return "windows.exe";
    } else if (arch === "x64" && platform === "linux") {
        return "linux";
    } else if (arch === "x64" && platform === "darwin") {
        return "osx";
    } else if (arch === "arm" && platform === "linux") {
        return "arm7";
    } else {
        return "arm7";
    }                    
};

const kdsservices = {
	name: 'kdsservices',
	id: 'kdsservices',
    args: [],
    home: '',
    options: {
        spawn: true,
        cwd: __dirname + '/../services' 
    },
    makeWithTheInfo(cb){
        si.osInfo().then( (data) => {
            //console.log(data);
            platform = data.platform;
            arch = data.arch;
            kdsservices.kdsnats.file = getosfilename(kdsservices.kdsnats.file);
            kdsservices.kdstcpserver.file = getosfilename(kdsservices.kdstcpserver.file);
            kdsservices.kdsweb.file = getosfilename(kdsservices.kdsweb.file);
            kdsservices.kdspoller.file = getosfilename(kdsservices.kdspoller.file);
            cb();
        });
    },
    kdstcpserver: {
        'args':['-base='+path.join(theapp.getAppPath(),'src')],
        'file' : path.join(theapp.getAppPath(),'src','services','kdstcpserver'),
        'servicename' : 'kdstcpserver',
        'log' : [],
        'status' : false,
        'command' : null,
        'run': function(){}
    },
    kdspoller: {
        'args':['-base='+path.join(theapp.getAppPath(),'src')],
        'file' : path.join(theapp.getAppPath(),'src','services','kdspoller'),
        'servicename' : 'kdspoller',
        'log' : [],
        'status' : false,
        'command' : null,
        'run': function(){}
    },
    kdsnats: {
        'file' : path.join(theapp.getAppPath(),'src','services','kdsnats'),
        'servicename' : 'kdsnats',
        'log' : [],
        'status' : false,
        'command' : null,
        'run': function(){}
    },
    kdsweb: {
        'args':['-base='+path.join(theapp.getAppPath(),'src')],
        'file' : path.join(theapp.getAppPath(),'src','services','kdswebserver'),
        'servicename' : 'kdswebserver',
        'log' : [],
        'status' : false,
        'command' : null,
        'run': function(){}
    },
    kdsdb: {
        'file' : path.join(theapp.getAppPath(),'src','db','kds.db'),
    }
};

kdsservices.setpath = function(path) {
    kdsservices.home = path; 
    kdsservices.options = {
        spawn: true,
        cwd: path + '/services' 
    };
};

//************************KDS WEB SERVER ***********************/
kdsservices.kdsweb.run = function() { 
    console.log("RUNNING WEB SERVER : " + kdsservices.kdsweb.file);                       
    kdsservices.kdsweb.command = child_process.spawn(kdsservices.kdsweb.file, kdsservices.kdsweb.args, kdsservices.options);
    kdsservices.kdsweb.status = true;    
    kdsservices.kdsweb.command.stdout.on('data', (data) => {
        console.log(`${kdsservices.kdsweb.command.pid} ${data}`);
        kdsservices.kdsweb.log.push(`stdout: ${data}`);
        kdsservices.kdsweb.status = true;
    });
    kdsservices.kdsweb.command.stderr.on('data', (data) => {
        console.log(`${kdsservices.kdsweb.command.pid} ${data}`);
        kdsservices.kdsweb.log.push(`stdout: ${data}`);
        kdsservices.kdsweb.status = true;
    });    
    kdsservices.kdsweb.command.on('close', (code) => {        
        kdsservices.kdsweb.log.push(`child process exited with code ${code}`);
        kdsservices.kdsweb.status = false;        
    });
};
kdsservices.kdsweb.term = function() {
    kdsservices.kdsweb.status = false;
    kill(kdsservices.kdsweb.command.pid);
};
kdsservices.kdsweb.checkrunning = function(cb){
    find('name', kdsservices.kdsweb.servicename, true).then(function (list) {        
        if (list.length > 0) {
            list.forEach((l) => {
                kill(l.pid , cb);
            });      
        } else {
            cb();
        }      
    });
};
kdsservices.kdsweb.check = () => {
    let ret = false;
    try {
        if (fs.existsSync(kdsservices.kdsweb.file)) {
            ret = true;
        } else {
            ret = false;
        }
    } catch(err) {
        ret = false;        
    }
    return ret;
};
//************************ END KDS WEB SERVER ***********************/


//************************KDS NATS SERVER ***********************/
kdsservices.kdsnats.run = function() {   
    console.log("RUNNING NATS"); 
    console.log(getosfilename(kdsservices.kdsnats.file));                 
    kdsservices.kdsnats.command = child_process.spawn(kdsservices.kdsnats.file, kdsservices.args, kdsservices.options);
    kdsservices.kdsnats.status = true;    
    kdsservices.kdsnats.command.stdout.on('data', (data) => {
        console.log(`${kdsservices.kdsnats.command.pid} ${data}`);
        kdsservices.kdsnats.log.push(`stdout: ${data}`);
        kdsservices.kdsnats.status = true;
    });
    kdsservices.kdsnats.command.stderr.on('data', (data) => {
        console.log(`${kdsservices.kdsnats.command.pid} ${data}`);
        kdsservices.kdsnats.log.push(`stdout: ${data}`);
        kdsservices.kdsnats.status = true;
    });    
    kdsservices.kdsnats.command.on('close', (code) => {        
        kdsservices.kdsnats.log.push(`child process exited with code ${code}`);
        kdsservices.kdsnats.status = false;        
    });
};
kdsservices.kdsnats.term = function() {
    kdsservices.kdsnats.status = false;
    kill(kdsservices.kdsnats.command.pid);    
};
kdsservices.kdsnats.checkrunning = function(cb){
    find('name', kdsservices.kdsnats.servicename, true).then(function (list) {        
        if (list.length > 0) {
            list.forEach((l) => {
                kill(l.pid , cb);
            });      
        } else {
            cb();
        }
    });
};
kdsservices.kdsnats.check = () => {
    let ret = false;
    try {
        if (fs.existsSync(kdsservices.kdsnats.file)) {
            ret = true;
        } else {
            ret = false;
        }
    } catch(err) {
        ret = false;        
    }
    return ret;
};
//************************END KDS NATS SERVER ***********************/

kdsservices.kdspoller.run = function() {     
    console.log("RUNNING : " + kdsservices.kdspoller.file);                 
    kdsservices.kdspoller.command = child_process.spawn(kdsservices.kdspoller.file, kdsservices.kdspoller.args, kdsservices.options);
    kdsservices.kdspoller.status = true;    
    kdsservices.kdspoller.command.stdout.on('data', (data) => {
        console.log(`${kdsservices.kdspoller.command.pid} ${data}`);
        kdsservices.kdspoller.log.push(`stdout: ${data}`);
        kdsservices.kdspoller.status = true;
    });
    kdsservices.kdspoller.command.stderr.on('data', (data) => {
        console.log(`${kdsservices.kdspoller.command.pid} ${data}`);
        kdsservices.kdspoller.log.push(`stdout: ${data}`);
        kdsservices.kdspoller.status = true;
    });    
    kdsservices.kdspoller.command.on('close', (code) => {        
        kdsservices.kdspoller.log.push(`child process exited with code ${code}`);
        kdsservices.kdspoller.status = false;        
    });
};
kdsservices.kdspoller.term = function() {
    kdsservices.kdspoller.status = false;
    kdsservices.kdspoller.command.kill();
};

kdsservices.kdspoller.checkrunning = function(cb){
    find('name', kdsservices.kdspoller.servicename, true).then(function (list) {
        //console.log(list);
        if (list.length > 0) {
            list.forEach((l) => {
                kill(l.pid , cb);
            });      
        } else {
            cb();
        }      
    });
};
kdsservices.kdspoller.check = () => {
    let ret = false;
    try {
        if (fs.existsSync(kdsservices.kdspoller.file)) {
            ret = true;
        } else {
            ret = false;
        }
    } catch(err) {
        ret = false;        
    }
    return ret;
};


//************************KDS TCP SERVER ***********************/
kdsservices.kdstcpserver.run = function() {    
    console.log("RUNNING TCP SERVER"); 
    console.log(kdsservices.kdstcpserver.file);
    kdsservices.kdstcpserver.command = child_process.spawn(kdsservices.kdstcpserver.file, kdsservices.kdstcpserver.args, kdsservices.options);
    kdsservices.kdstcpserver.status = true;
    console.log(kdsservices.kdstcpserver.command.pid);
    kdsservices.kdstcpserver.command.stdout.on('data', (data) => {
        console.log(`${kdsservices.kdstcpserver.command.pid} ${data}`);
        kdsservices.kdstcpserver.log.push(`stdout: ${data}`);
        kdsservices.kdstcpserver.status = true;
    });
    kdsservices.kdstcpserver.command.stderr.on('data', (data) => {
        console.log(`${kdsservices.kdstcpserver.command.pid} ${data}`);
        kdsservices.kdstcpserver.log.push(`stdout: ${data}`);
        kdsservices.kdstcpserver.status = true;
    });
    kdsservices.kdstcpserver.command.on('close', (code) => {
        kdsservices.kdstcpserver.log.push(`child process exited with code ${code}`);
        kdsservices.kdstcpserver.status = false;        
    });
};
kdsservices.kdstcpserver.term = function() {
    kdsservices.kdstcpserver.status = false;
    kdsservices.kdstcpserver.command.kill();
};
kdsservices.kdstcpserver.check = () => {
    let ret = false;
    try {
        if (fs.existsSync(kdsservices.kdstcpserver.file)) {
            ret = true;
        } else {
            ret = false;
        }
    } catch(err) {
        ret = false;        
    }
    return ret;
};
kdsservices.kdstcpserver.checkrunning = function(cb){
    find('name', kdsservices.kdstcpserver.servicename, true).then(function (list) {
        //console.log(list);
        if (list.length > 0) {
            list.forEach((l) => {
                kill(l.pid , cb);
            });      
        } else {
            cb();
        }
    });
};
//******************END TCP SERVER *****************************/



//************SERVICE CHECKS**************/
kdsservices.kdsdb.check = () => {
    let ret = false;
    try {
        if (fs.existsSync(kdsservices.kdsdb.file)) {
            ret = true;
        } else {
            ret = false;
        }
    } catch(err) {
        ret = false;        
    }
    return ret;
};
//****************END***********************/

//************UPDATE FUNCTIONS**************/
kdsservices.updateservice = (servicename , install = false , cb = null) => {    
    switch (servicename) {
        case 'kdspoller':
            if (!install)
                kdsservices.kdspoller.term();
            setTimeout(function(){ 
                console.log("Downloading :" + kdsservices.kdspoller.file);
                const file = fs.createWriteStream(kdsservices.kdspoller.file);
                const fname = "https://kdsdailybackup.s3-eu-west-1.amazonaws.com/releases/poller/files/kdspoller-"+getarchfilename();
                console.log("From : " + fname);
                const request = https.get(fname, function(response) {
                    response.pipe(file);
                    file.on('finish', function() {
                        file.close();                        
                        setTimeout(function() {
                            file.close();                            
                            fs.chmod(kdsservices.kdspoller.file,0o755,cb); 
                        }, 4000); 
                    });
                });
                request.setTimeout(30000, function () {
                    request.destroy();
                });
            },3000);
            break;
        case 'kdstcpserver':
            if (!install)
                kdsservices.kdstcpserver.term();
            setTimeout(function(){ 
                console.log("Downloading :" + kdsservices.kdstcpserver.file);
                const file = fs.createWriteStream(kdsservices.kdstcpserver.file);
                const fname = "https://kdsdailybackup.s3-eu-west-1.amazonaws.com/releases/tcpserver/files/kdstcpserver-"+getarchfilename()+"-v2";
                console.log("From : " + fname);
                const request = https.get(fname, function(response) {
                    response.pipe(file);
                    file.on('finish', function() {                        
                        file.close();
                        setTimeout(function() {
                            file.close();                            
                            fs.chmod(kdsservices.kdstcpserver.file,0o755,cb); 
                        }, 4000);                        
                    });
                });
                request.setTimeout(30000, function () {                    
                    request.destroy();
                });
            },3000);
            break;
        case 'kdsnats':
            if (!install)
                kdsservices.kdsnats.term();    
            setTimeout(function(){ 
                console.log("Downloading :" + kdsservices.kdsnats.file);
                const file = fs.createWriteStream(kdsservices.kdsnats.file);
                const fname = "https://kdsdailybackup.s3.eu-west-1.amazonaws.com/releases/nats/files/nats-server-"+getnatsfilename();
                console.log("From : " + fname);
                const request = https.get(fname, function(response) {
                    response.pipe(file);
                    file.on('finish', function() {                        
                        file.close(); 
                        console.log("DOWNLOAD FINISHED");                       
                        setTimeout(function() {
                            file.close();                            
                            fs.chmod(kdsservices.kdsnats.file,0o755,cb); 
                        }, 4000);
                                                
                    });
                });
                request.setTimeout(60000, function () {                    
                    request.destroy();
                });
            },3000);      
          break;
        case 'kdsweb':
            if (!install)
                kdsservices.kdsweb.term();
            setTimeout(function(){ 
                console.log("Downloading :" + kdsservices.kdsweb.file);
                const fname = "https://kdsdailybackup.s3-eu-west-1.amazonaws.com/releases/webserver/files/kdswebserver-"+getarchfilename()+"-v2";
                console.log("From : " + fname);
                const file = fs.createWriteStream(kdsservices.kdsweb.file);
                const request = https.get(fname, function(response) {
                    response.pipe(file);
                    file.on('finish', function() {
                        file.close();
                        setTimeout(function() {
                            file.close();                            
                            fs.chmod(kdsservices.kdsweb.file,0o755,cb); 
                        }, 4000);                        
                    });
                });
                request.setTimeout(30000, function () {
                    request.destroy();
                });
            },3000);
            break;
        case 'kdsdb':            
            setTimeout(function(){ 
                console.log("Downloading DB :" + kdsservices.kdsdb.file);
                const fname = "https://kdsdailybackup.s3-eu-west-1.amazonaws.com/releases/kdsdatabase/kds.db";
                console.log("From : " + fname);
                const file = fs.createWriteStream(kdsservices.kdsdb.file);
                const request = https.get(fname, function(response) {
                    response.pipe(file);
                    file.on('finish', function() {
                        file.close();   
                        setTimeout(function() {
                            file.close();                            
                            cb();
                        }, 4000);                                            
                    });
                });                   
                request.setTimeout(30000, function () {
                    request.destroy();
                });
            },3000);
            break;
        default:
          console.log(`Sorry`);
    }
};

//****************END***********************/
module.exports = kdsservices;
