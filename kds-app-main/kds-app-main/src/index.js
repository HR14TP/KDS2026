'use strict';

const { app, BrowserWindow, ipcMain, remote, shell } = require('electron');
const path = require('path');
const fs = require('fs');
app.disableHardwareAcceleration();

// The app was renamed from "kds2024" to "kdstabletpos" (package.json's name/
// productName, which Electron derives getPath('userData') from) - on a
// machine that already had kds2024 installed, that rename alone would make
// the app start up looking empty (no HWID registration, no client config,
// no order history) since it'd now be pointed at a brand new, empty
// ~/.config/kdstabletpos instead of the existing ~/.config/kds2024. Carry
// the old folder over once, before anything below reads getPath('userData').
try {
  const oldUserData = path.join(app.getPath('appData'), 'kds2024');
  const newUserData = app.getPath('userData');
  if (fs.existsSync(oldUserData) && !fs.existsSync(newUserData)) {
    fs.cpSync(oldUserData, newUserData, { recursive: true });
  }
} catch (err) {
  console.log(err);
}
//require('electron-reload')(__dirname);
const DEDEBUG = false;
const backend = require('../src/js/backend/backend.js');
var mainWindow = null;
const kds_sentry = require("../src/js/libs/sentry.js");
const gotTheLock = app.requestSingleInstanceLock();
const kdshwid = require("./js/libs/kdshwid.js");
var {Howl } = require('howler');

// Handle creating/removing shortcuts on Windows when installing/uninstalling.
if (require('electron-squirrel-startup')) 
{
  app.quit();
}

if (!gotTheLock) 
{
  app.quit()
}

// Initialize Sentry Error Reportings
try{
  kds_sentry.init();
  kdshwid.initHWID((app || remote.app).getPath('userData'));
  backend._kdshwid = kdshwid;  
}catch(err){
  console.log(err);
}

const createWindow = () => {    
  // Create the browser window.
  if (DEDEBUG){
    mainWindow = new BrowserWindow({
      width: 800,
      height: 600,
      kiosk: false,
      frame: true,
      webPreferences: {
        nodeIntegration: true,
        webSecurity: true,
        contextIsolation: false,
        preload: path.join(__dirname, 'preload.js')
      }
    });    
    mainWindow.loadFile(path.join(__dirname, 'loading.html')).then( () => {
      mainWindow.webContents.openDevTools();
      backend._mainWindow = mainWindow;
      startup();
    });
    mainWindow.once('ready-to-show', () => mainWindow.focus());
  } else {
    mainWindow = new BrowserWindow({
      width: 800,
      height: 600,
      kiosk: true,
      frame: false,
      alwaysOnTop: true,
      fullscreen : true,
      webPreferences: {
        nodeIntegration: true,
        webSecurity: true,
        contextIsolation: false,
        preload: path.join(__dirname, 'preload.js')
      }
    });    
    mainWindow.loadFile(path.join(__dirname, 'loading.html')).then( () => {
      backend._mainWindow = mainWindow;
      startup();
    });
    // A kiosk/alwaysOnTop window on Linux doesn't always pick up OS keyboard
    // focus on its own the first time it's shown - most visible on a truly
    // fresh boot, landing on register.html/clientconfig.html, where the
    // window looks focused but keystrokes never reach the page at all.
    // Forcing focus once it's actually paintable avoids that.
    mainWindow.once('ready-to-show', () => mainWindow.focus());
  }
};

// Boot progress reporting for loading.html - every background task run
// before the registration/main screen can show (checking, downloading,
// and starting each of the 5 local services) reports into one shared
// percent range and a running task log, rather than the old bare
// spinner giving no sense of what's actually happening or how long is
// left, which looked identical whether the app was 1 second or 30
// seconds from being usable.
const BOOT_STAGE_RANGES = {
  init: [0, 5],
  nats: [5, 20],
  db: [20, 35],
  tcp: [35, 55],
  poller: [55, 75],
  web: [75, 95],
  finalize: [95, 100],
};
function sendBootProgress(percent, label) {
  try {
    if (mainWindow && mainWindow.webContents) {
      mainWindow.webContents.send('boot-progress', { percent: percent, label: label });
    }
  } catch (err) {
    console.log(err);
  }
}
function stagePercent(stageKey, fraction) {
  const range = BOOT_STAGE_RANGES[stageKey];
  return range[0] + (range[1] - range[0]) * Math.max(0, Math.min(1, fraction));
}
// Wraps a service's updateService() download/install step with progress
// reporting into the given stage's percent range - shared by every one
// of the 5 services below instead of repeating the same byte-math each
// time.
function updateServiceWithProgress(serviceName, stageKey, label, cb) {
  backend._service.updateService(serviceName, true, cb, (total, received) => {
    const totalBytes = Number(total);
    if (totalBytes > 0) {
      const pct = Math.round((received / totalBytes) * 100);
      sendBootProgress(stagePercent(stageKey, 0.2 + 0.6 * (received / totalBytes)), label + '... ' + pct + '%');
    }
  });
}

