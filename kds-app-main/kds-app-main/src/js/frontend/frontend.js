'use-strict';
const ipc = require('electron').ipcRenderer;
const fs = require('fs');
const path = require('path');
const KDS = require('./js/frontend/kds.js');
const ipcFront = require('./js/frontend/ipcfront.js');
const { config } = require('process');
var {Howl } = require('howler');

require('./js/dialog.js');

let frontend = {
    id: 'frontend',
    name: 'frontend',
    _fs: fs,
    _path: path,
    _kds: KDS,
    _config: {},
    _clientconfig: {},
    _register: {},
    _home: {},
    _ipc: ipc,
    _ipcFront: ipcFront,
};
frontend._config.config = false;

// Which page a service-restart should land on. Checked in this order: a
// save just made on this page (frontend._config.justSavedOdsMode, set by
// saveConfig - the most current answer, since the on-disk config a restart
// reads back may not have propagated to frontend._config.config yet), then
// whatever config was last loaded (covers register.html, which restarts an
// already-configured terminal without ever presenting the odsmode toggle
// itself). Defaults to the normal kitchen board.
frontend._nextPageAfterRestart = () => {
    if (typeof frontend._config.justSavedOdsMode === 'boolean') {
        return frontend._config.justSavedOdsMode ? 'ods.html' : 'index.html';
    }
    try {
        if (frontend._config.config !== false) {
            let conf = JSON.parse(frontend._config.config);
            if (conf.flags && conf.flags.odsmode === true) {
                return 'ods.html';
            }
        }
    } catch (err) {
        console.log(err);
    }
    return 'index.html';
};

frontend.loadConfig = () => {
    try {
        frontend._config.ipcINIT();
        frontend._ipc.send('service-config', 'all');
        const configPath = frontend._path.normalize( frontend._path.join(__dirname, 'config', 'kdsconfig.json'));
        const rawdata = frontend._fs.readFileSync(configPath);
        return JSON.parse(rawdata);
    } catch(err) {
        console.log(err);
        return false;
    }
};
frontend.initialize = () => {
    console.log("frontend startup");
    frontend._kds.ipcINIT();
    frontend._ipc.send('service-config', 'all');
    
};

