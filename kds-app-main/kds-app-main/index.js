'use strict';

const { app, BrowserWindow, ipcMain, remote, shell } = require('electron');
const path = require('path');
// The setuid chrome-sandbox helper binary needs root:root ownership and
// mode 4755 to work at all - Chromium aborts on startup (FATAL, "SUID
// sandbox helper binary was found, but is not configured correctly") if
// those bits aren't exactly right. That's brittle to preserve through a zip
// distribution: several common extraction tools (GUI archive managers in
// particular) strip setuid bits as a security precaution, so a machine that
// installed via "extract zip, then dpkg -i" can end up with a perfectly
// valid package whose sandbox helper silently lost its setuid bit in
// transit. Disabling the sandbox outright removes the dependency on that
// bit ever surviving the trip, on any machine.
app.commandLine.appendSwitch('no-sandbox');
app.disableHardwareAcceleration();
// disableHardwareAcceleration() alone still leaves Electron trying to spawn
// a software (SwiftShader) GPU process for compositing - on this machine
// that process itself fails to launch (error_code=1002) and Electron treats
// an unusable GPU process as fatal, crashing the whole app ~8s after
// startup. These switches stop it from ever spawning a GPU process at all.
app.commandLine.appendSwitch('disable-gpu');
app.commandLine.appendSwitch('disable-gpu-compositing');
app.commandLine.appendSwitch('disable-software-rasterizer');
// The above still leaves Electron spawning a separate process for GPU work
// (a "GPU process" running SwiftShader in software mode even with GPU
// hardware disabled) - on this machine that separate process itself fails
// to launch (error_code=1002), which Electron treats as fatal. Forcing GPU
// work to run inside the main process instead avoids that separate launch
// altogether.
app.commandLine.appendSwitch('in-process-gpu');
//require('electron-reload')(__dirname);
const DEDEBUG = true;
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
    console.log('DEBUG: BrowserWindow constructed, calling loadFile loading.html at ' + path.join(__dirname, 'loading.html'));
    mainWindow.loadFile(path.join(__dirname, 'loading.html')).then( () => {
      console.log('DEBUG: loadFile resolved');
      mainWindow.webContents.openDevTools();
      backend._mainWindow = mainWindow;
      startup();
    }).catch((e) => {
      console.log('DEBUG: loadFile rejected: ' + e.stack);
    });
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
  }
};

function startup(){
  backend._service.init((app || remote.app).getPath('userData'));
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
  backend._service.closeService(serviceName , () => {
    if (!backend._service.checkService(serviceName)){
      backend._service.updateService(serviceName , true , () => {
        backend._service.runService(serviceName);
        checkDB();
      });
    } else {
      backend._service.runService(serviceName);
      //checkDB();
    }
  });
}
function checkDB(){
  let serviceName = 'kdsdb';
  if (!backend._service.checkService(serviceName)) {
    backend._service.updateService(serviceName , true , () => {
      checkTCP();
    });
  } else {
    checkTCP();
  }
}
function checkTCP(){
  let serviceName = 'kdstcpserver';
  backend._service.closeService(serviceName , () => {
    if (!backend._service.checkService(serviceName)){
      backend._service.updateService(serviceName , true , () => {
        backend._service.runService(serviceName);
        checkPoller();
      });
    } else {
      backend._service.runService(serviceName);
      checkPoller();
    }
  });
}
function checkPoller(){
  let serviceName = 'kdspoller';
  backend._service.closeService(serviceName , () => {
    if (!backend._service.checkService(serviceName)){
      backend._service.updateService(serviceName , true , () => {
        backend._service.runService(serviceName);
        checkWeb();
      });
    } else {
      backend._service.runService(serviceName);
      checkWeb();
    }
  });
}
function checkWeb(){
  let serviceName = 'kdsweb';
  backend._service.closeService(serviceName , () => {
    if (!backend._service.checkService(serviceName)){
      backend._service.updateService(serviceName , true , () => {
        backend._service.runService(serviceName);
        gotime();
      });
    } else {
      backend._service.runService(serviceName);
      gotime();
    }
  });
}
function gotime(){  
  setTimeout( function() {
    backend.initIPC();
    let page = "register.html";
    //let page = "clientconfig.html";
    if (backend._validTerminal) {
      page = "index.html";
      //page = "clientconfig.html";
    } 
    mainWindow.loadFile(path.join(__dirname, page)).then(() => {
      //backend.initIPC();
    });
  } , 10000);
};

function bypassRegistration(){
  mainWindow.loadFile(path.join(__dirname, "index.html")).then(() => {
    //backend.initIPC();
  });
};
// This method will be called when Electron has finished
// initialization and is ready to create browser windows.
// Some APIs can only be used after this event occurs.
app.on('ready', () => {
  console.log('DEBUG: ready event fired, gotTheLock=' + gotTheLock);
  try {
    createWindow();
    console.log('DEBUG: createWindow() returned, mainWindow=' + (mainWindow ? 'set' : 'null'));
  } catch (e) {
    console.log('DEBUG: createWindow threw: ' + e.stack);
  }
});
process.on('uncaughtException', (e) => {
  console.log('DEBUG: uncaughtException: ' + e.stack);
});
process.on('unhandledRejection', (e) => {
  console.log('DEBUG: unhandledRejection: ' + (e && e.stack ? e.stack : e));
});
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
  // Tells the main board window to reload right now, the same way it
  // already does for a config update arriving from a remote terminal (see
  // backend._communication.natsIncoming's own 'config-updates-remotely'
  // sends) - KDS.kdsconfig is otherwise only ever set once, at that
  // window's own page load, so saving a change here (Screen, Pass Mode,
  // Standalone Mode, item colors, etc.) previously took effect for the
  // board only after someone happened to restart the whole app. The
  // backend services (kdswebserver/kdspoller) still separately restart via
  // the existing 'service-restart' IPC call the config screen's own save
  // flow makes right after this - this only covers the board window's own
  // stale in-memory state, which that restart doesn't touch.
  try {
    if (mainWindow) {
      mainWindow.webContents.send('config-updates-remotely', 'DONE');
    }
  } catch (err) {
    console.log(err);
  }
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