function startup(){
  sendBootProgress(stagePercent('init', 0), 'Starting up...');
  backend._service.init((app || remote.app).getPath('userData'));
  sendBootProgress(stagePercent('init', 1), 'Local services initialized');
  setTimeout(function(){
    startNats();
    setTimeout(function(){
      backend.validateTerminal();
      checkDB();
    },5000);
  } , 5000);
}

function startNats(){
  let serviceName = 'kdsnats';
  sendBootProgress(stagePercent('nats', 0), 'Checking message bus...');
  backend._service.closeService(serviceName , () => {
    if (!backend._service.checkService(serviceName)){
      sendBootProgress(stagePercent('nats', 0.2), 'Downloading message bus');
      updateServiceWithProgress(serviceName, 'nats', 'Downloading message bus', () => {
        sendBootProgress(stagePercent('nats', 0.9), 'Starting message bus...');
        backend._service.runService(serviceName);
        sendBootProgress(stagePercent('nats', 1), 'Message bus ready');
        checkDB();
      });
    } else {
      sendBootProgress(stagePercent('nats', 0.9), 'Starting message bus...');
      backend._service.runService(serviceName);
      sendBootProgress(stagePercent('nats', 1), 'Message bus ready');
      //checkDB();
    }
  });
}
function checkDB(){
  let serviceName = 'kdsdb';
  sendBootProgress(stagePercent('db', 0), 'Checking local database...');
  if (!backend._service.checkService(serviceName)) {
    sendBootProgress(stagePercent('db', 0.2), 'Downloading local database');
    updateServiceWithProgress(serviceName, 'db', 'Downloading local database', () => {
      sendBootProgress(stagePercent('db', 1), 'Local database ready');
      checkTCP();
    });
  } else {
    sendBootProgress(stagePercent('db', 1), 'Local database ready');
    checkTCP();
  }
}
function checkTCP(){
  let serviceName = 'kdstcpserver';
  sendBootProgress(stagePercent('tcp', 0), 'Checking ticket receiver...');
  backend._service.closeService(serviceName , () => {
    if (!backend._service.checkService(serviceName)){
      sendBootProgress(stagePercent('tcp', 0.2), 'Downloading ticket receiver');
      updateServiceWithProgress(serviceName, 'tcp', 'Downloading ticket receiver', () => {
        sendBootProgress(stagePercent('tcp', 0.9), 'Starting ticket receiver...');
        backend._service.runService(serviceName);
        sendBootProgress(stagePercent('tcp', 1), 'Ticket receiver ready');
        checkPoller();
      });
    } else {
      sendBootProgress(stagePercent('tcp', 0.9), 'Starting ticket receiver...');
      backend._service.runService(serviceName);
      sendBootProgress(stagePercent('tcp', 1), 'Ticket receiver ready');
      checkPoller();
    }
  });
}
function checkPoller(){
  // An Order Display Screen never ingests tickets of its own - it only ever
  // learns about orders via the sync broadcasts other terminals send
  // (including the new-order broadcast that carries the ticket's real
  // "Chk N" number - see kds-webserver's applyNewOrderRemote). Running its
  // own poller too would race that broadcast and create a second copy of
  // the same order under a locally-computed, inconsistent number, which is
  // exactly the bug this is meant to avoid.
  try {
    let conf = backend._service.getConfig();
    if (conf.flags && conf.flags.odsmode === true) {
      sendBootProgress(stagePercent('poller', 1), 'Order Display Mode - skipping order processor');
      checkWeb();
      return;
    }
  } catch (err) {
    console.log(err);
  }
  let serviceName = 'kdspoller';
  sendBootProgress(stagePercent('poller', 0), 'Checking order processor...');
  backend._service.closeService(serviceName , () => {
    if (!backend._service.checkService(serviceName)){
      sendBootProgress(stagePercent('poller', 0.2), 'Downloading order processor');
      updateServiceWithProgress(serviceName, 'poller', 'Downloading order processor', () => {
        sendBootProgress(stagePercent('poller', 0.9), 'Starting order processor...');
        backend._service.runService(serviceName);
        sendBootProgress(stagePercent('poller', 1), 'Order processor ready');
        checkWeb();
      });
    } else {
      sendBootProgress(stagePercent('poller', 0.9), 'Starting order processor...');
      backend._service.runService(serviceName);
      sendBootProgress(stagePercent('poller', 1), 'Order processor ready');
      checkWeb();
    }
  });
}
function checkWeb(){
  let serviceName = 'kdsweb';
  sendBootProgress(stagePercent('web', 0), 'Checking web server...');
  backend._service.closeService(serviceName , () => {
    if (!backend._service.checkService(serviceName)){
      sendBootProgress(stagePercent('web', 0.2), 'Downloading web server');
      updateServiceWithProgress(serviceName, 'web', 'Downloading web server', () => {
        sendBootProgress(stagePercent('web', 0.9), 'Starting web server...');
        backend._service.runService(serviceName);
        sendBootProgress(stagePercent('web', 1), 'Web server ready');
        gotime();
      });
    } else {
      sendBootProgress(stagePercent('web', 0.9), 'Starting web server...');
      backend._service.runService(serviceName);
      sendBootProgress(stagePercent('web', 1), 'Web server ready');
      gotime();
    }
  });
}
function gotime(){
  sendBootProgress(stagePercent('finalize', 0), 'Finishing up...');
  setTimeout( function() {
    backend.initIPC();
    sendBootProgress(100, 'Ready');
    let page = "register.html";
    //let page = "clientconfig.html";
    if (backend._validTerminal) {
      page = "index.html";
      //page = "clientconfig.html";
      // A terminal switched into Order Display Mode boots straight to the
      // customer-facing ready board instead of the normal kitchen board.
      try {
        let conf = backend._service.getConfig();
        if (conf.flags && conf.flags.odsmode === true) {
          page = "ods.html";
        }
      } catch (err) {
        console.log(err);
      }
    }
    mainWindow.loadFile(path.join(__dirname, page)).then(() => {
      //backend.initIPC();
      // register.html/clientconfig.html are the first screens with actual
      // text inputs the app shows - re-assert OS keyboard focus here too
      // (not just at initial paint ~20s of service-startup delays ago),
      // since a kiosk/alwaysOnTop window on Linux can otherwise sit there
      // looking focused while keystrokes go nowhere.
      mainWindow.focus();
    });
  } , 10000);
};

