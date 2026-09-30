'use strict';
const path = require('path');
const fs = require('fs');
const { app, ipcMain, BrowserWindow } = require('electron');
const services = require('./services.js');
const natslocal = require('../nats-local.js');
const natsServer = require('../nats-server.js');
var XMLHttpRequest = require("xmlhttprequest").XMLHttpRequest;
const { JSONCodec, StringCodec } = require("nats");
const os = require('os');
const ThermalPrinter = require("node-thermal-printer").printer;
const Types = require("node-thermal-printer").types;
const log = require('electron-log');
 
let backend = {
    id: 'backend',
    name: 'backend',
    _jcodec: JSONCodec,
    _strcodec: StringCodec,
    _nlocal: natslocal,
    _nserver: natsServer,
    _hwid: null,
    _natsSubject: null,
    _mainWindow: null,
    _path: path,
    _fs: fs,
    _service: services,
    _ipcMain: ipcMain,
    _validTerminal: false,
    _serverNats: null,
    _serverNatsLocal: null,
    _kdshwid: null,    
    requestHWID: () => {
        try {            
            let id = backend._kdshwid.getHWID().hwid;
            console.log("HWID = " + id);
            return id;
        } catch (err) {
            console.log(err);
            var networkInterfaces = os.networkInterfaces();
            Object.keys(networkInterfaces).forEach((NetworkID, index, obj) => {
                if (NetworkID === "en0" || NetworkID === "eth0") {
                    networkInterfaces[NetworkID].forEach((data) => {
                        if (data.family === "IPv4") {
                            return data.mac;
                        }
                    });
                }
            });
        }
    },
    _communication: {
        natsIncoming: async (sub) => {
            for await (const msg of sub) {
                try {
                    let sc = StringCodec();
                    let obj = JSON.parse(sc.decode(msg.data));
                    //console.log(obj);
                    if (obj.type === "CONFIGUPDATE") {
                        //console.log(obj.message);
                        backend._service.setConfig(obj.message);
                        backend._mainWindow.send('config-updates-remotely', "DONE");
                        let msg = {
                            terminal_id: obj.terminal_id,
                            message: backend._hwid,
                            type: "CONFIGUPDATED"
                        }
                        backend._nserver.doPublish("kds.messages", msg);
                    } else if (obj.type === "TERMINALSTATUS") {
                        let message = {
                            terminal_id: obj.terminal_id,
                            message: "OK",
                            type: "STATUS"
                        };
                        msg.respond(backend._nserver._strcodec.encode(JSON.stringify(message)));
                    } else if (obj.type === "UPDATESERVICE" && backend._service.updateInProgress === false) {
                        backend._service.updateAllServices();
                    } else if (obj.type === "GETHWID") {
                        let message = {
                            terminal_id: obj.terminal_id,
                            message: backend._hwid,
                            type: backend._hwid,
                        };
                        msg.respond(backend._nserver._strcodec.encode(JSON.stringify(message)));
                    } else if (obj.type === "SYNCCONFIG") {
                        var conf = backend._service.getConfig();
                        let message = {
                            terminal_id: obj.terminal_id,
                            message: JSON.stringify(conf),
                            type: "SYNCCONFIG",
                        };
                        msg.respond(backend._nserver._strcodec.encode(JSON.stringify(message)));
                    }
                } catch (err) {
                    console.log(err);
                }
            }
        },
        natsIncomingLocal: async (sub) => {
            log.info('natsIncomingLocal');

            let sc = StringCodec();
            for await (const msg of sub) {
                try {
                    let obj = JSON.parse(sc.decode(msg.data));
                    log.info('natsIncomingLocal OBJECT:', obj);
                    log.info('ORDER TYPE:' , obj.type);
                    
                    if (obj.type === "CLOSEDORDER") {
                        //ORDER CLOSE EVENT
                        let order = JSON.parse(obj.message);
                        var conf = backend._service.getConfig();
                        if (conf.hasOwnProperty('smsconfig')) {
                            if (conf.smsconfig.enabled) {
                                let twilioclient = require('twilio')(conf.smsconfig.twilio.twilio_accountsid, conf.smsconfig.twilio.twilio_authtoken);
                                if (order.pass_status !== "OPEN") {
                                    if (undefined !== conf.smsconfig.twilio.prefix && conf.smsconfig.twilio.prefix !== ''){
                                        if (order.pass_status.charAt(0) == "0") {
                                            //Replacing Number
                                            order.pass_status = order.pass_status.substring(1);
                                            order.pass_status = conf.smsconfig.twilio.prefix + order.pass_status;
                                        }
                                    }
                                    console.log("sending");
                                    twilioclient.messages.create({
                                        body: conf.smsconfig.twilio.twilio_message,
                                        from: conf.smsconfig.twilio.twilio_sendingnumber,
                                        to: order.pass_status
                                    }).then(message => {
                                        console.log(message);
                                        var e = {
                                            message: "SMS has been send"
                                        };
                                        backend._mainWindow.send('toast-message', JSON.stringify(e));
                                        //event.sender.send('service-test-sms-reply', JSON.stringify(message));
                                    }).catch(e => {
                                        console.error('Got an error:', e.code, e.message);
                                        var m = {
                                            message: e.message
                                        };
                                        backend._mainWindow.send('toast-message', JSON.stringify(m));
                                    });
                                }
                            }
                        }
                        //Check for network print
                        var e = {
                            message: "Checking Order"
                        };
                        //backend._mainWindow.send('toast-message', JSON.stringify(e));
                        log.info('Has property -> conf.oncomplete' , conf.hasOwnProperty('oncomplete') ? 'TRUE' : 'FALSE');
                        if (conf.hasOwnProperty('oncomplete')) {
                            //console.log("STEP1");
                            //var e = {
                            //    message: "Order enabled for printing"
                            //};
                            //backend._mainWindow.send('toast-message', JSON.stringify(e));
                            log.info('conf.oncomplete.enablesendprinter:' , ((undefined !== conf.oncomplete.enablesendprinter && conf.oncomplete.enablesendprinter == true) ? "true" : 'false'));
                            log.info('conf.oncomplete.sendprinterip:' , conf.oncomplete.sendprinterip);

                            if (undefined !== conf.oncomplete.enablesendprinter && conf.oncomplete.enablesendprinter == true) {
                                //console.log("STEP2");

                                if (undefined !== conf.oncomplete.sendprinterip && conf.oncomplete.sendprinterip !== '') {
                                    //console.log("STEP3");
                                    backend.getSingleOrder(order.id , (singleorder) => {
                                        log.info('getSingleOrder SingleOrder:' , (singleorder !== null) ? 'Order is not null' : 'Order is Null');
                                        if (singleorder !== null){
                                            log.info('sendOrderToPrinter Call Made: True'); 
                                            //console.log(singleorder);
                                            backend.sendOrderToPrinter(singleorder , conf.oncomplete.sendprinterip);
                                        }
                                    });
                                    
                                }
                            }                        
                        } else {
                            var e = {
                                message: "Order not enabled for printing"
                            };
                            log.info('Order not enabled for printing'); 

                            //backend._mainWindow.send('toast-message', JSON.stringify(e));
                        }
                    }
                    //console.log(obj);
                } catch (err) {
                    console.log(err);
                    log.warn('natsIncomingLocal error', err); 
                }
            }
        },
        requestConfig: async () => {
            backend._nserver.doRequest("kds.requests", { terminal_id: 0, message: backend._hwid, type: "GETCONFIG" }, function (m) {
                try {
                    let sc = StringCodec();
                    //console.log(sc.decode(m.data));
                    let obj = JSON.parse(sc.decode(m.data));
                    if (obj.type == "REQUESTCONFIG") {
                        backend._service.setConfig(obj.message);
                        backend._mainWindow.send('config-updates-remotely', "DONE");
                        let msg = {
                            terminal_id: obj.terminal_id,
                            message: backend._hwid,
                            type: "CONFIGUPDATED"
                        };
                        backend._nserver.doPublish("kds.messages", msg);
                        console.log("CONFIG_UPDATED");
                        log.info('CONFIG_UPDATED'); 
                    }
                } catch (err) {
                    console.log(err);
                    log.warn('requestConfig error', err); 
                }
            });
        }
    },
    sendOrderToPrinter: async(singleorder, ip) => {
        try{
        let printer = new ThermalPrinter({
            type: Types.EPSON,  // 'star' or 'epson'
            interface: 'tcp://' + ip ,
            options: {
              timeout: 5000
            },
            //width: 48,                         // Number of characters in one line - default: 48
            characterSet: 'SLOVENIA',          // Character set - default: SLOVENIA
            removeSpecialCharacters: false,    // Removes special characters - default: false
            lineCharacter: "-",                // Use custom character for drawing lines - default: -
          });
        let isConnected = await printer.isPrinterConnected();
        console.log("Printer connected:", isConnected);
        log.info("Printer connected:", isConnected);
        printer.alignLeft();
        printer.setTextSize(2,1); 
        printer.bold(false);
        printer.println("TABLE:" + singleorder.table.toUpperCase());
        printer.setTextNormal();
        printer.drawLine();
        printer.bold(true);
        printer.println("WAITER: " + singleorder.waiter.toUpperCase());
        printer.println("PROFILE: " + singleorder.profile.toUpperCase());
        printer.println("REFERENCE: " + singleorder.reference.toUpperCase());
        printer.drawLine();
        let screen = "KDS";
        printer.setTextNormal();
        printer.bold(true);
        singleorder.items.forEach((item) => {
            printer.println(item.qty + " X " + item.name.toUpperCase());
            screen = item.screen.name;
            item.subitems.forEach( (subitem) => {
                printer.println(" -> " + subitem.qty + " X " + subitem.name.toUpperCase());
            });            
        });
        printer.drawLine(); 
        printer.bold(true);      
        printer.underline(true);        
        printer.println("ORDER HAS BEEN BUMPED FROM : " + screen.toUpperCase());                
        printer.partialCut();
        //printer.cut();
        printer.beep();
        try {
            await printer.execute();            
            var e = {
                message: "Order Sent To Printer"
            };
            log.info("Order Sent To Printer.", 'Toast Notification');
            backend._mainWindow.send('toast-message', JSON.stringify(e));
        } catch (error) {
            //console.error("Print error:", error);
            var e = {
                message: "Error Sending To Printer : " + error
            };
            log.warn("Error Sending To Printer.", 'Toast Notification');
            log.warn("Error Sending To Printer error:", error);

            backend._mainWindow.send('toast-message', JSON.stringify(e));
        }
        } catch (error) {
            var e = {
                message: "Error Sending To Printer : " + error
            };
            log.warn("Error Sending To Printer.", 'Toast Notification');
            log.warn("Error Sending To Printer error:", error);

            backend._mainWindow.send('toast-message', JSON.stringify(e));
        }
    },
    loadConfig: () => {
        try {
            const rawdata = backend._fs.readFileSync(backend._path.join(app.getAppPath(), 'src', 'config', 'kdsconfig.json'));
            return JSON.parse(rawdata);
        } catch (err) {
            console.log(err);
            alert(err);
            log.warn("loadConfig()  error:", err);
            return false;
        }
    },
    loadConfigPath: (path) => {
        try {
            const rawdata = backend._fs.readFileSync(path);
            return JSON.parse(rawdata);
        } catch (err) {
            console.log(err);
            alert(err);
            log.warn("loadConfigPath()  error:", err);
            return false;
        }
    },
    setupNats: async () => {
        backend._nserver.backendInit().then(done => {
            if (done){
                backend._nserver.subScribe(backend._natsSubject, backend._communication.natsIncoming);
                // requestConfig() used to fire here on every single boot/
                // reconnect, pulling whatever config the central server has
                // cached for this terminal's hwid and overwriting the local
                // file with it unconditionally - no merge, and nothing ever
                // pushes local changes back up to that cache. Any setting
                // changed locally (passmode, sortbyprodstate, etc.) only
                // ever lived in the just-saved local file, so the very next
                // boot's requestConfig() stomped it back to whatever stale
                // snapshot the server had. downloadConfig() (see
                // services.js maintenance()) already seeds a real config
                // for a genuinely new terminal, so this had no safe purpose
                // left to serve on every subsequent boot.
            }
        });
        backend._nlocal.backendInit().then(done => {
            if (done) {
                backend._nlocal.subScribe("events", backend._communication.natsIncomingLocal);
            }
        });
    },
    getSingleOrder: (orderid , cb) => {
        var xmlHttp = new XMLHttpRequest();
        xmlHttp.onreadystatechange = function () {
            if (xmlHttp.readyState === 4 && xmlHttp.status === 200) {
                try {
                    let j = JSON.parse(xmlHttp.responseText);
                    console.log(xmlHttp.responseText);
                    cb(j);                  
                } catch (err) {
                    console.log(err);
                    log.warn("getSingleOrder()  error:", err);
                    rcb(null);
                }
            }
        };
        console.log("http://127.0.0.1:2020/getsingledborder/" + orderid);
        xmlHttp.open("GET", "http://127.0.0.1:2020/getsingledborder/" + orderid, true); // true for asynchronous
        xmlHttp.send(null);
        //console.log(xmlHttp.responseText);
    },
    validateTerminal: () => {
        let conf = backend._service.getConfig();
        let id = backend.requestHWID();
        console.log("HWID : " + id);
        backend._hwid = id;
        backend._natsSubject = 'kds.' + id;
        var xmlHttp = new XMLHttpRequest();
        xmlHttp.onreadystatechange = function () {
            if (xmlHttp.readyState === 4 && xmlHttp.status === 200) {
                try {
                    let j = JSON.parse(xmlHttp.responseText);
                    console.log(xmlHttp.responseText);
                    console.log(j);
                    if (j.data.id > 0) {
                        console.log("Registered");
                        backend._validTerminal = true;
                    } else {
                        console.log("Unregistered");
                        backend._validTerminal = false;
                    }
                    backend.setupNats();
                } catch (err) {
                    log.warn('validateTerminal() error', err); 
                    console.log(err);
                }
            }
        };
        xmlHttp.open("GET", "https://admin.tabletpos.co.za:8080/kds/terminal/register?hwid=" + id + "&connection_code=" + conf.client.serial, true); // true for asynchronous
        xmlHttp.send(null);
    },
    initIPC: () => {
        /* SERVICE LOGS */
        backend._ipcMain.on('service-log', (event, arg) => {
            if (arg === 'all') {
                event.sender.send('service-log-reply', JSON.stringify({
                    kdstcpserver: backend._service.getService('kdstcpserver').log,
                    kdspoller: backend._service.getService('kdspoller').log,
                    kdsnats: backend._service.getService('kdsnats').log,
                    kdsweb: backend._service.getService('kdsweb').log,
                }));
            }
        });

        /* SERVICE STATUS */
        backend._ipcMain.on('service-status', (event, arg) => {
            event.sender.send('service-status-reply', JSON.stringify({
                kdstcpserver: backend._service.getService('kdstcpserver').status,
                kdsnats: backend._service.getService('kdsnats').status,
                kdspoller: backend._service.getService('kdspoller').status,
                kdsweb: backend._service.getService('kdsweb').status,
            }));
        });

        /* GET APP CONFIG */
        backend._ipcMain.on('service-config', (event, arg) => {
            console.log('service-config');
            let filejson = backend.loadConfigPath(backend._service.kdsconfig.file);
            if (undefined == filejson.bumpbars.mapping) {
                let temp = JSON.stringify(filejson.bumpbars);
                filejson.bumpbars.mapping = JSON.parse(temp);
            }
            
            //console.log(JSON.stringify(filejson));
            event.sender.send('service-config-reply', JSON.stringify(filejson));
        });

        /* SAVE CONFIG KEYMAPPING */
        backend._ipcMain.on('service-save-keymap', (event, arg) => {
            var config = backend._service.getConfig();
            console.log('service-save-keymap');
            if (!config.hasOwnProperty('bumpbars')) {
                config.bumpbars = {};
            }
            if (!config.bumpbars.hasOwnProperty('mapping')) {
                config.bumpbars.mapping = {};
            }
            let themappings = JSON.parse(arg);
            config.bumpbars.mapping = themappings;
            backend._service.setConfig(JSON.stringify(config));
            event.sender.send('service-config-reply', JSON.stringify(backend.loadConfigPath(backend._service.kdsconfig.file)));
        });

        /* GET KEYMAPS FROM CONFIG */
        backend._ipcMain.on('service-get-keymap', (event, arg) => {
            var config = backend._service.getConfig();
            console.log('service-get-keymap');
            if (arg === 'all') {
                let mapping = {
                    "enabled": false,
                };
                if (config.hasOwnProperty('bumpbars')) {                    
                    if (config.bumpbars.hasOwnProperty('mapping')) {
                        mapping = config.bumpbars.mapping;
                    } else {
                        mapping = config.bumpbars;
                    }
                }
                event.sender.send('service-get-keymap-reply', JSON.stringify(mapping));
            }
        });

        /* GET ODS COMPLETE-LIST STATE FROM CONFIG */
        backend._ipcMain.on('service-get-ods-state', (event, arg) => {
            var config = backend._service.getConfig();
            console.log('service-get-ods-state');
            let state = { "completed": [] };
            if (config.hasOwnProperty('odsstate')) {
                state = config.odsstate;
            }
            event.sender.send('service-get-ods-state-reply', JSON.stringify(state));
        });

        /* SAVE ODS COMPLETE-LIST STATE TO CONFIG */
        backend._ipcMain.on('service-save-ods-state', (event, arg) => {
            var config = backend._service.getConfig();
            console.log('service-save-ods-state');
            config.odsstate = JSON.parse(arg);
            backend._service.setConfig(JSON.stringify(config));
            event.sender.send('service-get-ods-state-reply', JSON.stringify(config.odsstate));
        });

        /* GET SMS FROM CONFIG */
        backend._ipcMain.on('service-get-sms', (event, arg) => {
            var config = backend._service.getConfig();
            console.log('service-get-sms');
            let sms = {
                "enabled": false,
            };
            if (config.hasOwnProperty('communication')) {
                if (config.communication.hasOwnProperty('sms')) {
                    sms = config.communication.sms;
                }
            }
            event.sender.send('service-get-sms-reply', JSON.stringify(sms));
        });

        /* SAVE SMS KEYMAPPING */
        backend._ipcMain.on('service-save-sms', (event, arg) => {
            var config = backend._service.getConfig();
            console.log('service-save-sms');
            if (!config.hasOwnProperty('communication')) {
                config.communication = {};
            }
            if (!config.communication.hasOwnProperty('sms')) {
                config.communication.sms = {};
            }
            let themappings = JSON.parse(arg);
            config.communication.sms = themappings;
            backend._service.setConfig(JSON.stringify(config));
        });
        /* GET SOUND FROM CONFIG */
        backend._ipcMain.on('service-get-sound', (event, arg) => {
            var config = backend._service.getConfig();
            console.log('service-get-sound');
            let updateOrder = {
                'updateOrder' : {
                    "enabled" : false,
                }
            };
            if (config.hasOwnProperty('sound')) {
                if (config.sound.hasOwnProperty('updateOrder')) {
                    updateOrder = config.sound.updateOrder;
                }
            }
            event.sender.send('service-get-sound-reply', JSON.stringify(updateOrder));
        });
        /* SAVE SOUND CONFIG */
        backend._ipcMain.on('service-save-sound', (event, arg) => {
            var config = backend._service.getConfig();
            console.log('service-save-sound');
                
            let themappings = JSON.parse(arg);
            config.sound = themappings;
            backend._service.setConfig(JSON.stringify(config));
            event.sender.send('service-config-reply', JSON.stringify(backend.loadConfigPath(backend._service.kdsconfig.file)));
        });

        /* SAVE APP CONFIG */
        backend._ipcMain.on('service-config-save', (event, arg) => {
            console.log('service-config-save');
            log.info('service-config-save', arg)
            backend._service.setConfig(arg, backend._hwid, backend._nserver);
            event.sender.send('service-config-save-reply', 'done');
        });

        /* TEST SMS FUNCTIONALITY */
        backend._ipcMain.on('service-test-sms', (event, arg) => {
            console.log('service-test-sms');
            let conf = backend._service.getConfig();
            try {
                let twilioclient = require('twilio')(conf.communication.sms.twilio.ACCOUNT_SID, conf.communication.sms.twilio.AUTH_TOKEN);
                twilioclient.messages.create({
                    body: conf.communication.sms.twilio.MESSAGE,
                    from: conf.communication.sms.twilio.SENDING_NUMBER,
                    to: arg
                }).then(message => {
                    event.sender.send('service-test-sms-reply', JSON.stringify(message));
                });
            } catch (err) {
                event.sender.send('service-test-sms-reply', JSON.stringify(err));
            }
        });

        /* REGISTER TERMINAL VIA WEBSERVICE */
        backend._ipcMain.on('service-register-terminal', (event, arg) => {
            console.log('service-register-terminal');
            let id = backend.requestHWID();
            console.log("HWID : " + id);
            var xmlHttp = new XMLHttpRequest();
            xmlHttp.onreadystatechange = function () {
                if (xmlHttp.readyState === 4 && xmlHttp.status === 200) {
                    try {
                        let j = JSON.parse(xmlHttp.responseText);
                        if (j.data.id > 0) {
                            console.log("REGISTERED");
                            let c = backend._service.getConfig();
                            c.client.serial = arg;
                            backend._service.setConfig(JSON.stringify(c));
                            backend._validTerminal = true;
                            event.sender.send('service-register-terminal-reply', backend._validTerminal);
                        } else {
                            console.log(j);
                            console.log("UNREGISTERED");
                            backend._validTerminal = false;
                            event.sender.send('service-register-terminal-reply', backend._validTerminal);
                        }
                    } catch (err) {
                        console.log(err);
                    }
                }
            };
            let d = {
                connection_code: arg,
                hwid: id
            };
            xmlHttp.open("POST", "https://admin.tabletpos.co.za:8080/kds/terminal/register");
            xmlHttp.setRequestHeader("Content-Type", "application/json");
            console.log(d);
            xmlHttp.send(JSON.stringify(d));
        });

        /* RESTART AND UPDATE ALL BUT THE NATS SERVICE */
        backend._ipcMain.on('service-update', (event, arg) => {
            backend._service.stopService('kdsweb');
            backend._service.updateService('kdsweb', true, () => {
                backend._service.runService('kdsweb');
                backend._service.stopService('kdstcpserver');
                backend._service.updateService('kdstcpserver', true, () => {
                    backend._service.runService('kdstcpserver');
                    backend._service.stopService('kdspoller');
                    backend._service.updateService('kdspoller', true, () => {
                        backend._service.runService('kdspoller');
                        event.sender.send('service-status-reply', JSON.stringify({
                            kdstcpserver: backend._service.getService('kdstcpserver').status,
                            kdsnats: backend._service.getService('kdsnats').status,
                            kdspoller: backend._service.getService('kdspoller').status,
                            kdsweb: backend._service.getService('kdsweb').status,
                        }));
                    });
                });
            });
        });
        /* INSTALL THE AUTOSTART FILE FOR RASPBERRY PI */
        backend._ipcMain.on('service-autostart', (event, arg) => {
            console.log('service-autostart');
            backend._service.addAutostart();
        });
        /* RESTART ALL THE SERVICES */
        backend._ipcMain.on('service-restart', (event, arg) => {
            console.log("Restarting");
            // An Order Display Screen never ingests tickets of its own (see
            // index.js checkPoller) - kdspoller is left untouched here too,
            // otherwise saving any setting would silently start it anyway
            // (the installer pre-seeds the kdspoller binary regardless of
            // mode, so it's there to run even though ODS boot deliberately
            // never launches it).
            var conf = {};
            try {
                conf = backend._service.getConfig();
            } catch (err) {
                console.log(err);
            }
            var isOdsMode = !!(conf.flags && conf.flags.odsmode === true);

            backend._service.stopService('kdsweb');
            setTimeout(function () {
                backend._service.stopService('kdstcpserver');
                if (!isOdsMode) {
                    backend._service.stopService('kdspoller');
                }
                setTimeout(function () {
                    backend._service.runService('kdsweb');
                    backend._service.runService('kdstcpserver');
                    if (!isOdsMode) {
                        backend._service.runService('kdspoller');
                    }
                    setTimeout(function () {
                        console.log('DONE');
                        event.sender.send('service-restart-reply', JSON.stringify({
                            kdstcpserver: backend._service.getService('kdstcpserver').status,
                            kdsnats: backend._service.getService('kdsnats').status,
                            kdspoller: backend._service.getService('kdspoller').status,
                            kdsweb: backend._service.getService('kdsweb').status,
                        }));
                    }, 5000);
                }, 3000);
            }, 3000);
        });
    }
};

module.exports = backend;
