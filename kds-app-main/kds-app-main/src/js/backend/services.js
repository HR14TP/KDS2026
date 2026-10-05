'use-strict';
const child_process = require('child_process');
const generic = require(__dirname + '/../generic.js');
const path = require('path');
const fs = require('fs');
const kill  = require('tree-kill');
const https = require('https');
const find = require('find-process');
const { BrowserWindow } = require('electron');

var sudo = require('sudo-prompt');
const backend = require('./backend');

const services = {
    name: 'services',
    id: 'services',
    args: [],
    home: '',
    options: {
        spawn: true,
        cwd: __dirname + '/../services'
    },
    updateInProgress: false,
    maintenance: (userDataPath) => {
        const servicesPath = path.join(userDataPath,'services');
        const dbPath = path.join(userDataPath,'db');
        const configPath = path.join(userDataPath,'config');
        const filesPath = path.join(userDataPath,'files');
        const donePath = path.join(filesPath,'done');
        const errorPath = path.join(filesPath,'error');
        if (!fs.existsSync(servicesPath)){
            fs.mkdirSync(servicesPath);
        }
        if (!fs.existsSync(dbPath)){
            fs.mkdirSync(dbPath);
        }
        if (!fs.existsSync(filesPath)){
            fs.mkdirSync(filesPath);
        }
        if (!fs.existsSync(donePath)){
            fs.mkdirSync(donePath);
        }
        if (!fs.existsSync(errorPath)){
            fs.mkdirSync(errorPath);
        }
        if (!fs.existsSync(configPath)){
            fs.mkdirSync(configPath);
            services.downloadConfig();
        }
        if (!fs.existsSync(services.kdsconfig.file)){
            services.downloadConfig();
        }
    },
    init: (userDataPath) => {        
        const servicesPath = path.join(userDataPath,'services');
        const dbPath = path.join(userDataPath,'db');
        const configPath = path.join(userDataPath,'config');
        services.kdsconfig.file = path.join(configPath,'kdsconfig.json');
        services.kdshwid.file = path.join(configPath,'hwid.json');
        services.maintenance(userDataPath);

        const app_args = ['-base=' + userDataPath];
        services.kdstcpserver.args = ['-base=' + userDataPath];
        services.kdstcpserver.file = path.join(servicesPath, generic.getOSname(services.kdstcpserver.servicename));

        services.kdspoller.args = app_args;
        services.kdspoller.file = path.join(servicesPath, generic.getOSname(services.kdspoller.servicename));

        services.kdsweb.args = app_args;
        services.kdsweb.file = path.join(servicesPath, generic.getOSname(services.kdsweb.servicename));

        services.kdsnats.file = path.join(servicesPath, generic.getOSname(services.kdsnats.servicename));
        services.kdsdb.file = path.join(dbPath, 'kds.db');

        services.kdsautostart.file = path.join(dbPath, 'autostart');
    },
    kdstcpserver: {
        'args': null,
        'file' : null,
        'servicename' : 'kdstcpserver',
        'log' : [],
        'status' : false,
        'command' : null,
        'killed' : false,
        'state' : 'red'
    },
    kdspoller: {
        'args': null,
        'file' : null,
        'servicename' : 'kdspoller',
        'log' : [],
        'status' : false,
        'command' : null,
        'killed' : false,
        'state' : 'red'
    },
    kdsweb: {
        'args': null,
        'file' : null,
        'servicename' : 'kdswebserver',
        'log' : [],
        'status' : false,
        'command' : null,
        'killed' : false,
        'state' : 'red'
    },
    kdsnats: {
        'args': [],
        'file' : null,
        'servicename' : 'kdsnats',
        'log' : [],
        'status' : false,
        'command' : null,
        'state' : 'red'
    },
    kdsdb: {
        'file' : null,
        'servicename' : 'kds.db',
    },
    kdsconfig: {
        'file' : null,
        'servicename' : 'kdsconfig.json',
    },
    kdsautostart: {
        'file' : null,
        'servicename' : 'autostart',
    },
    kdshwid: {
        'file' : null,
        'servicename' : 'hwid.json',
    },
    checkService: (serviceName) => {
        try{
            const localService = services.getService(serviceName);
            return fs.existsSync(localService.file);
        } catch(err) {
            console.log(err);
            return false;
        }
    },
    getService: (serviceName) => {
        switch(serviceName) {
            case 'kdstcpserver':
                return services.kdstcpserver;
            case 'kdspoller':
                return services.kdspoller;
            case 'kdsweb':
                return services.kdsweb;
            case 'kdsnats':
                return services.kdsnats;
            case 'kdsdb':
                return services.kdsdb;
            case 'kdsautostart':
                return services.kdsautostart;
            default:
                break;
        }
    },
    downloadAutoStart: (cb) => {
        //https://kdsdailybackup.s3.eu-west-1.amazonaws.com/releases/config/autostart-new
        const file = fs.createWriteStream(services.kdsautostart.file);
        console.log(file);
        console.log(services.kdsautostart.file);
        const fname = services.getServiceDownloadLink('kdsautostart');  
        console.log(fname);      
        setTimeout(function(){
            const request = https.get(fname, function(response) {
                response.pipe(file);
                file.on('finish', function() {
                    file.close();
                    console.log("STEP2"); 
                    cb();
                });
            });
            request.setTimeout(180000, function () {
                request.destroy();
            });
        },3000);
    },
    addAutostart: () => {
        services.downloadAutoStart(() => {
            var options = {
                name: 'KDS Tabletpos',
            };
            sudo.exec("/bin/cp -rf " + services.kdsautostart.file + " /etc/xdg/lxsession/LXDE-pi/autostart", options,
                function(error, stdout, stderr) {
                    if (error) throw error;
                    console.log('stdout: ' + stdout);
                }
            );
        });        
    },
    saferunService: (serviceName) => {
        services.stopService(serviceName , services.runService(serviceName));
    },
    runServicePromise: (serviceName) => {
        if (services.checkService(serviceName)){            
            const localService = services.getService(serviceName);  
            if (localService.state === "red") {   
                localService.state = "orange";       
                console.log("RUNNING SERVICE " + serviceName + " : " + localService.file);
                localService.command = child_process.spawn(localService.file, localService.args, localService.options);
                localService.status = true;
                localService.killed = false;
                //console.log(localService.command.pid);
                localService.command.stdout.on('data', (data) => {
                    console.log(`${localService.command.pid} ${data}`);
                    localService.log.push(`stdout: ${data}`);
                    localService.status = true;
                });
                localService.command.stderr.on('data', (data) => {
                    console.log(`${localService.command.pid} ${data}`);
                    localService.log.push(`stdout: ${data}`);
                    localService.status = true;
                });
                localService.command.on('close', (code) => {
                    localService.log.push(`child process exited with code ${code}`);
                    localService.status = false;
                    if (localService.killed === false){
                        setTimeout(function() {
                            localService.log.push('Restarting Service Due To Failure');
                            services.runService(serviceName);
                        }, 10000);
                    }
                });
                return new Promise(function(resolve , reject) {
                    setTimeout(function() {
                        //5 seconds to start service should be enough
                        const localService = services.getService(serviceName);
                        console.log(localService.command.pid);
                        if (localService.command.pid){
                            localService.state = "green"; 
                            console.log("SUCCESS");
                            resolve("DONE");
                        } else {
                            localService.state = "red"; 
                            reject(Error("PID INVALID"));
                        }
                    }, 3000);
                });
            } else {
                return new Promise(function(resolve , reject) {
                    console.log("Invalid State To Start An App");
                    resolve("DONE");
                });
            }
        } else {
            return new Promise(function(resolve , reject) {
                console.log("Service Not Found");
                resolve("DONE");
            });
        }
    },
    runService: (serviceName) => {
        if (services.checkService(serviceName)){            
            const localService = services.getService(serviceName);            
            console.log("RUNNING KDS SERVICE " + serviceName + " : " + localService.file);
            console.log("RUNNING KDS ARGS " + serviceName + " : " + localService.args);
            localService.command = child_process.spawn(localService.file, localService.args, localService.options);
            localService.status = true;
            localService.killed = false;
            //console.log(localService.command.pid);
            localService.command.stdout.on('data', (data) => {
                console.log(`${localService.command.pid} ${data}`);
                localService.log.push(`stdout: ${data}`);
                localService.status = true;
            });
            localService.command.stderr.on('data', (data) => {
                console.log(`${localService.command.pid} ${data}`);
                localService.log.push(`stdout: ${data}`);
                localService.status = true;
            });
            localService.command.on('close', (code) => {
                localService.log.push(`child process exited with code ${code}`);
                localService.status = false;
                if (localService.killed === false){
                    setTimeout(function() {
                        localService.log.push('Restarting Service Due To Failure');
                        services.runService(serviceName);
                    }, 10000);
                }
            });
        }
    },
    getServiceDownloadLink: (serviceName) => {
        switch(serviceName) {
            case 'kdstcpserver':
                return "https://kdsdailybackup.s3-eu-west-1.amazonaws.com/releases/tcpserver/files/kdstcpserver-" + generic.getArchFilename();
            case 'kdspoller':
                return "https://kdsdailybackup.s3-eu-west-1.amazonaws.com/releases/poller/files/kdspoller-" + generic.getArchFilename();
            case 'kdsweb':
                return "https://kdsdailybackup.s3-eu-west-1.amazonaws.com/releases/webserver/files/kdswebserver-" + generic.getArchFilename();
            case 'kdsnats':
                return "https://kdsdailybackup.s3.eu-west-1.amazonaws.com/releases/nats/files/nats-server-" + generic.getNatsFilename();
            case 'kdsdb':
                return "https://kdsdailybackup.s3-eu-west-1.amazonaws.com/releases/kdsdatabase/kds.db";
            case 'kdsconfig':
                return "https://kdsdailybackup.s3-eu-west-1.amazonaws.com/releases/config/template.json";
            case 'kdsautostart':
                return "https://kdsdailybackup.s3.eu-west-1.amazonaws.com/releases/config/autostart-new";
            default:
                break;
        }
    },
    stopService: (serviceName , cb = null) => {
        const localService = services.getService(serviceName);
        localService.status = false;
        localService.killed = true;
        // A service that was never started this run (e.g. kdspoller on an
        // Order Display Screen terminal - see index.js checkPoller) has no
        // .command to kill. Without this guard, .command.pid throws here,
        // and since this runs inside the service-restart IPC handler's
        // setTimeout chain, the throw silently aborts that chain before it
        // ever sends service-restart-reply - leaving the Config page's save
        // spinner stuck forever.
        if (!localService.command) {
            if (cb !== null) {
                cb();
            }
            return;
        }
        if(cb !== null){
            kill(localService.command.pid , function(error) {
                console.log(error);
                localService.log = [];
                cb();
            });
        } else {
            kill(localService.command.pid);
            localService.log = [];
        }
    },
    /* Download Config File From S3 */
    downloadConfig: () => {        
        const file = fs.createWriteStream(services.kdsconfig.file);
        const fname = services.getServiceDownloadLink('kdsconfig');        
        setTimeout(function(){
            const request = https.get(fname, function(response) {
                response.pipe(file);
                file.on('finish', function() {
                    file.close();
                });
            });
            request.setTimeout(180000, function () {
                request.destroy();
            });
        },3000);
    },
    /* Write Config JSON To FileSystem */
    setConfig: (config , hwid = null , serverpointer = null) => {
        let conf = JSON.parse(config);
        try{
            // Write to a temp file in the same directory, then rename over
            // the real path - a plain writeFileSync on the live path left a
            // truncated/corrupt kdsconfig.json behind when this machine lost
            // power mid-write, and kds-webserver-main's own LoadConfiguration
            // silently swallows that JSON decode error and boots with a
            // blank config instead of erroring - wiping out real settings
            // (e.g. duplicate.peers) with no indication anything went wrong.
            // A rename() is atomic at the filesystem level even across a
            // crash: any reader (or a future boot) only ever sees the old
            // complete file or the new complete file, never a partial one.
            // Same reasoning as seed_binary's atomic swap in
            // seed-custom-binaries.sh.
            const dir = path.dirname(services.kdsconfig.file);
            const tmp = path.join(dir, '.' + path.basename(services.kdsconfig.file) + '.' + process.pid + '.tmp');
            fs.writeFileSync(tmp, JSON.stringify(conf, null, 4));
            fs.renameSync(tmp, services.kdsconfig.file);
            if (hwid !== null && serverpointer !== null){
                let msg = {
                    terminal_id: 0,
                    hwid: hwid,
                    message: config,
                    type: "CONFIGCHANGED"
                }
                console.log(msg);
                serverpointer.doPublish("kds.messages", msg);
            }
            console.log("JSON saved to " + services.kdsconfig.file);
        }catch(err){
            console.log(err);
        }        
    },
    /* Get Config JSON From FileSystem */
    getConfig: () => {        
        const rawdata = fs.readFileSync(services.kdsconfig.file);
        return JSON.parse(rawdata);
    },
    downloadServiceUI: (servicename , childWindow, cb = null) => {
        services.updateService(servicename , false , () => {
            if (cb !== null){
                cb();
            }            
        } , (total , rec) => {   
            //console.log(total);
            //console.log(rec);                                 
            let m = {
                service: servicename,
                total: Math.round((rec/total) * 100),                                    
            };
            try{
                //console.log("sending progress");
                childWindow.send('download-progress' , JSON.stringify(m));                                
            }catch(er){
                console.log(er);
            }
        }); 
    },
    updateAllServices: (mainwindow) => {        
        const child = new BrowserWindow({ parent: mainwindow, frame: false, modal: true, show: false, webPreferences: {
            nodeIntegration: true,
            webSecurity: true,
            contextIsolation: false,           
        }});
        child.loadFile(path.join(__dirname, '../../downloads.html'))
        child.once('ready-to-show', () => {
            child.show();
        });
        services.stopService('kdsweb' , () => {
            //Webserver Stopped
            console.log('kdsweb STOPPED');
            services.stopService('kdspoller' , () => {                
                //KDS Poller Stopped
                console.log('kdspoller STOPPED');
                services.stopService('kdstcpserver' , () => {
                    //KDS TCP Server Stopped
                    console.log('kdstcpserver STOPPED');
                    services.downloadServiceUI('kdsweb' , child , () => {
                        services.downloadServiceUI('kdstcpserver' , child , () => {
                            services.downloadServiceUI('kdspoller' , child , () => {                                
                                services.runServicePromise('kdstcpserver').then( (done) => {
                                    services.runServicePromise('kdspoller').then( (done) => {
                                        services.runServicePromise('kdsweb').then( (done) => {
                                            try{child.close();}catch(e){console.log(e);}
                                        }).catch( (e) => {
                                            console.log(e);
                                        });
                                    }).catch( (e) => {
                                        console.log(e);
                                    });
                                }).catch( (e) => {
                                    console.log(e);
                                });             
                            });
                        });
                    });                    
                });
            });
        });
    },
    updateService: (serviceName, install = false , cb = null , progress = null) => {
        if (services.updateInProgress === false){
            services.updateInProgress = true;
            const localService = services.getService(serviceName);
            if (!install)
                services.stopService(serviceName);
            setTimeout(function(){
                let receivedBytes = 0
                //console.log("Downloading :" + localService.file);
                const file = fs.createWriteStream(localService.file);
                const fname = services.getServiceDownloadLink(serviceName);
                //console.log("From : " + fname);
                //const totalBytes = 0;
                const request = https.get(fname, function(response) {
                    const totalBytes = response.headers['content-length'];
                    //console.log("=====");
                    //console.log(totalBytes);
                    response.pipe(file);
                    response.on('data', (chunk) => {
                        receivedBytes += chunk.length;                    
                        if (progress !== null) {
                            progress(totalBytes , receivedBytes);
                        }
                    });
                    file.on('finish', function() {
                        file.close();
                        setTimeout(function() {
                            file.close();
                            services.updateInProgress = false;
                            if (serviceName !== 'kdsdb') {
                                fs.chmod(localService.file, 0o755, cb);
                            } else {
                                if (cb !== null){
                                    cb();
                                }                            
                            }
                        }, 5000);
                    });                
                });            
                request.setTimeout(180000, function () {
                    request.destroy();
                });
            }, 5000);
        }
    },
    closeService: (serviceName, cb) => {
        const localService = services.getService(serviceName);
        find('name', localService.servicename, true).then(function (list) {
            if (list.length > 0) {
                list.forEach((l) => {
                    kill(l.pid , cb);
                });
            } else {
                cb();
            }
        });
    }
};

module.exports = services;