function bypassRegistration(){
  mainWindow.loadFile(path.join(__dirname, "index.html")).then(() => {
    //backend.initIPC();
    mainWindow.focus();
  });
};
// This method will be called when Electron has finished
// initialization and is ready to create browser windows.
// Some APIs can only be used after this event occurs.
app.on('ready', createWindow);
// Quit when all windows are closed, except on macOS. There, it's common
// for applications and their menu bar to stay active until the user quits
// explicitly with Cmd + Q.
app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    setTimeout(function(){
      backend._service.stopService('kdstcpserver');
      backend._service.stopService('kdspoller');
      backend._service.stopService('kdsweb');
      backend._service.stopService('kdsnats');
      setTimeout(function(){         
        app.quit();
      },2000);
    },2000);                    
  } else {
    setTimeout(function(){
      console.log("STOPPING SERVICES");
      backend._service.stopService('kdstcpserver');
      backend._service.stopService('kdspoller');
      backend._service.stopService('kdsweb');
      backend._service.stopService('kdsnats');            
    },2000); 
  }
});

app.on('activate', () => {
  // On OS X it's common to re-create a window in the app when the
  // dock icon is clicked and there are no other windows open.
  if (BrowserWindow.getAllWindows().length === 0) {
    createWindow();
  }
});



//*********** END IPC MESSAGES *************/

// In this file you can include the rest of your app's specific main process
// code. You can also put them in separate files and import them here.

const { LocalEvents , LocalRequests , GlobalEvents } = require(__dirname + '/js/libs/globals/events.js');
const { IPC_Client } = require(__dirname + '/js/libs/interface/ipclient.js');
const mainIPC = new IPC_Client(ipcMain, mainWindow);

function IPC_RequestConfig(event, args) {
  console.log('IPC_RequestConfig');
  var config = backend._service.getConfig();
  event.returnValue = JSON.stringify(config);
};
function IPC_SaveConfig(event, args) {
  console.log('IPC_SaveConfig');
  backend._service.setConfig(args, backend._hwid, backend._nserver);    
};
function IPC_PlaySound(event, args) {
  console.log('IPC_PlaySound');
      var sound = new Howl({
        src: ['assets/ding.mp3']
      });
      
      sound.play();
};

mainIPC.register(LocalRequests.REQUEST_CONFIG, IPC_RequestConfig);
mainIPC.register(LocalRequests.REQUEST_CONFIG_SAVE, IPC_SaveConfig);
mainIPC.register(LocalEvents.EVENT_PLAY_SOUND, IPC_PlaySound);
mainIPC.register("registration-skip-for-now", bypassRegistration);