frontend._config.registerTerminal = () => {
    let serial = document.getElementById('serialField').value;
    frontend._ipc.send('service-register-terminal', serial);
};
frontend._config.initialize = () => {
    dialog.init();
    frontend._config.ipcINIT();
    frontend._ipc.send('service-config', 'all');
};
// Shared by loadPassTerminals and rediscoverPassTerminals - fills the
// "Pass Terminal" <select> from a /getpassterminals-shaped terminal list.
// Always keeps selectedValue as an option even if it's missing from that
// list, so a previously-picked pass terminal that's temporarily offline (or
// just hasn't re-announced since this list was last fetched) doesn't
// silently disappear from the field and get overwritten with "None" on
// next save.
frontend._config._populatePassTerminalSelect = (select, terminals, selectedValue) => {
    let names = (terminals || []).map((t) => t.name).filter((name) => !!name);
    if (selectedValue && names.indexOf(selectedValue) === -1) {
        names.push(selectedValue);
    }
    select.innerHTML = '<option value="">None (bumps itself)</option>';
    names.forEach((name) => {
        let option = document.createElement('option');
        option.value = name;
        option.text = name;
        select.appendChild(option);
    });
    select.value = selectedValue || '';
};
// Populates the "Pass Terminal" dropdown from this terminal's own local
// backend (which accumulates announcements from every pass-mode terminal
// on-site - see announceAsPassTerminal in kds-webserver-main).
frontend._config.loadPassTerminals = (selectedValue) => {
    let select = document.getElementById('passterminalField');
    if (!select) {
        return;
    }
    let xmlHttp = new XMLHttpRequest();
    xmlHttp.onreadystatechange = () => {
        if (xmlHttp.readyState === 4) {
            let terminals = [];
            if (xmlHttp.status === 200) {
                try {
                    terminals = JSON.parse(xmlHttp.responseText) || [];
                } catch (err) {
                    console.log(err);
                }
            }
            frontend._config._populatePassTerminalSelect(select, terminals, selectedValue);
        }
    };
    xmlHttp.open("GET", "http://127.0.0.1:2020/getpassterminals", true);
    xmlHttp.send(null);
};
// Backs the "Rediscover Pass Terminals" button next to the dropdown -
// unlike loadPassTerminals (a passive read of whatever's already been
// heard), this asks this terminal's backend to actively re-announce and
// ask every other terminal on-site to do the same right now (see
// rediscoverPassTerminalsHandler in kds-webserver-main), then repopulates
// the dropdown from the result once that discovery round has had a moment
// to complete. Kept as its own function (rather than folding into
// loadPassTerminals) so the button can show its own pending state - the
// backend's ~1.5s grace period for LAN replies makes this visibly slower
// than the passive load, and a plain, unchanged-looking dropdown during
// that wait reads as "did my click even register?".
frontend._config.rediscoverPassTerminals = () => {
    let select = document.getElementById('passterminalField');
    let btn = document.getElementById('rediscoverPassTerminalsBtn');
    if (!select) {
        return;
    }
    let selectedValue = select.value;
    if (btn) {
        btn.disabled = true;
        btn.textContent = 'Rediscovering...';
    }
    let xmlHttp = new XMLHttpRequest();
    xmlHttp.onreadystatechange = () => {
        if (xmlHttp.readyState === 4) {
            let terminals = [];
            if (xmlHttp.status === 200) {
                try {
                    terminals = JSON.parse(xmlHttp.responseText) || [];
                } catch (err) {
                    console.log(err);
                }
            }
            frontend._config._populatePassTerminalSelect(select, terminals, selectedValue);
            if (btn) {
                // Standalone Mode may have been switched on while this was
                // in flight - if so, keep the button disabled rather than
                // blindly re-enabling a control that's not supposed to be
                // usable right now.
                let standaloneEl = document.getElementById('standalonemode');
                btn.disabled = !!(standaloneEl && standaloneEl.checked);
                btn.textContent = 'Rediscover Pass Terminals';
            }
        }
    };
    xmlHttp.open("GET", "http://127.0.0.1:2020/rediscoverpassterminals", true);
    xmlHttp.send(null);
};
// Backs the "Repair Broken Orders" button (Client Config -> Other) - an
// on-demand trigger for migrateOrphanedBlankScreenItems/
// repairBrokenOrdersHandler in kds-webserver-main, for anyone who doesn't
// want to wait for the next restart (that same repair also runs
// automatically at boot). Reports how many items it found and fixed, since
// 0 is a perfectly normal, healthy answer worth showing rather than hiding
// behind a bare "done".
frontend._config.repairBrokenOrders = () => {
    let btn = document.getElementById('repairBrokenOrdersBtn');
    if (btn) {
        btn.disabled = true;
        btn.textContent = 'Repairing...';
    }
    let xmlHttp = new XMLHttpRequest();
    xmlHttp.onreadystatechange = () => {
        if (xmlHttp.readyState === 4) {
            let repaired = null;
            if (xmlHttp.status === 200) {
                try {
                    repaired = JSON.parse(xmlHttp.responseText).repaired;
                } catch (err) {
                    console.log(err);
                }
            }
            if (btn) {
                btn.disabled = false;
                btn.textContent = 'Repair Broken Orders';
            }
            if (repaired === null) {
                alert('Repair failed - check the connection and try again.');
            } else if (repaired === 0) {
                alert('Nothing to repair - no broken orders found.');
            } else {
                alert('Repaired ' + repaired + ' item' + (repaired === 1 ? '' : 's') + ' stuck on ' + (repaired === 1 ? 'a broken screen' : 'broken screens') + '.');
            }
        }
    };
    xmlHttp.open("GET", "http://127.0.0.1:2020/repairbrokenorders", true);
    xmlHttp.send(null);
};
frontend._config.loadconfigHTML = () => {
    console.log('loadconfigHTML');
    if (frontend._config.config !== false) {        
        let conf = JSON.parse(frontend._config.config);
        
        document.getElementById('nameField').value = conf.client.name;
        document.getElementById('kdsField').value = conf.client.screen;
        document.getElementById('locationField').value = conf.client.terminal;
        document.getElementById('ipField').value = conf.server.host;
        //document.getElementById('serialField').value = conf.client.serial;
        document.getElementById("compressitems").checked = conf.flags.compressed;
        document.getElementById("showheadings").checked = conf.flags.showheadings;
        document.getElementById("leftalign").checked = conf.flags.leftalign;
        document.getElementById("checkboxes").checked = conf.flags.checkboxes;

        // Defaults to disabled (false) - existing installs keep bumping
        // through Pass Mode/Pass Terminal exactly as already configured
        // until someone explicitly opts a screen into Standalone Mode.
        // Applied (ApplyStandaloneMode) after passmode/passterminal are
        // loaded below, so it can immediately grey them out and blank the
        // terminal field's value if this screen already has it on.
        if (document.getElementById("standalonemode")) {
            document.getElementById("standalonemode").checked = conf.flags.standalonemode === true;
        }

        if (conf.flags.hasOwnProperty('passmode')) {
            document.getElementById("passmode").checked = conf.flags.passmode;
        } else {
            document.getElementById("passmode").checked = false;
        }

        if (document.getElementById("passterminalField")) {
            frontend._config.loadPassTerminals(conf.client.passterminal || '');
        }

        if (document.getElementById("standalonemode")) {
            frontend._clientconfig.ApplyStandaloneMode(conf.flags.standalonemode === true);
        }

        // Defaults to disabled (false), same as passmode - a terminal only
        // becomes an Order Display Screen if explicitly switched into it.
        if (document.getElementById("odsmode")) {
            document.getElementById("odsmode").checked = conf.flags.odsmode === true;
        }

        document.getElementById("fontsize").value = conf.flags.hasOwnProperty('fontsize') ? conf.flags.fontsize : 16;

        // Defaults to enabled (true) rather than the field's own falsy
        // default, unlike passmode above - existing installs upgrading to
        // this config option shouldn't lose in-production buttons/timers
        // just because the field didn't exist in their saved config yet.
        document.getElementById("productionmode").checked = conf.flags.hasOwnProperty('productionmode') ? conf.flags.productionmode : true;

        // Defaults to disabled (false) - "touch interface" is a new, opt-in
        // denser-vs-larger layout choice, not a fix existing installs
        // should be switched onto without asking.
        document.getElementById("touchui").checked = conf.flags.touchui === true;
        document.body.classList.toggle('touchui', conf.flags.touchui === true);

        // Defaults to disabled (false) - newest-first is the new default
        // sort, but an existing install that's already used to arrival
        // order (oldest-first, since orders are simply appended as they're
        // discovered) shouldn't have that flip under it unasked.
        if (document.getElementById("showoldestfirst")) {
            document.getElementById("showoldestfirst").checked = conf.flags.showoldestfirst === true;
        }

        // Defaults to disabled (false) - board sorting is a new, opt-in
        // behavior, not a fix existing installs should be switched onto
        // without asking.
        document.getElementById("sortbyprodstate").checked = conf.flags.sortbyprodstate === true;

        // Defaults to enabled (true), same reasoning as productionmode
        // above - tile collapse/expand was already always-on before this
        // became a flag, so an existing install shouldn't lose it just
        // because the field didn't exist in its saved config yet.
        if (document.getElementById("enabletilecollapse")) {
            document.getElementById("enabletilecollapse").checked = conf.flags.hasOwnProperty('enabletilecollapse') ? conf.flags.enabletilecollapse : true;
        }

        // Defaults to disabled (false) - opt-in layout change, not a fix
        // existing installs should be switched onto without asking.
        if (document.getElementById("fixedtilesize")) {
            document.getElementById("fixedtilesize").checked = conf.flags.fixedtilesize === true;
        }

        // Defaults to 2 (see wimpy.css's own --tile-fixed-rows fallback) -
        // matches what Fixed Tile Size always did before this became
        // adjustable, so an existing install's tiles don't silently change
        // size just from upgrading.
        if (document.getElementById("fixedtilerows")) {
            document.getElementById("fixedtilerows").value = conf.flags.hasOwnProperty('fixedtilerows') ? conf.flags.fixedtilerows : 2;
        }

        // Defaults to disabled (false) too - existing installs keep the
        // original flat MAIN ITEMS/SUB ITEMS summary until someone opts
        // into the color-coded-by-status one.
        document.getElementById("summarybystatus").checked = conf.flags.summarybystatus === true;

        document.getElementById("late1color").value = conf.colors.late1;
        document.getElementById("late2color").value = conf.colors.late2;
        //document.getElementById("waitercolor").value = conf.colors.waitercolor;
        if (conf.colors.hasOwnProperty('late1time')) {
            document.getElementById("late1time").value = conf.colors.late1time;
        } else {
            document.getElementById("late1time").value = 8;
        }
        if (conf.colors.hasOwnProperty('late2time')) {
            document.getElementById("late2time").value = conf.colors.late2time;
        } else {
            document.getElementById("late2time").value = 10;
        }

        if (conf.flags.hasOwnProperty('enablemulticolumn')) {
            document.getElementById("enablemulticolumn").checked = conf.flags.enablemulticolumn;
        } else {
            document.getElementById("enablemulticolumn").checked = false;
        }        

        //Show All Production Centers
        if (conf.flags.hasOwnProperty('enableallcenters')) {
            document.getElementById("enableallcenters").checked = conf.flags.enableallcenters;
        } else {
            document.getElementById("enableallcenters").value = false;
        }
        //Allow Multiple Process To Be Open
        if (conf.flags.hasOwnProperty('allowmultiprogram')) {
            document.getElementById("allowmultiprogram").checked = conf.flags.allowmultiprogram;
        } else {
            document.getElementById("allowmultiprogram").value = false;
        }

        //Enable playing sound on Order updates
        if (undefined !== conf.sound && conf.hasOwnProperty('sound')) {
            document.getElementById("updateOrderSound").checked = conf.sound.updateOrder.enabled;
        } else {
            conf.sound = {};
            conf.sound.updateOrder = {};
            conf.sound.updateOrder.enabled = false;
        }

        // NEW
        if (undefined !== conf.duplicate && conf.hasOwnProperty('duplicate')) {
            document.getElementById("enableduplication").checked =  conf.duplicate.enableduplication;
            document.getElementById("screen1").value = conf.duplicate.screen1;
            // screen2 is a second NATS-mirror target (same mechanism as
            // screen1); peers is an arbitrary-length list, fanned out to via
            // plain HTTP instead of a NATS connection, for sites with more
            // than two other KDS terminals that all need to receive updates
            // (e.g. in-production status) no matter which one they're made on.
            document.getElementById("screen2").value = conf.duplicate.hasOwnProperty('screen2') ? conf.duplicate.screen2 : '';
            document.getElementById("peers").value = conf.duplicate.hasOwnProperty('peers') ? conf.duplicate.peers.join(', ') : '';
            if (conf.duplicate.enableduplication === true){
                document.getElementById("duplicationcontainer").classList.remove("dis");
            }
        } else {
            conf.duplicate = {};
            conf.duplicate.enableduplication = false;
            conf.duplicate.screen1 = '';
            conf.duplicate.screen2 = '';
            conf.duplicate.peers = [];
        }

        if (undefined !== conf.oncomplete) {
            document.getElementById("enablesendprinter").checked =  conf.oncomplete.enablesendprinter;
            document.getElementById("sendprinterip").value = conf.oncomplete.sendprinterip;
            if (conf.oncomplete.enablesendprinter === true){
                document.getElementById("sendprintercontainer").classList.remove("dis");     
            }
        } else {
            conf.duplicate.enablesendprinter = false;
            conf.duplicate.sendprinterip = '';
        }
        
        if (undefined !== conf.bumpbars && undefined !== conf.bumpbars.mapping) {  
            
            if (conf.bumpbars.mapping.hasOwnProperty("enabled")){
                document.getElementById("enablebump").checked = conf.bumpbars.mapping.enabled;                
            }
            if (conf.bumpbars.mapping.hasOwnProperty("KEY_CLOSE_ORDER")){
                document.getElementById("keycloseorder").value = conf.bumpbars.mapping["KEY_CLOSE_ORDER"].KEY;
            }
            if (conf.bumpbars.mapping.hasOwnProperty("KEY_RECALL_ORDER")){
                document.getElementById("keyrecallorder").value = conf.bumpbars.mapping["KEY_RECALL_ORDER"].KEY;
            }
            if (conf.bumpbars.mapping.hasOwnProperty("KEY_CANCEL_ACTION")){
                document.getElementById("keycancelaction").value = conf.bumpbars.mapping["KEY_CANCEL_ACTION"].KEY;
            }             
        } else if (undefined !== conf.bumpbars) {  
                      
            if (conf.bumpbars.hasOwnProperty("enabled")){
                document.getElementById("enablebump").checked = conf.bumpbars.enabled;                
            }
            if (conf.bumpbars.hasOwnProperty("KEY_CLOSE_ORDER")){
                document.getElementById("keycloseorder").value = conf.bumpbars["KEY_CLOSE_ORDER"].KEY;
            }
            if (conf.bumpbars.hasOwnProperty("KEY_RECALL_ORDER")){
                document.getElementById("keyrecallorder").value = conf.bumpbars["KEY_RECALL_ORDER"].KEY;
            }
            if (conf.bumpbars.hasOwnProperty("KEY_CANCEL_ACTION")){
                document.getElementById("keycancelaction").value = conf.bumpbars["KEY_CANCEL_ACTION"].KEY;
            }             
        }

        if (undefined !== conf.kdsfeatures) {
            if (undefined !== conf.kdsfeatures.prioritykeywordenabled) {
                document.getElementById("prioritykeywordenabled").checked =  conf.kdsfeatures.prioritykeywordenabled;
                if (conf.kdsfeatures.prioritykeywordenabled === true){
                    document.getElementById("keywordcontainer").classList.remove("dis");     
                }
                if (undefined !== conf.kdsfeatures.prioritykeywordtext) {
                    document.getElementById("prioritykeywordtext").value =  conf.kdsfeatures.prioritykeywordtext;
                }
            } else {
                conf.kdsfeatures.prioritykeywordenabled = false;
                conf.kdsfeatures.prioritykeywordtext = '';
            }

            if (undefined !== conf.kdsfeatures.priorityitemenabled) {
                document.getElementById("priorityitemenabled").checked =  conf.kdsfeatures.priorityitemenabled;
                
                if (conf.kdsfeatures.priorityitemenabled === true){
                    document.getElementById("priorityitemcontainer").classList.remove("dis");     
                }

                if (undefined !== conf.kdsfeatures.priorityitemcontainertext1) {
                    document.getElementById("priorityitemcontainertext1").value =  conf.kdsfeatures.priorityitemcontainertext1;
                }
                if (undefined !== conf.kdsfeatures.priorityitemcontainertext2) {
                    document.getElementById("priorityitemcontainertext2").value =  conf.kdsfeatures.priorityitemcontainertext2;
                }
                if (undefined !== conf.kdsfeatures.priorityitemcontainertext3) {
                    document.getElementById("priorityitemcontainertext3").value =  conf.kdsfeatures.priorityitemcontainertext3;
                }

            } else {
                conf.kdsfeatures.priorityitemenabled = false;
                conf.kdsfeatures.prioritykeywordtext = '';
            }

            if (undefined !== conf.kdsfeatures.enablecourses) {
                document.getElementById("enablecourses").checked =  conf.kdsfeatures.enablecourses;
            }

            frontend._clientconfig.renderItemColorRows(conf.kdsfeatures.itemcolors);

        }
    }
};
frontend.exitApp = () => {
    window.close();
};
frontend._config.saveConfig = () => {
    dialog.showSpinnerDialog();
    if (frontend._config.config !== false){
        let conf = JSON.parse(frontend._config.config);
        console.log(conf);
        conf.client.name = document.getElementById('nameField').value;
        conf.client.screen = document.getElementById('kdsField').value;
        conf.client.terminal = document.getElementById('locationField').value;
        conf.server.host = document.getElementById('ipField').value;
        conf.client.passterminal = document.getElementById("passterminalField") ? document.getElementById("passterminalField").value : (conf.client.passterminal || '');
        //conf.client.serial = document.getElementById('serialField').value;
        conf.flags.compressed = document.getElementById("compressitems").checked;
        conf.flags.showheadings = document.getElementById("showheadings").checked;
        conf.flags.leftalign = document.getElementById("leftalign").checked;
        conf.flags.checkboxes = document.getElementById("checkboxes").checked;
        conf.flags.passmode = document.getElementById("passmode").checked;
        if (document.getElementById("standalonemode")) {
            conf.flags.standalonemode = document.getElementById("standalonemode").checked;
            if (conf.flags.standalonemode) {
                // Standalone Mode overrides both, regardless of whatever the
                // (disabled, but not physically unclickable via bumpbar/
                // keyboard shortcuts) controls above still show - see
                // ApplyStandaloneMode, which keeps the UI in sync with this
                // while the checkbox is on, but enforce it here too as the
                // actual source of truth on save.
                conf.flags.passmode = false;
                conf.client.passterminal = '';
            }
        }
        // config.html predates this field, so guard its absence there the
        // same way loadconfigHTML does above - keep whatever was already
        // saved rather than silently forcing it off.
        conf.flags.odsmode = document.getElementById("odsmode") ? document.getElementById("odsmode").checked : conf.flags.odsmode === true;
        // Cached so the service-restart-reply handler below knows which
        // screen to land on after the restart it triggers - reading the
        // checkbox again there is unreliable since the DOM may already be
        // mid-navigation by the time that reply arrives.
        frontend._config.justSavedOdsMode = conf.flags.odsmode;
        conf.flags.fontsize = parseInt(document.getElementById("fontsize").value, 10) || 16;
        conf.flags.productionmode = document.getElementById("productionmode").checked;
        conf.flags.touchui = document.getElementById("touchui").checked;
        if (document.getElementById("showoldestfirst")) {
            conf.flags.showoldestfirst = document.getElementById("showoldestfirst").checked;
        }
        conf.flags.sortbyprodstate = document.getElementById("sortbyprodstate").checked;
        if (document.getElementById("enabletilecollapse")) {
            conf.flags.enabletilecollapse = document.getElementById("enabletilecollapse").checked;
        }
        if (document.getElementById("fixedtilesize")) {
            conf.flags.fixedtilesize = document.getElementById("fixedtilesize").checked;
        }
        if (document.getElementById("fixedtilerows")) {
            // Clamped the same way the input's own min/max already suggest -
            // belt and braces against a value that got here some other way
            // (an older saved config, or the field emptied out entirely)
            // producing a broken (0, negative, or non-numeric) CSS calc().
            let rows = parseInt(document.getElementById("fixedtilerows").value, 10);
            if (isNaN(rows) || rows < 1) {
                rows = 1;
            } else if (rows > 6) {
                rows = 6;
            }
            conf.flags.fixedtilerows = rows;
        }
        conf.flags.summarybystatus = document.getElementById("summarybystatus").checked;
        conf.flags.enablemulticolumn = document.getElementById("enablemulticolumn").checked;
        

        conf.flags.enableallcenters = document.getElementById("enableallcenters").checked;
        conf.flags.allowmultiprogram = document.getElementById("allowmultiprogram").checked;
        
        if (undefined !== conf.bumpbars && undefined !== conf.bumpbars.mapping) {  
            let m = JSON.stringify(conf.bumpbars.mapping);
            //console.log(m);
            conf.bumpbars = JSON.parse(m);
            conf.bumpbars["mapping"] = {};          
            conf.bumpbars.mapping = JSON.parse(m);
        }

        if (undefined !== conf.sound && conf.hasOwnProperty('sound')) {
            conf.sound.updateOrder.enabled = document.getElementById("updateOrderSound").checked;
        } else {
            conf.sound = {};
            conf.sound.updateOrder = {};
            conf.sound.updateOrder.enabled = document.getElementById("updateOrderSound").checked;
        }        

        conf.colors.late1 = document.getElementById("late1color").value;
        conf.colors.late2 = document.getElementById("late2color").value;
        conf.colors.late1time = document.getElementById("late1time").value;
        conf.colors.late2time = document.getElementById("late2time").value;

        //NEW
        try{
            conf.duplicate.enableduplication = document.getElementById("enableduplication").checked;
            conf.duplicate.screen1 = document.getElementById("screen1").value;
            conf.duplicate.screen2 = document.getElementById("screen2").value;
            conf.duplicate.peers = document.getElementById("peers").value
                .split(',')
                .map((ip) => ip.trim())
                .filter((ip) => ip.length > 0);

            conf.oncomplete.enablesendprinter = document.getElementById("enablesendprinter").checked;
            conf.oncomplete.sendprinterip = document.getElementById("sendprinterip").value;

            conf.kdsfeatures.prioritykeywordenabled = document.getElementById("prioritykeywordenabled").checked;
            conf.kdsfeatures.prioritykeywordtext = document.getElementById("prioritykeywordtext").value;

            conf.kdsfeatures.priorityitemenabled = document.getElementById("priorityitemenabled").checked;
            conf.kdsfeatures.priorityitemcontainertext1 = document.getElementById("priorityitemcontainertext1").value;
            conf.kdsfeatures.priorityitemcontainertext2 = document.getElementById("priorityitemcontainertext2").value;
            conf.kdsfeatures.priorityitemcontainertext3 = document.getElementById("priorityitemcontainertext3").value;
            
            conf.kdsfeatures.enablecourses = document.getElementById("enablecourses").checked;

            conf.kdsfeatures.itemcolors = frontend._clientconfig.collectItemColors();
        } catch(err) {
            console.log(err);
        }
        
        try {
            
            if (undefined !== conf.bumpbars && undefined !== conf.bumpbars.mapping) {  
                let m = JSON.stringify(conf.bumpbars.mapping);
                //console.log(m);
                conf.bumpbars = JSON.parse(m);
                conf.bumpbars["mapping"] = {};          
                conf.bumpbars.mapping = JSON.parse(m);
            }
        }
        catch(err) {
            console.log(err);
        }

        //console.log(conf);
        frontend._ipc.send('service-config-save', JSON.stringify(conf));

        try{
            window.IPCOUT.saveconfig(JSON.stringify(conf));
        }catch(err){
            console.log(err);
        }        
    }
};
frontend._config.ipcINIT = () => {
    frontend._ipc.on('service-restart-reply' , (event, arg) => {
        dialog.hide();
        window.location = frontend._nextPageAfterRestart();
    });
    frontend._ipc.on('service-config-reply' , (event, arg) => {
        frontend._config.config = arg;  
        //console.log(arg);      
        frontend._config.loadconfigHTML();
    });
    frontend._ipc.on('service-config-save-reply' , (event, arg) => {
        frontend._ipc.send('service-restart', 'all');
    });
};
frontend._kds.ipcINIT = () => {
    frontend._ipc.on('service-config-reply' , (event, arg) => {
        frontend._kds.init(JSON.parse(arg));
    });
    frontend._ipc.on('config-updates-remotely' , (event, arg) => {
        console.log("config-updates-remotely");
        location.reload();
    }); 
    /*frontend._ipc.on('service-config-reply' , (event, arg) => {
        frontend._kds.init(JSON.parse(arg));
    });*/   
};


frontend._home.goAdmin = () => {
    dialog.askPassword('PASSWORD' , function(password) {
        dialog.hide();
        if (password === 'zzzzzaaaaaqqqqq') {
            window.location = 'admin.html';
        } else {
            alert('wrong password');
        }
    });    
};
frontend._home.goConfig = () => {
    dialog.askPassword('PASSWORD' , function(password) {
        dialog.hide();
        if (password === 'zzzzaaaaqqqq') {
            window.location = 'config.html';
        } else {
            alert('wrong password');
        }
    });    
};

frontend._register.initialize = () => {
    dialog.init();
    frontend._register.ipcINIT();  
    frontend._ipc.send('service-config', 'all');  
};
frontend._register.ipcINIT = () => {
    frontend._ipc.on('service-restart-reply' , (event, arg) => {
        dialog.hide();
        window.location = frontend._nextPageAfterRestart();
    });
    frontend._ipc.on('service-config-reply' , (event, arg) => {
        frontend._config.config = arg;          
        frontend._register.loadconfigHTML();    
    });
    frontend._ipc.on('service-config-save-reply' , (event, arg) => {
        frontend._ipc.send('service-restart', 'all');
    });
    frontend._ipc.on('service-register-terminal-reply' , (event, arg) => {
        dialog.hide();
        if (arg === false){
            alert("Your registration failed.");
        } else if (arg === true) {
            alert("Thank you for registering your KDS.");
            window.location = 'clientconfig.html';
        }
    });
};
frontend._register.loadconfigHTML = () => {
    if (frontend._config.config !== false) {
        let conf = JSON.parse(frontend._config.config);
        //console.log(conf);
        document.getElementById('serialField').value = conf.client.serial;
        document.body.classList.toggle('touchui', conf.flags && conf.flags.touchui === true);
    }
};
frontend._register.registerTerminal = () => {
    dialog.showSpinnerDialog();
    let serial = document.getElementById('serialField').value;
    frontend._ipc.send('service-register-terminal', serial);
};



/* CLIENT CONFIG PAGE */

frontend._clientconfig.TabsBindAll = () => {
    var menuElements = document.querySelectorAll('[data-tab]');
    for(var i = 0; i < menuElements.length ; i++) {
        menuElements[i].addEventListener('click', frontend._clientconfig.TabsChange, false);
    }
};
frontend._clientconfig.ToggleInput = (elem , id) => {
    var menuElement = document.getElementById(id);
    if (elem.checked == true){
        //menuElement.style.display = "block!important";
        menuElement.classList.remove("dis");
    } else {
        //menuElement.style.display = "none!important";
        menuElement.classList.add("dis");
    }         
};
// Standalone Mode is a single switch that guarantees this screen only ever
// bumps its own orders: forces Pass Mode off (see saveConfig, the actual
// source of truth) and clears any assigned Pass Terminal (canCloseOrderHere
// in kds.js only self-bumps when client.passterminal is empty). Disables
// (rather than hides) the two controls it overrides, so it reads as "not in
// play at all" rather than just switched off - re-enabled, unchanged,
// the moment Standalone Mode is switched off again. Called both from the
// checkbox's own onchange and once from loadconfigHTML so a screen that
// already has this on shows the disabled state immediately, before anyone
// touches anything.
frontend._clientconfig.ApplyStandaloneMode = (isStandalone) => {
    let passmodeEl = document.getElementById('passmode');
    let passterminalEl = document.getElementById('passterminalField');
    let rediscoverBtn = document.getElementById('rediscoverPassTerminalsBtn');
    if (passmodeEl) {
        passmodeEl.disabled = isStandalone;
        if (isStandalone) {
            passmodeEl.checked = false;
        }
    }
    if (passterminalEl) {
        passterminalEl.disabled = isStandalone;
        if (isStandalone) {
            passterminalEl.value = '';
        }
    }
    if (rediscoverBtn) {
        rediscoverBtn.disabled = isStandalone;
    }
};
// Item Highlight Colors (KDS Config -> Other): one row per {name, color}
// instead of the raw "name, #hexcode" textarea this replaced - a native
// color swatch (input[type=color]) means nobody has to know or type a hex
// code, and Add/Remove buttons make the list length obvious rather than
// relying on counting newlines correctly.
frontend._clientconfig.itemColorRow = (name, color) => {
    let row = document.createElement('div');
    row.className = 'itemcolor-row';

    let nameInput = document.createElement('input');
    nameInput.type = 'text';
    nameInput.className = 'itemcolor-name';
    nameInput.placeholder = 'Item name (e.g. kids meal)';
    nameInput.value = name || '';

    let colorInput = document.createElement('input');
    colorInput.type = 'color';
    colorInput.className = 'itemcolor-swatch';
    colorInput.value = color || '#ffcc00';

    let removeBtn = document.createElement('input');
    removeBtn.type = 'button';
    removeBtn.className = 'btn btn-danger itemcolor-remove';
    removeBtn.value = 'Remove';
    removeBtn.onclick = () => row.remove();

    row.appendChild(nameInput);
    row.appendChild(colorInput);
    row.appendChild(removeBtn);
    return row;
};
frontend._clientconfig.addItemColorRow = (name, color) => {
    let container = document.getElementById('itemcolors-rows');
    if (container) {
        container.appendChild(frontend._clientconfig.itemColorRow(name, color));
    }
};
frontend._clientconfig.renderItemColorRows = (itemcolors) => {
    let container = document.getElementById('itemcolors-rows');
    if (!container) {
        return;
    }
    container.innerHTML = '';
    (Array.isArray(itemcolors) ? itemcolors : []).forEach((entry) => {
        frontend._clientconfig.addItemColorRow(entry.name, entry.color);
    });
};
// Reads the current rows back into the {name, color}[] shape saveConfig
// stores - blank-named rows (an "Add" left untouched, or a name the user
// cleared out) are dropped rather than saved as a stray empty entry.
frontend._clientconfig.collectItemColors = () => {
    let rows = document.querySelectorAll('#itemcolors-rows .itemcolor-row');
    let result = [];
    rows.forEach((row) => {
        let name = row.querySelector('.itemcolor-name').value.trim();
        let color = row.querySelector('.itemcolor-swatch').value;
        if (name.length > 0) {
            result.push({ name: name, color: color });
        }
    });
    return result;
};

frontend._clientconfig.TabsClear = () => {
    var menuElements = document.querySelectorAll('[data-tab]');
    for(var i = 0; i < menuElements.length ; i++) {
        menuElements[i].classList.remove('active');
        var id = menuElements[i].getAttribute('data-tab');
        document.getElementById(id).classList.remove('active');
    }
};
frontend._clientconfig.TabsChange = (e) => {
    frontend._clientconfig.TabsClear();
    e.target.classList.add('active');
    var id = e.currentTarget.getAttribute('data-tab');
    document.getElementById(id).classList.add('active');
};
frontend._clientconfig.CollapsibleBindAll = () => {
    var headers = document.querySelectorAll('.settings-group-header.is-collapsible');
    for (var i = 0; i < headers.length; i++) {
        headers[i].addEventListener('click', frontend._clientconfig.CollapsibleToggle, false);
    }
};
frontend._clientconfig.CollapsibleToggle = (e) => {
    // A header can contain its own controls (admin.html's per-service
    // Update button, for example) - clicking those should not also
    // collapse the section underneath them.
    if (e.target.closest('input, button, a, select, textarea')) {
        return;
    }
    e.currentTarget.closest('.settings-group').classList.toggle('collapsed');
};
frontend._clientconfig.initialize = () => {
    console.log("CLIENT CONFIG INIT");
    dialog.init();
    frontend._clientconfig.TabsBindAll();
    frontend._clientconfig.CollapsibleBindAll();
    frontend._clientconfig.ipcINIT();
    frontend._ipc.send('service-get-keymap', 'all');
    frontend._ipc.send('service-get-sms', 'all');
    frontend._ipc.send('service-get-sound', 'all');
    frontend._ipc.send('service-config', 'all');
};
frontend._clientconfig.mapKey = (id , keyid) => {
    dialog.showSpinnerCustomDialog("Press the button on the bumpbar now please." , (event) => {
        //console.log(event);
        document.getElementById(id).value = event.key;
        frontend._clientconfig.mappings[keyid] = {
            "KEY" : event.key,
            "SHIFT" : event.shiftKey,
            "CONTROL" : event.ctrlKey,
            "ALT" : event.altKey
        };
        document.getElementById(id + '-save').disabled = false;
        //console.log(frontend._clientconfig.mappings);
    });//service-save-keymap
};
frontend._clientconfig.resetMapKey = (id , keyid) => {
    document.getElementById(id).value = '';
    frontend._clientconfig.mappings[keyid] = {
        "KEY" : "",
        "SHIFT" : false,
        "CONTROL" : false,
        "ALT" : false
    };
    document.getElementById(id + '-save').disabled = false;
    console.log(frontend._clientconfig.mappings); 
};
frontend._clientconfig.updateEnabled = () => {
    //console.log(document.getElementById("enablebump").checked);
    frontend._clientconfig.mappings.enabled = document.getElementById("enablebump").checked;
    if (frontend._clientconfig.mappings.enabled) {
        document.getElementById("bumpbar_extra").style.display = "block";
    } else {
        document.getElementById("bumpbar_extra").style.display = "none";
    }
    frontend._ipc.send('service-save-keymap', JSON.stringify(frontend._clientconfig.mappings));     
};
frontend._clientconfig.updateOrderSound = () => {
    console.log(frontend._clientconfig);
    if (!frontend._clientconfig.sound.hasOwnProperty('updateOrder')) {
        frontend._clientconfig.sound.updateOrder = {}; 
    }
    frontend._clientconfig.sound.updateOrder.enabled = document.getElementById("updateOrderSound").checked;

    frontend._ipc.send('service-save-sound', JSON.stringify(frontend._clientconfig.sound));     
};
frontend._clientconfig.updateMapKey = (mapkey) => {
    console.log(JSON.stringify(frontend._clientconfig.mappings));
    frontend._ipc.send('service-save-keymap', JSON.stringify(frontend._clientconfig.mappings));    
};
frontend._clientconfig.sendTestSMS = () => {
    frontend._ipc.send('service-test-sms' , document.getElementById("twilio_test_number").value);    
};
frontend._clientconfig.sendTestSound = () => {
    // var fileaudio = window.document.getElementById("kdstestaudio");
    // fileaudio.play();
    window.KDSAPI.playSound();      
 
    var sound = new Howl({
        src: ['assets/ding.mp3']
      });
      
      sound.play();
};
frontend._clientconfig.updateSMSEnabled = () => {
    //console.log(document.getElementById("enablesms").checked);
    frontend._clientconfig.smsconfig.enabled = document.getElementById("enablesms").checked;
    if (frontend._clientconfig.smsconfig.enabled) {
        document.getElementById("sms_extra").style.display = "block";
    } else {
        document.getElementById("sms_extra").style.display = "none";
    }
    frontend._ipc.send('service-save-sms', JSON.stringify(frontend._clientconfig.smsconfig));      
};
frontend._clientconfig.updateSMS = () => {
    if (!frontend._clientconfig.smsconfig.hasOwnProperty('twilio')){
        frontend._clientconfig.smsconfig.twilio = {};
    }
    frontend._clientconfig.smsconfig.enabled = document.getElementById("enablesms").checked;
    frontend._clientconfig.smsconfig.twilio.ACCOUNT_SID = document.getElementById("twilio_accountsid").value;
    frontend._clientconfig.smsconfig.twilio.AUTH_TOKEN = document.getElementById("twilio_authtoken").value;
    frontend._clientconfig.smsconfig.twilio.SENDING_NUMBER = document.getElementById("twilio_sendingnumber").value;
    frontend._clientconfig.smsconfig.twilio.MESSAGE = document.getElementById("twilio_message").value;
    //console.log(JSON.stringify(frontend._clientconfig.smsconfig));
    frontend._ipc.send('service-save-sms', JSON.stringify(frontend._clientconfig.smsconfig));    
};
/* END CLIENT CONFIG PAGE */
frontend._clientconfig.ipcINIT = () => {
    frontend._ipc.on('service-save-keymap-reply' , (event, arg) => {
        console.log(arg);        
    });
    frontend._ipc.on('service-test-sms-reply' , (event, arg) => {
        console.log(arg);        
    });
    frontend._ipc.on('service-config-reply' , (event, arg) => {
        console.log('service-config-reply');
        //console.log(arg);
        frontend._config.config = arg;
        frontend._config.loadconfigHTML();
    });
    frontend._ipc.on('service-restart-reply' , (event, arg) => {
        dialog.hide();
        window.location = frontend._nextPageAfterRestart();
    });
    frontend._ipc.on('service-config-save-reply' , (event, arg) => {
        frontend._ipc.send('service-restart', 'all');
    });
    frontend._ipc.on('service-get-keymap-reply' , (event, arg) => {
        frontend._clientconfig.mappings = JSON.parse(arg);
        if (frontend._clientconfig.mappings.hasOwnProperty("enabled")){
            document.getElementById("enablebump").checked = frontend._clientconfig.mappings.enabled;
            frontend._clientconfig.updateEnabled();
        }
        if (frontend._clientconfig.mappings.hasOwnProperty("KEY_CLOSE_ORDER")){
            document.getElementById("keycloseorder").value = frontend._clientconfig.mappings["KEY_CLOSE_ORDER"].KEY;
        }        
        if (frontend._clientconfig.mappings.hasOwnProperty("KEY_RECALL_ORDER")){
            document.getElementById("keyrecallorder").value = frontend._clientconfig.mappings["KEY_RECALL_ORDER"].KEY;
        }
        if (frontend._clientconfig.mappings.hasOwnProperty("KEY_CANCEL_ACTION")){
            document.getElementById("keycancelaction").value = frontend._clientconfig.mappings["KEY_CANCEL_ACTION"].KEY;
        }        
    });
    frontend._ipc.on('service-get-sound-reply' , (event, arg) => {    
        console.log(JSON.parse(arg));    
        frontend._clientconfig.sound = JSON.parse(arg);

        if (frontend._clientconfig.sound.hasOwnProperty("updateOrder")){
                document.getElementById("updateOrderSound").checked = frontend._clientconfig.sound.updateOrder.enabled;
                frontend._clientconfig.updateOrderSound();
        }
    });
    frontend._ipc.on('service-get-sms-reply' , (event, arg) => {        
        frontend._clientconfig.smsconfig = JSON.parse(arg);
        if (frontend._clientconfig.smsconfig.hasOwnProperty("enabled")){
            document.getElementById("enablesms").checked = frontend._clientconfig.smsconfig.enabled;
            frontend._clientconfig.updateSMSEnabled();
        }
        if (frontend._clientconfig.smsconfig.hasOwnProperty("twilio")){
            if (frontend._clientconfig.smsconfig.twilio.hasOwnProperty("ACCOUNT_SID")){
                document.getElementById("twilio_accountsid").value = frontend._clientconfig.smsconfig.twilio.ACCOUNT_SID;
            }        
            if (frontend._clientconfig.smsconfig.twilio.hasOwnProperty("AUTH_TOKEN")){
                document.getElementById("twilio_authtoken").value = frontend._clientconfig.smsconfig.twilio.AUTH_TOKEN;
            }
            if (frontend._clientconfig.smsconfig.twilio.hasOwnProperty("SENDING_NUMBER")){
                document.getElementById("twilio_sendingnumber").value = frontend._clientconfig.smsconfig.twilio.SENDING_NUMBER;
            }
            if (frontend._clientconfig.smsconfig.twilio.hasOwnProperty("MESSAGE")){
                document.getElementById("twilio_message").value = frontend._clientconfig.smsconfig.twilio.MESSAGE;
            }
        }
    });
};
module.exports = frontend;
