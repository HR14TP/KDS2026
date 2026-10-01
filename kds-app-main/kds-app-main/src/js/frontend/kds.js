'use strict';
const NATS = require('nats');
const KDSAJAX = require('./kdsajax.js');
const Toastify = require('toastify-js');
var {Howl } = require('howler');
const ipc = require('electron').ipcRenderer;
class NATS_Message {
	terminal_id = null;
    hwid = null;
    message = null;
    type = null;
    // screen/passmode identify the sending terminal - only read by
    // sub-order-close (natsCloseOrder in kds-webserver-main) so a close can
    // be scoped to this terminal's own screen, or to every screen assigned
    // to it if it's a pass terminal. null/false for every other message
    // type built with this class, which ignore them.
    screen = null;
    passmode = false;
    constructor(message = null, hwid = null, terminal_id = null, type = null, screen = null, passmode = false){
        this.message = message;
        this.hwid = hwid;
        this.terminal_id = terminal_id;
        this.type = type;
        this.screen = screen;
        this.passmode = passmode;
    };
    getJSON() {
        return {
            message: this.message,
            hwid:  this.hwid,
            terminal_id: this.terminal_id,
            type: this.type,
            screen: this.screen,
            passmode: this.passmode,
        }
    };
};

let KDS = {
	name: 'KDS',
	id: 'KDS',
	helper: {},
	nats: {},
	features: {},
	summary_open: {},
	summary_sub: {},
	summary_by_status: { notProduced: {}, inProduction: {}, completed: {} }
};
KDS.ws = null;
KDS.eventsource = null;
KDS.timerPID = null;
KDS.mintimerPID = null;
KDS.debug = false;
KDS.itemtimers = [];
KDS.productionTimers = [];
// Tracks the item ids already seen (on this screen) for each open order -
// keyed by order id, value a Set of item ids - purely so KDS.process_order
// can tell "this order already existed, but a new item was just added to
// it" apart from any other kind of update (a status tick, a production
// start/stop). See KDS.HTML_order_has_new_item, and the "youngest" item-time
// tracking in HTML_GENERATE_ORDER_ITEMS that the reset itself reads from.
KDS.orderItemIds = {};
// Full order data (including .items) keyed by id, rebuilt from every
// periodic update - lets KDS.close_order check for incomplete items from
// a board-tile double-click without needing the order-detail dialog open
// first (that path only ever populates KDS.activeOrderData, for whichever
// single order's dialog happens to be open).
KDS.orders = {};
// Counts item-completion requests (KDS.close_item) currently in flight to
// the backend. Checking a checkbox updates its visual state instantly (a
// browser-native checkbox click), but the /closeitem round trip that
// actually persists "COMPLETE" to the database takes real time - clicking
// Close Order right after checking the last item (an entirely normal thing
// to do) can otherwise land before that write has happened at all, so even
// re-fetching the order fresh (see KDS.close_order) still correctly reports
// it as still OPEN. KDS.close_order waits for this to drop back to 0 before
// evaluating anything, so it only ever judges "is this order actually done"
// once every check the user just made has actually landed.
KDS.pendingCloseItemRequests = 0;
// Screen names are matched case-insensitively everywhere on the board -
// backend tickets can land under whatever case a POS or manual entry
// happened to use ("Kitchen" vs "kitchen"), and a terminal's own configured
// screen name is free-typed too, so a bare === here would silently drop
// items that are really this terminal's own onto no board at all (see the
// matching kds-webserver-main fix, which makes the same names on the
// backend side).
KDS.screenMatches = (a, b) => (a || '').toLowerCase() === (b || '').toLowerCase();
KDS.lastrecalled = 0;
KDS.kdsconfig = {};
KDS.nc = null;
KDS.activemodal = false;
KDS.activemodalID = 0;
KDS.activeRecall = false;
KDS.reversed = false;
KDS.reversedminutes = 24;
KDS.late1 = 2;
KDS.late2 = 4;
KDS.features.prioritykeywordenabled = false;
KDS.features.prioritykeywordtext = '';
KDS.summary_counts = { notProduced: 0, inProduction: 0, completed: 0 };

KDS.features.priorityitemenabled = false;
KDS.features.priorityitemcontainertext1 = '';
KDS.features.priorityitemcontainertext2 = '';
KDS.features.priorityitemcontainertext3 = '';

KDS.init = function(config) {
	try {
		KDS.kdsconfig = config;
		document.title = KDS.kdsconfig.client.name + " KDS";
		kdsscreen = KDS.kdsconfig.client.screen;
		current_page = KDS.kdsconfig.client.screen;
		KDS.helper.update_value('screen_button' , KDS.kdsconfig.client.screen + " KDS");
		let root = document.documentElement;
		root.style.setProperty('--mainitemcolor', KDS.kdsconfig.colors.mainitem);
		root.style.setProperty('--subitemcolor', KDS.kdsconfig.colors.subitem);
		root.style.setProperty('--late1color', KDS.kdsconfig.colors.late1);
		root.style.setProperty('--late2color', KDS.kdsconfig.colors.late2);
		// How many tiles tall a column is when Fixed Tile Size (KDS Config)
		// is on - see wimpy.css's own --tile-fixed-rows fallback for why
		// this is only set here rather than also hardcoded there. Defaults
		// to 2 to match what Fixed Tile Size always did before this became
		// adjustable.
		root.style.setProperty('--tile-fixed-rows', KDS.kdsconfig.flags.hasOwnProperty('fixedtilerows') ? KDS.kdsconfig.flags.fixedtilerows : 2);
		// Most item/badge text is sized as a % of the inherited font-size
		// (see wimpy.css), so setting this on body scales the board's main
		// text up or down together, without needing every element rewritten.
		document.body.style.fontSize = (KDS.kdsconfig.flags.hasOwnProperty('fontsize') ? KDS.kdsconfig.flags.fontsize : 16) + 'px';
		// "Touch Interface" (KDS config) - opt-in larger buttons/checkboxes/
		// production controls for finger/glove use, vs. the original denser
		// layout. Off by default so existing installs keep their current
		// look until someone explicitly turns this on.
		document.body.classList.toggle('touchui', KDS.kdsconfig.flags.touchui === true);
		// Board is split into not-produced/in-production/completed sections
		// (see HTML_GENERATE_ORDER_ITEMS + wimpy.css) - a pass-mode terminal
		// needs to action already-completed orders first, so that section
		// goes leftmost there instead of rightmost everywhere else.
		document.body.classList.toggle('passmode-boardorder', KDS.flagcheck('passmode'));
		// Recalling completed orders only ever makes sense on a screen that
		// can actually close orders in the first place - a pass-mode
		// terminal, or a self-bumping screen (Standalone Mode, or any plain
		// screen with no pass terminal assigned) closing its own. Matches
		// KDS.canCloseOrderHere exactly, since recall is just the undo of
		// that same action.
		let recallOrdersColumn = document.getElementById('recallorderscolumn');
		if (recallOrdersColumn) {
			recallOrdersColumn.style.display = KDS.canCloseOrderHere() ? '' : 'none';
		}
		if (KDS.kdsconfig.colors.hasOwnProperty('reversecount')){
			KDS.reversed = KDS.kdsconfig.colors.reversecount;
			if (KDS.reversed) {
				if (KDS.kdsconfig.colors.hasOwnProperty('reversecounttime')){
					KDS.reversedminutes = KDS.kdsconfig.colors.reversecounttime;
				}
			}
		}
		if (KDS.kdsconfig.colors.hasOwnProperty('late1time')){
			KDS.late1 = KDS.kdsconfig.colors.late1time;
		}
		if (KDS.kdsconfig.colors.hasOwnProperty('late2time')){
			KDS.late2 = KDS.kdsconfig.colors.late2time;
		}
		
		if (KDS.kdsconfig.hasOwnProperty('kdsfeatures')){
			if (KDS.kdsconfig.kdsfeatures.hasOwnProperty('prioritykeywordenabled')){
				KDS.features.prioritykeywordenabled = KDS.kdsconfig.kdsfeatures.prioritykeywordenabled;				
			}
			if (KDS.kdsconfig.kdsfeatures.hasOwnProperty('prioritykeywordtext')){				
				KDS.features.prioritykeywordtext = KDS.kdsconfig.kdsfeatures.prioritykeywordtext;
			}

			if (KDS.kdsconfig.kdsfeatures.hasOwnProperty('c')){
				KDS.features.priorityitemenabled = KDS.kdsconfig.kdsfeatures.priorityitemenabled;				
			}
			if (KDS.kdsconfig.kdsfeatures.hasOwnProperty('priorityitemcontainertext1')){				
				KDS.features.priorityitemcontainertext1 = KDS.kdsconfig.kdsfeatures.priorityitemcontainertext1;
			}
			if (KDS.kdsconfig.kdsfeatures.hasOwnProperty('priorityitemcontainertext2')){				
				KDS.features.priorityitemcontainertext2 = KDS.kdsconfig.kdsfeatures.priorityitemcontainertext2;
			}
			if (KDS.kdsconfig.kdsfeatures.hasOwnProperty('priorityitemcontainertext3')){				
				KDS.features.priorityitemcontainertext3 = KDS.kdsconfig.kdsfeatures.priorityitemcontainertext3;
			}
		}

		KDS.per_second_timer();
		KDS.ajax_startup();
		dialog.init();
		KDS.nats.init();
		//toast-message
		ipc.on('toast-message' , (event, arg) => {
			try{
				let m = JSON.parse(arg);			
				KDS.show_toast(m.message);
			}catch(e) {
				console.log(e);
			}
		});

	}
	catch(err){
		alert("Error : " + err);
	}
};

KDS.show_summary = function(){
	var elem = document.getElementById("summary");
	elem.style.width = '400px';
	elem.style.paddingLeft = '10px';
	KDS.render_summary();
};
KDS.render_summary = function(){
	var countselem = document.getElementById("summary_counts");
	if (countselem) {
		countselem.innerHTML =
			'<div class="sectionheading sectionheading-notproduced">' + KDS.summary_counts.notProduced + ' NOT PRODUCED</div>' +
			'<div class="sectionheading sectionheading-inprod">' + KDS.summary_counts.inProduction + ' IN PRODUCTION</div>' +
			'<div class="sectionheading sectionheading-complete">' + KDS.summary_counts.completed + ' COMPLETED</div>';
	}

	var thehtml = '';
	if (KDS.flagcheck('summarybystatus')) {
		// Same three groups/colors as the board's own per-order production
		// sections (see HTML_GENERATE_ORDER_ITEMS) - a kitchen-wide item
		// only shows up under the heading matching its current state, and
		// any sub-items sit right under their own parent item's row rather
		// than in a separate, disconnected sub-items list.
		var groups = [
			{ key: 'notProduced', label: 'NOT PRODUCED', headingClass: 'sectionheading-notproduced' },
			{ key: 'inProduction', label: 'IN PRODUCTION', headingClass: 'sectionheading-inprod' },
			{ key: 'completed', label: 'COMPLETED', headingClass: 'sectionheading-complete' }
		];
		groups.forEach((group) => {
			var bucket = KDS.summary_by_status[group.key];
			var entries = Object.entries(bucket);
			if (entries.length === 0) {
				return;
			}
			thehtml += '<div class="sectionheading ' + group.headingClass + '">' + group.label + '</div>';
			entries.forEach(([name, value]) => {
				thehtml += '<div class="row"><div class="column doleft column-100 sumentry">' + value.qty + ' X ' + name + '</div></div>';
				Object.entries(value.sub).forEach(([subName, subValue]) => {
					thehtml += '<div class="row"><div class="column doleft column-100 subentry">' + subValue.qty + ' X ' + subName + '</div></div>';
				});
			});
		});
	} else {
		// Original flat layout - not split by production status, main
		// items and their sub-items listed as two separate, unconnected
		// groups.
		thehtml += '<div class="row"><div class="column"><strong style="font-size: 22px;">MAIN ITEMS</strong></div></div>';
		for (const [name, value] of Object.entries(KDS.summary_open)) {
			thehtml += '<div class="row"><div class="column doleft column-100 sumentry">' + value.qty + ' X ' + name + '</div></div>';
		}
		thehtml += '<div class="row"><div class="column"><strong style="font-size: 22px;">SUB ITEMS</strong></div></div>';
		for (const [name, value] of Object.entries(KDS.summary_sub)) {
			thehtml += '<div class="row"><div class="column doleft column-100 subentry">' + value.qty + ' X ' + name + '</div></div>';
		}
	}

	var elem = document.getElementById("main_summary");
	elem.innerHTML = thehtml;
};
KDS.hide_summary = function(){
	var elem = document.getElementById("summary");
	elem.style.width = '0px';
	elem.style.paddingLeft = '0px';
};

KDS.hideDialog = function() {
	KDS.activemodal = false;
	KDS.activemodalID = 0;
	KDS.activeRecall = false;
	dialog.hide();
};

// A board tile's onclick and ondblclick both fire on a real double-click
// (the browser sends click, click, dblclick in sequence) - without this,
// double-clicking a tile to close it would also pop the order dialog open
// twice first, which close_order's own KDS.hideDialog() then immediately
// closes again: a visible flash rather than a clean double-tap-to-close.
// Delaying the single-click's dialog open just long enough to see whether
// a second click follows, and skipping it when one does, is the standard
// fix for this on any element that needs to distinguish click from
// dblclick.
KDS.tileClickTimer = null;
KDS.tileClickId = null;
KDS.handleTileClick = function(id) {
	if (KDS.tileClickTimer !== null) {
		clearTimeout(KDS.tileClickTimer);
		KDS.tileClickTimer = null;
		if (KDS.tileClickId === id) {
			// Second click of a double-click on this same tile - let
			// ondblclick handle it, the dialog should never open.
			KDS.tileClickId = null;
			return;
		}
	}
	KDS.tileClickId = id;
	KDS.tileClickTimer = setTimeout(function() {
		KDS.tileClickTimer = null;
		KDS.tileClickId = null;
		KDS.prompt_dialog(id);
	}, 280);
};

// Collapsing a tile only hides its own item list (.tile-collapsible-body)
// - check/table/waiter/timer stay visible either way, since that's what
// someone scanning a busy board actually needs at a glance. The class
// lives on elem_<id> itself (not on the items list that gets replaced
// every update), so it survives every KDS.HTML_GENERATE_ORDER_ITEMS
// re-render without needing to be restored.
KDS.toggleTileCollapse = function(event, id) {
	event.stopPropagation();
	let toggleButton = document.getElementById('tiletoggle-' + id);
	if (KDS.hasClass('elem_' + id, 'tile-collapsed')) {
		KDS.removeClass('elem_' + id, 'tile-collapsed');
		if (toggleButton) { toggleButton.textContent = 'Collapse Order'; }
	} else {
		KDS.addClass('elem_' + id, 'tile-collapsed');
		if (toggleButton) { toggleButton.textContent = 'Expand Order'; }
	}
};

KDS.prompt_dialog = function(id) {
	KDS.getSingleOrder(id);
	KDS.activemodalID = id;
	KDS.activemodal = true;
	var html = '<div class="dialogtopbar">';
	html += '<span class="dialogheading">Close Order / Overview</span>';
	html += '<button onclick="KDS.hideDialog()" class="button button-black">CLOSE THIS SCREEN</button>';
	html += '</div>';
	html += '<div class="row orderdialogbody"><div class="column orderdialogitems">';
	html += '<div id="makesingle"></div></div>';
	// A pass-mode terminal can always close the whole order. A non-pass
	// screen only gets this button when nobody else has been assigned to
	// bump it (see KDS.canCloseOrderHere) - in that case it only ever bumps
	// its own items, never another screen's, so it's exactly as safe as the
	// per-item checkboxes it already has.
	if (KDS.canCloseOrderHere()) {
		html += '<div class="column orderdialogclose">';
		html += '<button onclick="KDS.close_order(' + id + ')" class="closeorderbutton">CLOSE ORDER</button></div>';
	}
	html += '</div>';
	dialog.showcustom(html);
	// Scales with the actual window/screen instead of a fixed pixel box, so
	// this looks right whether KDS is running on a small kiosk touchscreen
	// or a full desktop monitor.
	dialog.resizeRelative(92, 88, 1400, 900);
};

KDS.messages_closed = function(js, kdsscreen) {
	var id = Number(js.data.data);
	var screenie = js.data.messagetype;
	//TODO : Check all screens
	if (KDS.screenMatches(kdsscreen, screenie) || KDS.flagcheck("enableallcenters")) {
		try {
			if (undefined !== document.getElementById('elem_' + id) && null !== document.getElementById('elem_' + id)) {
				document.getElementById('elem_' + id).remove();
			}
		} catch (e) {
			KDS.log(e);
		}
	}
};

// ========== AJAX 
KDS.getSingleOrder = function(id) {
	KDS.ajax_getsingleorder(id);
};

KDS.ajax_startup = function(id) {
    var xmlHttp = new XMLHttpRequest();    
    xmlHttp.open("GET", "http://127.0.0.1:2020/updated", true); // true for asynchronous 
    xmlHttp.send(null);
	setTimeout(function(){ var xmlHttp = new XMLHttpRequest();    
		xmlHttp.open("GET", "http://127.0.0.1:2020/updated", true); // true for asynchronous 
		xmlHttp.send(null); 
	}, 3000);
};

KDS.ajax_getsingleorder = function(id) {
    var xmlHttp = new XMLHttpRequest();
    xmlHttp.onreadystatechange = function() { 
        if (xmlHttp.readyState === 4 && xmlHttp.status === 200){
			var js = JSON.parse(xmlHttp.responseText);
			if (undefined !== js){
				KDS.message_single_generate(js);
			}			
		}
    };
    xmlHttp.open("GET", "http://127.0.0.1:2020/getsingleorder/" + id, true); // true for asynchronous 
    xmlHttp.send(null);
};

KDS.ajax_messages_getcompleted = function() {
	var xmlHttp = new XMLHttpRequest();
	xmlHttp.onreadystatechange = function() {
		if (xmlHttp.readyState === 4 && xmlHttp.status === 200){
			var js = JSON.parse(xmlHttp.responseText);
			if (undefined !== js){
				KDS.generate_completed(js);
			}
		}
	};
	xmlHttp.open("GET", "http://127.0.0.1:2020/getcompletedorders/" + kdsscreen, true); // true for asynchronous
	xmlHttp.send(null);
};

// Passmode is sent along so the backend can tell a pass terminal's Recall
// Orders (reopen the whole order, every screen - same "whole ticket"
// semantics as its Close Order force-close) apart from a normal terminal's
// own recall (scoped to just its own screen) - and so it can fan the recall
// out to every other terminal the order originally showed on, not just
// reopen this terminal's own copy. See recallorderHandler in
// kds-webserver-main. Standalone Mode deliberately does NOT get force-close
// treatment here, same as KDS.nats.close_order - see its comment: a
// standalone terminal is only guaranteed to be the sole owner of ITS OWN
// screen, not of every screen on-site.
KDS.message_recallorder = function(id) {
	var xmlHttp = new XMLHttpRequest();
	xmlHttp.onreadystatechange = function() {
		if (xmlHttp.readyState === 4 && xmlHttp.status === 200){
			console.log(xmlHttp.responseText);
		}
	};
	xmlHttp.open("GET", "http://127.0.0.1:2020/recallorder/" + id + "/" + kdsscreen + "/" + KDS.flagcheck("passmode"), true); // true for asynchronous
	xmlHttp.send(null);
};



KDS.prompt_completed = function() {
	// See KDS.canCloseOrderHere - recalling only makes sense on a screen
	// that can actually close orders in the first place (pass mode, or
	// self-bumping in Standalone Mode/with no pass terminal assigned).
	// This also blocks the bumpbar KEY_RECALL_ORDER hotkey on any screen
	// where the button itself is hidden.
	if (!KDS.canCloseOrderHere()) {
		KDS.log("prompt_completed blocked: this screen cannot close orders here, so it has nothing to recall");
		return;
	}
	KDS.ajax_messages_getcompleted();
	var html = '<h4>Completed Orders Last 30 Minutes</h4>';
	html += '<div class="row doleft">';
	html += '<div class="column">';
	html += '<button onclick="KDS.hideDialog()" class="button button-black doleft">CLOSE THIS SCREEN</button>';
	html += '<div class="doleft lds-hourglass" id="completedorders">';
	//html += '<div class="completedentry"><strong>Waiter - Table 12 - Chk 12 - Restaurant</strong><br/><i>[1 x Breakfast abc, 1 x Breakfast abc, 1 x Breakfast abc]</i></div>';
	html += '</div>';
	html += '<div style="clear:both;"></div>';
	html += '</div>';
	html += '</div>';
	dialog.showcustom(html);
	// Every other dialog.showcustom caller in this file also calls a
	// dialog.resize* function right after - this one didn't, so the popup
	// box was left at whatever size/position the PREVIOUS dialog happened
	// to leave it at (e.g. the order detail dialog's own 92vw/88vh, or the
	// small fixed-pixel box from dialog.resize()/resizeXY elsewhere) instead
	// of ever sizing itself. A tall completed-orders list stuffed into a
	// leftover small/oddly-positioned box reads as the dialog's own header
	// and close button cutting off at the top. Same relative sizing as the
	// order detail dialog so it's consistent and always correctly placed
	// regardless of whatever was open before it.
	dialog.resizeRelative(92, 88, 1400, 900);
};

KDS.generate_completed = function(js) {
	var html = '';
	if (Array.isArray(js)){
		js.forEach((item) => {
			html +=
				'<div class="completedentry"><strong>' +
				item.waiter +
				' - ' +
				item.table +
				' - CHK ' +
				item.order_forday +
				' - ' +
				item.profile +
				'</strong><a class="button button-dark-blue margleft doright" href="javascript:KDS.click_restore_order(' +
				item.id +
				')">RESTORE</a><br/>';
			html += '<i>[';
			item.items.forEach((mainitem) => {
				html += mainitem.qty + ' x ' + mainitem.name + ', ';
			});
			html += ']</i></div>';
		});
		KDS.removeClass('completedorders', 'lds-hourglass');
		KDS.updateHTML('completedorders', html);
	}
	KDS.activeRecall = true;
};

KDS.click_restore_order = function(id) {
	KDS.message_recallorder(id);
	KDS.hideDialog();	
};

KDS.close_item = function(id , orderid) {
	// Only the pass mode terminal can mark items as complete - prevents
	// inadvertent marking on non-pass terminals in a multi-terminal setup.
	if (!KDS.flagcheck("passmode")) {
		alert("Only the pass terminal can mark items as complete.");
		return;
	}
	var element = document.getElementById('close_item_' + id );
	var ischecked = element.checked;
	console.log(ischecked);
	if (ischecked == null || undefined == ischecked) {
		ischecked = false;
	}
	var xmlHttp = new XMLHttpRequest();
	KDS.pendingCloseItemRequests++;
	if (ischecked) {
		// Item is being struck out as produced - stop its production timer,
		// freezing the displayed elapsed time at the moment it was checked.
		KDS.removeProductionTimer(id);
		xmlHttp.onreadystatechange = function() {
			if (xmlHttp.readyState == 4){
				KDS.pendingCloseItemRequests = Math.max(0, KDS.pendingCloseItemRequests - 1);
			}
			if (xmlHttp.readyState == 4 && xmlHttp.status == 200){
				if (xmlHttp.responseText === "CLOSE") {
					try {
						//order was closed , same logic must fire.
						document.getElementById('elem_' + orderid).remove();
						document.getElementById('index_' + orderid).remove();
					} catch (e) {
						KDS.log(e);
					}
					finally {
						KDS.removeTimer(orderid);
						KDS.activemodal = false;
						KDS.activemodalID = 0;
						KDS.hideDialog();
					}
				} else {
					KDS.nats.update_order();
					KDS.refreshSingleOrderDialog(orderid);
				}

			}
		};
		xmlHttp.open("GET", "http://127.0.0.1:2020/closeitem/" + id, true); // true for asynchronous
		xmlHttp.send(null);
	} else {
		console.log("opening item");
		xmlHttp.onreadystatechange = function() {
			if (xmlHttp.readyState == 4){
				KDS.pendingCloseItemRequests = Math.max(0, KDS.pendingCloseItemRequests - 1);
			}
			if (xmlHttp.readyState == 4 && xmlHttp.status == 200){
				KDS.nats.update_order();
				KDS.refreshSingleOrderDialog(orderid);
				// Reopening never resumes production by itself - ask, rather
				// than silently marking it back in production for the user.
				if (confirm('Start production again on this item?')) {
					KDS.mark_production(id);
				}
			}
		};
		xmlHttp.open("GET", "http://127.0.0.1:2020/openitem/" + id, true); // true for asynchronous
		xmlHttp.send(null);
	}
};

KDS.mark_production = function(id) {
	var xmlHttp = new XMLHttpRequest();
	xmlHttp.onreadystatechange = function() {
		if (xmlHttp.readyState === 4 && xmlHttp.status === 200){
			KDS.nats.update_order();
			KDS.refreshSingleOrderDialog(KDS.activemodalID);
		}
	};
	xmlHttp.open("GET", "http://127.0.0.1:2020/startproduction/" + id, true);
	xmlHttp.send(null);
};

KDS.clear_production = function(id) {
	// Stop the timer immediately rather than waiting on the round trip below,
	// same as the optimistic freeze in KDS.close_item.
	KDS.removeProductionTimer(id);
	var xmlHttp = new XMLHttpRequest();
	xmlHttp.onreadystatechange = function() {
		if (xmlHttp.readyState === 4 && xmlHttp.status === 200){
			KDS.nats.update_order();
			KDS.refreshSingleOrderDialog(KDS.activemodalID);
		}
	};
	xmlHttp.open("GET", "http://127.0.0.1:2020/clearproduction/" + id, true);
	xmlHttp.send(null);
};

// Marks a completed item as physically carried out to the table - moves it
// from the COMPLETED section into its own SENT TO TABLE section at the
// bottom of the tile (see HTML_GENERATE_ORDER_ITEMS) once the update round
// trips back through KDS.nats.update_order, same flow as every other
// per-item action here (mark_production/clear_production/close_item).
KDS.send_to_table = function(id) {
	// Only the pass mode terminal can send an item to the table - same
	// restriction, same reasoning as KDS.close_item's own passmode check.
	// The backend enforces this independently (senditemtotable rejects a
	// non-pass terminal's call outright) - this is just the matching UI
	// courtesy so a section terminal gets an immediate explanation instead
	// of a silently-failed request.
	if (!KDS.flagcheck("passmode")) {
		alert("Only the pass terminal can send an item to the table.");
		return;
	}
	var xmlHttp = new XMLHttpRequest();
	xmlHttp.onreadystatechange = function() {
		if (xmlHttp.readyState === 4 && xmlHttp.status === 200){
			KDS.nats.update_order();
		}
	};
	xmlHttp.open("GET", "http://127.0.0.1:2020/senditemtotable/" + id, true);
	xmlHttp.send(null);
};

// Inline status control/badge appended next to an item's name in the item
// detail dialog (message_single_generate): a "START" button while OPEN and
// not yet in production, a clickable orange "IN PROD x" tag once started
// (click to clear it, back to the START button), and a green "COMPLETE" tag
// once the item has been ticked off as produced in KDS.close_item.
KDS.HTML_production_control = function(mainitem) {
	// Master switch (KDS config -> Production Mode): with it off, starting/
	// clearing production isn't a concept this KDS exposes at all - no
	// button, no in-progress tag, no timer, nothing to click.
	if (KDS.kdsconfig.flags.productionmode === false) {
		return '';
	}
	if (mainitem.status === 'COMPLETE') {
		return ' <span class="production-tag production-tag-complete" title="Marked as produced">COMPLETE</span>';
	}
	if (mainitem.status !== 'OPEN') {
		return '';
	}
	if (mainitem.in_production === true) {
		return ' <span class="production-tag production-tag-clickable" onclick="event.stopPropagation();KDS.clear_production(' + mainitem.id + ');" title="Click to remove production marker">IN PROD &#10005;</span>';
	}
	return ' <button type="button" class="button button-small start-production-btn" onclick="event.stopPropagation();KDS.mark_production(' + mainitem.id + ');">START</button>';
};

// Live "time in production" counter, shown both next to an item in the item
// detail dialog (id "prodtimer-<id>") and on its main-board tile (id
// "prodtimer-main-<id>") once it has actually been marked in production via
// the START button (KDS.mark_production). Seeded from the server's
// production_started_at timestamp - which the backend stamps fresh on every
// /startproduction call and clears on /clearproduction and on completion -
// so starting, clearing, and restarting production naturally resets the
// timer each time, with no client-side bookkeeping needed. Ticks up every
// second via KDS.per_second_timer/KDS.updateProductionTimers (which updates
// both copies), and is frozen (via KDS.removeProductionTimer) once the item
// is either checked off as produced (KDS.close_item) or taken out of
// production (KDS.clear_production).
KDS.HTML_production_timer = function(mainitem, idPrefix) {
	idPrefix = idPrefix || 'prodtimer-';
	if (KDS.kdsconfig.flags.productionmode === false) {
		return '';
	}
	if (mainitem.status !== 'OPEN' || mainitem.in_production !== true || !mainitem.production_started_at) {
		return '';
	}
	if (KDS.productionTimers.indexOf(mainitem.id) === -1) {
		KDS.productionTimers.push(mainitem.id);
	}
	// Full MM:SS elapsed, not just minutes - both the dialog and the board
	// tile can be re-rendered by acting on a completely different item in the
	// same order (any tick/untick/start/clear refreshes the whole order), and
	// a seconds-less seed would visibly snap every other item's timer back to
	// ":00" on every such refresh, looking like they'd all been reset too.
	var elapsed = KDS.time_difference_now_full(Date.parse(KDS.fixDate(mainitem.production_started_at)));
	return ' <span id="' + idPrefix + mainitem.id + '" class="production-timer" title="Time in production">' + elapsed + '</span>';
};

KDS.close_order = function(id) {
	// See KDS.canCloseOrderHere - this also blocks the bumpbar
	// KEY_CLOSE_ORDER hotkey on any screen where the button itself is
	// hidden.
	if (!KDS.canCloseOrderHere()) {
		KDS.log("close_order blocked: this section's orders are closed by its assigned pass terminal");
		return;
	}
	// Standalone Mode already guarantees passmode is off and no pass
	// terminal is assigned (see ApplyStandaloneMode/saveConfig in
	// frontend.js), so this KDS only ever bumps its own orders - the
	// "close anyway?" prompt exists to stop a pass terminal from closing
	// out items it can't see, which can't happen here, so skip it
	// entirely and just bump.
	if (KDS.flagcheck("standalonemode")) {
		KDS.finishCloseOrder(id);
		return;
	}
	// Bumping an order out from under the kitchen while it still has
	// unfinished items is usually a mistake - ask once, rather than
	// silently closing out food that was never actually marked done.
	// Wait for any of this terminal's own item-completion requests still in
	// flight (see KDS.pendingCloseItemRequests) before checking anything -
	// otherwise even a fresh fetch immediately below can correctly report
	// an item as still OPEN simply because its own /closeitem write hasn't
	// reached the database yet, not because it's actually unfinished.
	KDS.waitForPendingCloseItems(function() {
		KDS.evaluateAndCloseOrder(id);
	});
};

// Polls KDS.pendingCloseItemRequests down to 0 (checked every 100ms, capped
// at 3s so a genuinely stuck/failed request can't wedge Close Order
// forever) before calling back - see KDS.close_order.
KDS.waitForPendingCloseItems = function(callback, elapsedMs) {
	elapsedMs = elapsedMs || 0;
	if (KDS.pendingCloseItemRequests <= 0 || elapsedMs >= 3000) {
		callback();
		return;
	}
	setTimeout(function() {
		KDS.waitForPendingCloseItems(callback, elapsedMs + 100);
	}, 100);
};

// Re-fetches the order straight from the database rather than trusting
// KDS.activeOrderData/KDS.orders[id] - both are only as current as the last
// periodic "updates" push, which lags behind a just-submitted close_item
// click by however long that round trip takes. Combined with
// KDS.waitForPendingCloseItems above (which this runs after), this is what
// actually guarantees an accurate "N items not complete" count on a Close
// Order click made immediately after ticking the last box.
KDS.evaluateAndCloseOrder = function(id) {
	var xmlHttp = new XMLHttpRequest();
	xmlHttp.onreadystatechange = function() {
		if (xmlHttp.readyState !== 4) {
			return;
		}
		if (xmlHttp.status === 200) {
			try {
				var orderData = JSON.parse(xmlHttp.responseText);
			} catch (e) {
				KDS.log(e);
				KDS.finishCloseOrder(id);
				return;
			}
			if (orderData) {
				// A pass terminal covers its own screen plus every sub-terminal
				// screen assigned to it (screen.assigned_pass_terminal, set via
				// the Client Config "Pass Terminal" dropdown on each section
				// screen) - it must never count items belonging to a completely
				// separate terminal (e.g. Bar) just because they're on the same
				// order. A self-bumping screen only ever affects its own items,
				// so the count (and the "close anyway?" decision it drives) is
				// scoped the same way the board itself already filters what
				// this screen displays.
				var itemsInScope = orderData.items.filter((mainitem) => {
					if (!mainitem.screen) { return false; }
					if (KDS.screenMatches(mainitem.screen.name, KDS.kdsconfig.client.screen)) { return true; }
					return KDS.flagcheck("passmode") && KDS.screenMatches(mainitem.screen.assigned_pass_terminal, KDS.kdsconfig.client.screen);
				});
				var incomplete = itemsInScope.filter((mainitem) => mainitem.status === 'OPEN').length;
				if (incomplete > 0) {
					var plural = incomplete === 1 ? 'item is' : 'items are';
					if (!confirm(incomplete + ' ' + plural + ' not marked as complete. Close this order anyway?')) {
						return;
					}
				}
			}
		} else {
			// Fetch failed outright - fall back to whatever's cached rather
			// than silently doing nothing on a Close Order click.
			KDS.log("close_order: fresh fetch failed (status " + xmlHttp.status + "), falling back to cached data");
		}
		KDS.finishCloseOrder(id);
	};
	xmlHttp.open("GET", "http://127.0.0.1:2020/getsingleorder/" + id, true);
	xmlHttp.send(null);
};

// Actually closes the order once KDS.close_order has decided (immediately
// in Standalone Mode, or after the "close anyway?" confirm elsewhere) that
// it should proceed - separated out so both paths share the exact same
// tail instead of duplicating it.
KDS.finishCloseOrder = function(id) {
	// No optimistic local removal of the tile/timer here anymore - a close
	// from here no longer guarantees the whole order is now done (another
	// screen may still have open items), so the tile has to stay until the
	// backend's own "updates" push (fired at the end of natsCloseOrder,
	// arrives well before this) says otherwise, the same way every other
	// real-time change already reaches the board.
	KDS.activemodal = false;
	KDS.activemodalID = 0;
	KDS.hideDialog();
	KDS.nats.close_order(id);
};

KDS.something_wrong = function() { 
	location.reload();
};

// ============= END AJAX


// ============================= DRAW ORDERS FUNCTIONS ==================== //


/* ******************************** BUMPBAR ACTIONS *********************** */
KDS.timeOutID = 0;
KDS.keyBuffer = '';

KDS.keypress = function(event){
	clearTimeout(KDS.timeOutID);
	//Step1 : Check KDS Config
	//console.log(KDS.kdsconfig);
	if (KDS.kdsconfig.hasOwnProperty('bumpbars')){
		if (KDS.kdsconfig.bumpbars.hasOwnProperty('mapping')){
			if (KDS.kdsconfig.bumpbars.mapping.hasOwnProperty('enabled')) {
				if (KDS.kdsconfig.bumpbars.mapping.enabled){
					//console.log("ENABLED");
					if (KDS.kdsconfig.bumpbars.mapping.KEY_RECALL_ORDER.KEY !== "" && 
					event.key.toLowerCase() === KDS.kdsconfig.bumpbars.mapping.KEY_RECALL_ORDER.KEY.toLowerCase() && 
					event.shiftKey === KDS.kdsconfig.bumpbars.mapping.KEY_RECALL_ORDER.SHIFT && 
					event.ctrlKey === KDS.kdsconfig.bumpbars.mapping.KEY_RECALL_ORDER.CONTROL && 
					event.altKey === KDS.kdsconfig.bumpbars.mapping.KEY_RECALL_ORDER.ALT){
						//KEY_RECALL_ORDER
						//console.log('KEY_RECALL_ORDER');
						KDS.keyBuffer = '';
						KDS.prompt_completed();
					} else if (KDS.kdsconfig.bumpbars.mapping.KEY_CLOSE_ORDER.KEY !== "" && 
					event.key.toLowerCase() === KDS.kdsconfig.bumpbars.mapping.KEY_CLOSE_ORDER.KEY.toLowerCase() && 
					event.shiftKey === KDS.kdsconfig.bumpbars.mapping.KEY_CLOSE_ORDER.SHIFT && 
					event.ctrlKey === KDS.kdsconfig.bumpbars.mapping.KEY_CLOSE_ORDER.CONTROL && 
					event.altKey === KDS.kdsconfig.bumpbars.mapping.KEY_CLOSE_ORDER.ALT){
						//KEY_RECALL_ORDER
						console.log('KEY_CLOSE_ORDER');
						KDS.keyBuffer = '';
						if (KDS.activemodal) {
							KDS.close_order(KDS.activemodalID);
							// location.reload();
						}
					} else if (KDS.kdsconfig.bumpbars.mapping.KEY_CANCEL_ACTION.KEY !== "" && 
					event.key.toLowerCase() === KDS.kdsconfig.bumpbars.mapping.KEY_CANCEL_ACTION.KEY.toLowerCase() && 
					event.shiftKey === KDS.kdsconfig.bumpbars.mapping.KEY_CANCEL_ACTION.SHIFT && 
					event.ctrlKey === KDS.kdsconfig.bumpbars.mapping.KEY_CANCEL_ACTION.CONTROL && 
					event.altKey === KDS.kdsconfig.bumpbars.mapping.KEY_CANCEL_ACTION.ALT){
						//KEY_RECALL_ORDER
						//console.log('KEY_CANCEL_ACTION');
						KDS.keyBuffer = '';
						if (KDS.activemodal) {
							KDS.hideDialog()
						} else if (KDS.activeRecall) {
							KDS.hideDialog();
						}
					} else {
						KDS.log(event);
						KDS.keyBuffer += event.key;
						KDS.timeOutID = setTimeout(function(){
							KDS.keytimeout(event);
						}, 1000);
					}
				}
			}
		}
	}
	
	/*if (event.key === "b" || event.key === "B"){
		KDS.keyBuffer = '';
		KDS.prompt_completed();
	} else if (event.key === "a" || event.key === "a"){
		KDS.keyBuffer = '';
		if (KDS.activemodal) {
			KDS.close_order(KDS.activemodalID);
		}
	} else if (event.key === "c" || event.key === "C"){
		KDS.keyBuffer = '';
		if (KDS.activemodal) {
			KDS.hideDialog()
		} else if (KDS.activeRecall) {
			KDS.hideDialog();
		}
	} else {
		KDS.log(event);
		KDS.keyBuffer += event.key;
		KDS.timeOutID = setTimeout(function(){
			KDS.keytimeout(event);
		}, 1000);
	}*/
};

KDS.keytimeout = function(event) {
	let divcount = 0;
	let elems = null;
	if (KDS.activeRecall) {
		elems = document.getElementById('completedorders').childNodes;
		KDS.log(elems);
		elems.forEach(function(item) {
			if (item.nodeName === "DIV") {
				divcount += 1;
				if (divcount === parseInt(KDS.keyBuffer)) {
					item.childNodes.forEach(function(subitem) {
						if (subitem.innerText === "RESTORE") {
							subitem.click();
						}
					});
				}
			}
		});
	} else {
		elems = document.getElementById('cardss').childNodes;
		KDS.log(elems);
		elems.forEach(function(item) {
			if (item.nodeName === "DIV") {
				divcount += 1;
				if (divcount === parseInt(KDS.keyBuffer)) {				
					var thedivtoclick = item.id.replace("index_", "");
					document.getElementById("elem_"+parseInt(thedivtoclick)).click();
				}
			}
		});
	}
	KDS.keyBuffer = '';
};
/* ******************************** END BUMPBAR ACTIONS *********************** */


KDS.update_preptime = function(js) {
	var t = js.data.avgmins + 'M:' + js.data.avgsecs + 'S';
	KDS.updateHTML('preptimebutton', t);
};

// ================== TOAST ================== //

KDS.show_toast = function(message) {
	try {
		Toastify({
			text: message,
			duration: 5000,        
			newWindow: true,
			close: true,
			gravity: "top", // `top` or `bottom`
			position: "right", // `left`, `center` or `right`
			backgroundColor: "linear-gradient(to right, ##33b5e5, #4285f4)",
			stopOnFocus: true, // Prevents dismissing of toast on hover
			onClick: function(){} // Callback after click
		  }).showToast();
	} catch (error) {
		KDS.log("Toastify Notification Failed" . error);
	}

};

// END TOASTS ================================ //


// ============================= TIMERS ================= //
KDS.set_system_time = function() {
	var date = new Date();
	var hours = date.getHours();
	var minutes = date.getMinutes();
	var seconds = date.getSeconds();
	//var ampm = hours >= 12 ? 'pm' : 'am';
	//hours = hours % 12;
	//hours = hours ? hours : 12; // the hour '0' should be '12'
	minutes = minutes < 10 ? '0' + minutes : minutes;
	seconds = seconds < 10 ? '0' + seconds : seconds;
	//var strTime = hours + ':' + minutes + ':' + seconds + ' ' + ampm;
	var strTime = hours + ':' + minutes + ':' + seconds;
	KDS.updateHTML('system_time', strTime);
};
KDS.removeTimer = function(id) {
	var index = KDS.itemtimers.indexOf(id);
	if (index > -1) {
		KDS.itemtimers.splice(index, 1);
	}
};
KDS.removeProductionTimer = function(id) {
	var index = KDS.productionTimers.indexOf(id);
	if (index > -1) {
		KDS.productionTimers.splice(index, 1);
	}
};
KDS.updateProductionTimers = function(timers) {
	timers.forEach((value) => {
		// An item's timer can be showing in both the dialog and its
		// main-board tile at once, under different DOM ids - tick both
		// independently; each has its own innerHTML, so this can't cross
		// items or the two copies up with each other.
		['prodtimer-', 'prodtimer-main-'].forEach((idPrefix) => {
			var element = document.getElementById(idPrefix + value);
			if (null !== element && undefined !== element) {
				const combined = element.innerHTML.split(':');
				var secs = Number(combined[1]);
				var mins = Number(combined[0]);
				if (secs < 59) {
					secs += 1;
				} else {
					secs = 0;
					mins += 1;
				}
				if (secs < 10) {
					secs = '0' + secs;
				}
				if (mins < 10) {
					mins = '0' + mins;
				}
				element.innerHTML = mins + ':' + secs;
			}
		});
	});
};
KDS.upateitemstimers = function(timers) {
	
	timers.forEach((value) => {
		var element = document.getElementById('timer-' + value);
		var mainid = 'elem_' + value;						
		if (null !== element && undefined !== element) {
			if (!KDS.reversed){
				// Normal Logic
				const combined = element.innerHTML.split(':');
				var secs = Number(combined[1]);
				var mins = Number(combined[0]);

				if (secs < 59) {
					secs += 1;
				} else {
					secs = 0;				
					mins += 1;
				}

				let late1 = 8;
				let late2 = 10;

				if (KDS.kdsconfig.colors.hasOwnProperty('late1time')){
					late1 = KDS.kdsconfig.colors.late1time;
				}
				if (KDS.kdsconfig.colors.hasOwnProperty('late2time')){
					late2 = KDS.kdsconfig.colors.late2time;
				}

				if (mins >= late1 && mins <= late2) {
					//console.log('A');
					if (KDS.hasClass(mainid, 'orderyellow') == false) {
						KDS.addClass(mainid, 'orderyellow');
					}
				} else if (mins > late2) {
					//console.log('B');
					if (KDS.hasClass(mainid, 'orderyellow') == true) {
						KDS.removeClass(mainid, 'orderyellow');
					}
					if (KDS.hasClass(mainid, 'orderlate2') == false) {
						KDS.addClass(mainid, 'orderlate2');
					}
				}

				if (secs < 10) {
					secs = '0' + secs;
				}
				if (mins < 10) {
					mins = '0' + mins;
				}

				element.innerHTML = mins + ':' + secs;
			} else {
				//all timers start @ X mins
				const combined = element.innerHTML.split(':');
				var secs = Number(combined[1]);
				var mins = Number(combined[0]);
				if (mins >= 0){
					//counting down logic here
					if (mins > KDS.reversedminutes && mins > 0) {
						//mins = KDS.reversedminutes;
					}
					if (secs > 0) {						
						secs -= 1;
					} else {						
						mins = mins - 1;
						if (mins >= 0){
							secs = 59;
						} else {
							secs = 0;
						}												
					}
				} else {
					//counting up logic here
					if (secs < 59) {						
						secs += 1;
					} else {
						mins = mins - 1;
						if (mins >= 0){
							secs = 59;
						} else {
							secs = 0;
						}
					}
				}
				
				// Reverse Logic
				if (mins <= KDS.late1){
					//Past Late1 Stage					
					if (mins <= KDS.late2){
						//Past Late2 Stage
						if (KDS.hasClass(mainid, 'late1colorclass') == true) {
							KDS.removeClass(mainid, 'late1colorclass');
						}
						if (KDS.hasClass(mainid, 'late2colorclass') == false) {
							KDS.addClass(mainid, 'late2colorclass');
						}
					} else {
						//Just Past Stage1
						if (KDS.hasClass(mainid, 'late1colorclass') == false) {
							KDS.addClass(mainid, 'late1colorclass');
						}
					} 
				}								
				
				if (secs < 10) {
					secs = '0' + secs;
				}				
				if (mins < 10 && mins > -1) {
					
					//mins = '0' + mins;
				}

				element.innerHTML = mins + ':' + secs;
			}
		}
	});	
};
KDS.per_second_timer = function() {
	KDS.timerPID = setInterval(function() {
		KDS.set_system_time();
		KDS.upateitemstimers(KDS.itemtimers);
		KDS.updateProductionTimers(KDS.productionTimers);
	}, 1000);
};
KDS.cancel_timer = function() {
	KDS.timerPID = null;
};
KDS.time_difference_now = function(d1) {	
	var msec = Date.now() - d1;
	var mins = Math.floor(msec / 60000);
	var hours = mins / 60;
	var rhours = Math.floor(hours);
	var minutes = (hours - rhours) * 60 + (rhours * 60);
	var rminutes = Math.round(minutes);
	if (!KDS.reversed){
		rminutes = rminutes;
	} else {
		if (rminutes > KDS.reversedminutes) {
			rminutes = 1;
		} else {
			rminutes = (KDS.reversedminutes - rminutes);
		}		
	}
	if (rminutes < 10) {
		rminutes = '0' + rminutes;
	}
	return rminutes + ':' + '00';
};
// Same idea as KDS.time_difference_now but keeps real seconds instead of
// always seeding ":00" - needed anywhere a timer can be re-seeded from
// scratch repeatedly (e.g. the production timer, re-rendered whenever any
// item in the same order changes), where snapping seconds back on every
// re-render would look like the timer itself got reset.
KDS.time_difference_now_full = function(d1) {
	var msec = Date.now() - d1;
	if (msec < 0) {
		msec = 0;
	}
	var totalSecs = Math.floor(msec / 1000);
	var mins = Math.floor(totalSecs / 60);
	var secs = totalSecs % 60;
	if (mins < 10) {
		mins = '0' + mins;
	}
	if (secs < 10) {
		secs = '0' + secs;
	}
	return mins + ':' + secs;
};
KDS.message_preptime = function() {
	var instruction = {};
	instruction.type = 'PREPTIME';
	instruction.screen = kdsscreen;
	instruction.order_id = '0';
	instruction.prepmins = 0;
	instruction.prepsecs = 0;
	KDS.ws.send(JSON.stringify(instruction));
};
// ============================= END TIMERS ================= //

// ============================== ORDER RECALLING  ================== //


KDS.messages_getcompleted = function() {
	var instruction = {};
	instruction.type = 'GETCOMPLETEDFROMSCREEN';
	instruction.screen = kdsscreen;
	instruction.order_id = '0';
	KDS.log(instruction);
	KDS.ws.send(JSON.stringify(instruction));
};


// ============================ NATS FUNCTIONS ======================= //

KDS.nats.init = async function() {
	KDS.nc = await NATS.connect({servers:"0.0.0.0:4222"});
	KDS.nats.subscription();
};
KDS.nats.close_order = async function(id) {
	try {
		// Screen/passmode tell the backend whose Close Order this is, so it
		// can scope the completion to this terminal's own screen (or, in
		// pass mode, force-close the whole order regardless of screen)
		// instead of closing every item indiscriminately. Standalone Mode
		// is NOT the same thing as pass mode here, despite an earlier
		// attempt in this codebase's history to treat it that way - a
		// standalone terminal only means "this screen closes its own items
		// without needing a pass-terminal middleman assigned to it"; it
		// says nothing about being the only screen in the building. A site
		// with, say, food going to a Kitchen terminal and drinks going to
		// a separately Standalone-Mode Bar terminal has TWO standalone
		// terminals, each rightly scoped to its own screen - force-closing
		// the whole order from the Bar would incorrectly bump the
		// Kitchen's still-unprepared food right along with it. Only actual
		// pass mode (a terminal explicitly responsible for bumping across
		// multiple screens on a site's behalf) gets the unconditional
		// force-close. See natsCloseOrder in kds-webserver-main.
		// KDS.kdsconfig itself is kept current by IPC_SaveConfig (index.js)
		// sending 'config-updates-remotely' right after every local save,
		// same as it already does for a config update arriving from a
		// remote terminal - the board's own frontend._kds.ipcINIT listener
		// reloads the page on that event, so this always reflects whatever
		// was actually last saved rather than whatever this page happened
		// to load with.
		let message = new NATS_Message(""+id, null , null, "sub-order-close", KDS.kdsconfig.client.screen, KDS.flagcheck("passmode"));
		let sc = NATS.StringCodec();
		console.log(message.getJSON());
		KDS.nc.publish("sub-order-close", sc.encode(JSON.stringify(message.getJSON())) );
	} catch (err) {
		console.log(err);
	}
	setTimeout(() => {
		KDS.nats.update_order();
	}, 2000);
};
KDS.nats.close_item = async function(){
	try {		
		let message = new NATS_Message(""+id, null , null, "sub-order-close");
		let sc = NATS.StringCodec();
		//console.log(message.getJSON());
		KDS.nc.publish("sub-order-close", sc.encode(JSON.stringify(message.getJSON())) );
	} catch (err) {
		console.log(err);
	}
	setTimeout( () => {
		KDS.nats.update_order();
	} , 2000);
}
KDS.nats.update_order = async function() {
	try {		
		let message = new NATS_Message("", null , null, "getorders");
		let sc = NATS.StringCodec();
		//console.log(message);	
		KDS.nc.publish("getorders", sc.encode(JSON.stringify(message.getJSON())) );
	} catch (err) {
		console.log(err);
	}
};
KDS.nats.incomingMessage = async function(sub){
	let sc = NATS.StringCodec();	
	for await (const msg of sub) {  				
		try {			
			var js = JSON.parse(sc.decode(msg.data));					
			//console.log(js);
			if (js.messagetype === 'preptime') {
				KDS.log("PREPTIME SSE");
				//console.log("PREPTIME");
				KDS.update_preptime(js);
			} else if (js.messagetype === 'update') {
				KDS.summary_open = {};
				KDS.summary_sub = {};
				KDS.summary_by_status = { notProduced: {}, inProduction: {}, completed: {} };
				KDS.summary_counts = { notProduced: 0, inProduction: 0, completed: 0 };
				KDS.log("UPDATE");
				KDS.messages_incoming(js.data);
			} else if (js.messagetype === 'reload') {
				location.reload();
			}
		}
		catch(err) {
			KDS.log(err);
			console.log("INCOMING ERROR");
			console.log(err);
			//console.log(sc.decode(msg.data));
		}
		finally {
			//console.log("INCOMING");
			KDS.log("Message Received1");
			KDS.render_summary();
		} 		             
	}
};
KDS.nats.subscription = function() {
	const updates_channel = KDS.nc.subscribe('updates');
	KDS.nats.incomingMessage(updates_channel);	
	if (KDS.nc.protocol.connected){
		//KDS.log(`connect to ${KDS.nc.currentServer.url.host}`);
		KDS.addAndremoveClass('connection_status', 'button-green', 'button-red');
		KDS.ajax_startup();
	}	
};

KDS.messages_incoming = function(js) {
	console.log("INCOMING");
	var need2playsound = false;

	// js is the full current set of open orders every cycle (the
	// reconciliation below already relies on that to know what to remove),
	// so rebuilding this fresh each time keeps it exactly in sync with
	// what's on screen with no separate pruning needed.
	KDS.orders = {};

	js.forEach((element) => {
		try {
			KDS.orders[element.id] = element;
			if (KDS.process_order(element)){
				need2playsound = true;
			}
		} catch (error) {
			console.log(error);
		}
	});
	// Any order tile still on screen that isn't in this update is no longer
	// open - most commonly because a DIFFERENT terminal (e.g. the pass-mode
	// KDS) closed it. Closing locally already removes this terminal's own
	// tile immediately/optimistically (see KDS.close_order), but a remote
	// close only ever reaches this terminal through this periodic "updates"
	// push, which simply omits the now-closed order rather than saying
	// anything explicit about it - so this reconciliation is the only place
	// a remotely-closed order's tile ever gets cleaned up.
	//
	// Also removes a tile whose order is STILL open overall but no longer
	// belongs_to_screen here (see that function) - a non-pass terminal on a
	// multi-screen order can finish its own part while other screens are
	// still working theirs, so the order stays in this same "still open"
	// update indefinitely. KDS.process_order's own el1!==false branch only
	// ever UPDATES a tile that still belongs here; it has no path to remove
	// one that's stopped belonging, so without this the tile would just sit
	// there frozen forever - already fully done from this terminal's own
	// side, but never actually going away.
	try {
		let stillRelevantIds = new Set(
			js.filter((order) => KDS.HTML_order_belongs_to_screen(order)).map((order) => String(order.id))
		);
		document.querySelectorAll('#cardss [data-id]').forEach((tile) => {
			let id = tile.getAttribute('data-id');
			if (!stillRelevantIds.has(id)) {
				KDS.removeTimer(id);
				// Otherwise a closed order's id lingers in here forever and,
				// on the rare chance the same numeric order id is ever
				// reused, its very first item would wrongly look like a
				// "new item added" reset instead of a fresh order.
				delete KDS.orderItemIds[id];
				let indexEl = document.getElementById('index_' + id);
				if (indexEl) {
					indexEl.remove();
				}
				tile.remove();
			}
		});
	} catch (error) {
		console.log(error);
	}
	// Runs once per cycle, after every tile's own data-boardrank has just
	// been refreshed above (via each order's own HTML_GENERATE_ORDER_ITEMS
	// call) and any now-irrelevant tile has just been removed - see
	// KDS.HTML_resort_board's own comment for why a single full re-sort
	// here replaces the old per-tile incremental repositioning.
	try {
		KDS.HTML_resort_board();
	} catch (error) {
		console.log(error);
	}
	if (need2playsound) {
		//console.log("play");
		KDS.playSound();
		// KDS.playthesound();
	} else {
		//console.log("not play");
	}
};

// Full, stable re-sort of the whole board, run once per update cycle after
// every tile's data-timerank/data-boardrank has just been refreshed (see
// HTML_GENERATE_ORDER_ITEMS). Two modes, mutually exclusive:
//
// Show Oldest Orders First: unconditional strict arrival-time order, oldest
// (smallest data-timerank) first - every single cycle, regardless of
// production state, priority keyword, just-recalled, or anything else. This
// used to only be applied once, at a tile's initial insertion
// (HTML_INSERT_ORDER_SORTED), relative to whatever else was on the board at
// that exact moment - correct at insertion time doesn't stay correct
// forever, and nothing ever re-validated a tile's position after the fact,
// so any other code path touching board order could leave a tile stuck in
// the wrong place with nothing to self-correct it (confirmed live: a
// 51-second-old order sitting ahead of ones 10+ minutes old). Re-deriving
// the full order from data-timerank every cycle instead means it is always
// correct, however it got there.
//
// Sort Orders By Production State: in-production first, then not-started,
// then complete (or reversed on a pass terminal via the
// passmode-boardorder body class) - only when Show Oldest Orders First is
// off, matching its own "overrides everything else" precedence. Stable:
// ties (same rank) keep their existing relative DOM order, so within a
// rank group tiles still read in whatever order they were originally
// inserted - this only ever reorders between rank groups, never within
// one. Same self-correction rationale as oldest-first above: every order
// in a single update cycle goes through HTML_GENERATE_ORDER_ITEMS in
// whatever order the backend returned them in, not already sorted by rank,
// so repositioning one tile at a time mid-pass could use a sibling's stale
// rank with no guarantee of ever converging - a single full sort after
// every tile's rank is current has no such ordering dependency.
KDS.HTML_resort_board = function() {
	let cardssEl = document.getElementById('cardss');
	if (!cardssEl) {
		return;
	}
	if (KDS.flagcheck('showoldestfirst')) {
		// data-timerank is oldestOnThisScreen, falling back to order.id -
		// see HTML_GENERATE_ORDER_ITEMS's comment - so this read-back
		// never needs its own separate fallback direction to pick between;
		// whatever was already stamped there is safe to sort on as-is.
		let byTime = Array.prototype.map.call(cardssEl.children, (el, i) => ({
			el: el,
			i: i,
			rank: Number(el.getAttribute('data-timerank')) || 0
		}));
		byTime.sort((a, b) => (a.rank - b.rank) || (a.i - b.i));
		let timeFrag = document.createDocumentFragment();
		byTime.forEach((entry) => timeFrag.appendChild(entry.el));
		cardssEl.appendChild(timeFrag);
		return;
	}
	if (!KDS.flagcheck('sortbyprodstate')) {
		return;
	}
	let ranked = Array.prototype.map.call(cardssEl.children, (el, i) => ({
		el: el,
		i: i,
		rank: Number(el.getAttribute('data-boardrank') || 0)
	}));
	ranked.sort((a, b) => (a.rank - b.rank) || (a.i - b.i));
	let frag = document.createDocumentFragment();
	ranked.forEach((entry) => frag.appendChild(entry.el));
	cardssEl.appendChild(frag);
};


// ============================ END NATS ============================= //

// ============================= GENERAL FUNCTIONS ==================== //
KDS.log = function(data) {
	if (KDS.debug) {
		console.log(data);
	}
};
KDS.playSound = function() {
	var sound = new Howl({
	src: ['assets/ding.mp3']
	});
	
	sound.play();
    window.KDSAPI.playSound();   
};
KDS.helper.update_value = function(id , value) {
	var element = document.getElementById(id);
	element.innerText = value;	
};

KDS.appendHTML = function(id, content) {
	var element = document.getElementById(id);
	element.innerHTML += content;
};
KDS.prependHTML = function(id, content) {
	var element = document.getElementById(id);
	element.innerHTML = content + element.innerHTML;
};
KDS.prependTO = function(elementid , containerid) {
	var elem = document.getElementById(elementid);
	var elem2 = elem;
	elem.remove();
	var container = document.getElementById(containerid);
	container.prepend(elem2);
};
KDS.updateHTML = function(id, content) {
	var element = document.getElementById(id);
	element.innerHTML = content;
};
KDS.isJSON = function(str) {
	try {
		JSON.parse(str);
	} catch (e) {
		return false;
	}
	return true;
};
KDS.removeClass = function(id, classname) {
	var element = document.getElementById(id);
	if (null !== element) {
		element.classList.remove(classname);
	}
};
KDS.addClass = function(id, classname) {
	var element = document.getElementById(id);
	if (null !== element) {
		element.classList.add(classname);
	}
};
KDS.hasClass = function(id, classname) {
	var element = document.getElementById(id);
	if (null !== element) {
		return element.classList.contains(classname);
	} else {
		return false;
	}
};
KDS.addAndremoveClass = function(id, addclass, removeclass) {
	var element = document.getElementById(id);
	if (null !== element) {
		element.classList.remove(removeclass);
		element.classList.add(addclass);
	}
};
KDS.fixDate = function(d1) {
	d1 = d1.replace('T', ' ');
	d1 = d1.replace('Z', ' ');
	return d1;
};
// ============================= END GENERAL FUNCTIONS ==================== //


// ***************************** HTML GENERATORS ========================== //

KDS.gen_mainitems = function(element) {
	var open = [];
	var kiddies = {};
	var voids = {};
	element.items.forEach((mainitem) => {
		if (KDS.screenMatches(mainitem.screen.name, kdsscreen)) {
			if (mainitem.status === 'OPEN') {
				if (mainitem.iskiddies === 1) {
					if (mainitem.name in kiddies) {
						kiddies[mainitem.name].qty += Number(mainitem.qty);
						mainitem.subitems.forEach((subitem) => {
							kiddies[mainitem.name].sub.push(subitem);
						});
					} else {
						kiddies[mainitem.name] = {};
						kiddies[mainitem.name].qty = Number(mainitem.qty);
						kiddies[mainitem.name].sub = [];
						mainitem.subitems.forEach((subitem) => {
							kiddies[mainitem.name].sub.push(subitem);
						});
					}
				} else {
					if (mainitem.name in open) {
						open[mainitem.name].qty += Number(mainitem.qty);
						mainitem.subitems.forEach((subitem) => {
							open[mainitem.name].sub.push(subitem);
						});
					} else {
						open[mainitem.name] = {};
						open[mainitem.name].qty = Number(mainitem.qty);
						open[mainitem.name].sub = [];
						mainitem.subitems.forEach((subitem) => {
							open[mainitem.name].sub.push(subitem);
						});
					}
				}
			} else if (mainitem.status === 'VOID') {
				if (mainitem.name in voids) {
					voids[mainitem.name].qty += Number(mainitem.qty);
					voids[mainitem.name].sub += mainitem.subitems;
				} else {
					voids[mainitem.name] = {};
					voids[mainitem.name].qty = Number(mainitem.qty);
					voids[mainitem.name].sub = mainitem.subitems;
				}
			}
		}
	});

	if (undefined !== KDS.kdsconfig.flags.sortsubitems && KDS.kdsconfig.flags.sortsubitems){
		for (var key in open) {
			open[key].sub.sort( (a,b) => (a.name > b.name) ? 1 : -1);
		}
		for (var key in kiddies) {
			kiddies[key].sub.sort( (a,b) => (a.name > b.name) ? 1 : -1);
		}
	}
	return {
		kiddies: kiddies,
		voids: voids,
		open: open
	};
};

// Returns true the first time a given item id shows up for this order on
// this screen since it was last checked - i.e. a new item was added to an
// already-open order - and updates the tracked set (KDS.orderItemIds)
// either way, ready for the next comparison. Always false on an order's
// very first render (nothing to compare against yet, tracked here as
// undefined) - HTML_GENERATE_ORDER's own updatetimers=true call already
// seeds the tile's timer correctly for that case, so process_order only
// needs this for the "already existed" branch. Screen-scoped like every
// other per-screen item check here: a new drink added to an order must not
// reset the food screen's own timer for that same order, and vice versa.
KDS.HTML_order_has_new_item = function(order) {
	let currentIds = order.items
		.filter((mainitem) => KDS.screenMatches(mainitem.screen.name, kdsscreen) || KDS.flagcheck("enableallcenters"))
		.map((mainitem) => mainitem.id);
	let known = KDS.orderItemIds[order.id];
	let hasNew = !!known && currentIds.some((id) => !known.has(id));
	KDS.orderItemIds[order.id] = new Set(currentIds);
	return hasNew;
};

KDS.process_order = function(element) {
	let playsound = false;
	let isvalid = KDS.HTML_check_order_valid(element);
	// isvalid counts non-COMPLETE items, so it hits 0 once every item on the
	// order has been ticked off - it's kept as-is for numofitemsforthiscreen
	// (column-layout thresholds), but must not gate whether a card gets
	// drawn/refreshed at all. In pass mode a fully-completed order stays on
	// the board (only a manual Close Order closes it) instead of being
	// removed like it normally would be, so gating on isvalid here would
	// leave its card stale, or never draw it in the first place on a fresh
	// load. HTML_order_belongs_to_screen is the correct gate: it only checks
	// screen relevance, independent of completion status.
	let belongsHere = KDS.HTML_order_belongs_to_screen(element);
	let el1 = KDS.HTML_find_order(element.id);
	if (el1 !== false) {
		if (belongsHere) {
			try{
				if (KDS.kdsconfig.hasOwnProperty('sound')) {
					if (KDS.kdsconfig.sound.hasOwnProperty('updateOrder')) {
						if (KDS.kdsconfig.sound.updateOrder.enabled) {
							playsound = true;
						}
					}
				}
				KDS.HTML_UPDATE_MAIN_ORDER(element);
				// A new item added to this already-open order resets this
				// tile's own elapsed-time display back to the new item's
				// arrival, instead of it staying anchored to whichever item
				// happened to arrive first - see HTML_order_has_new_item.
				let newItemAdded = KDS.HTML_order_has_new_item(element);
				KDS.HTML_GENERATE_ORDER_ITEMS(element, isvalid , true, newItemAdded);
			}catch(error){
				console.log(error);
			}
		}
	} else if (belongsHere) {
		KDS.HTML_GENERATE_ORDER(element, isvalid);
		console.log("NOT NEW ORDER?");
		// Seeds KDS.orderItemIds for this order so the next update can tell
		// whether a further item was added - the return value doesn't
		// matter here since updatetimers is already forced true below.
		KDS.HTML_order_has_new_item(element);
		KDS.HTML_GENERATE_ORDER_ITEMS(element, isvalid , false , true);
		playsound = true;
	}
	return playsound;
};
// ****************************** END ===================================== //




//////////////////////////////////// HTML GENERATION //////////////////////////////////
KDS.HTML_find_order = function(orderid) {
	let ele =  document.getElementById('elem_' + orderid);	
	if (typeof ele != 'undefined' && ele != null) {
		//console.log("RET ELEME");
		return ele;
	} else {
		return false;
	}
};
KDS.HTML_UPDATE_TABLE_WAITER = function(orderid , waitername) {
	let ele =  document.getElementById('waiter_' + orderid);
	ele.innerHTML = waitername;
};
KDS.HTML_UPDATE_TABLE_TABLE = function(orderid , table) {
	let ele =  document.getElementById('table_' + orderid);
	ele.innerHTML = table;
};

KDS.HTML_check_order_valid = function(order) {
	let forthiscreen = order.items.filter((mainitem) => {
		//FLAG: enableallcenters
		if (KDS.flagcheck("enableallcenters")){
			return mainitem.status !== 'COMPLETE';
		} else {
			return KDS.screenMatches(mainitem.screen.name, kdsscreen) && mainitem.status !== 'COMPLETE';
		}

	});
	return forthiscreen.length;
};
// Whether any of this order's items belong on this screen at all, regardless
// of completion status - unlike HTML_check_order_valid (which only counts
// non-COMPLETE items, for column-layout thresholds), this must stay true for
// a fully-completed-but-still-open order so its card keeps getting drawn and
// refreshed in pass mode instead of going stale or never appearing.
KDS.HTML_order_belongs_to_screen = function(order) {
	if (KDS.flagcheck("enableallcenters")) {
		return order.items.length > 0;
	}
	if (KDS.flagcheck("passmode")) {
		// This used to assume a pass terminal could never land in the "my
		// own items are done but other screens aren't" state the non-pass
		// branch below exists for, on the theory that its Close Order is
		// always all-or-nothing. That's wrong: closeOrderForPassScreen only
		// force-closes its OWN section, and a genuinely separate terminal
		// on the same order (e.g. Bar) can easily still be OPEN while this
		// screen's own items are already done - same as any non-pass
		// terminal. Filtered to OPEN so the tile disappears once there's
		// nothing left on THIS screen to bump.
		//
		// Deliberately still matches on this screen's own name only, NOT
		// mainitem.screen.assigned_pass_terminal - every tile-content
		// renderer (HTML_GENERATE_ORDER_ITEMS and everything downstream of
		// it) only ever shows items whose screen.name literally matches
		// kdsscreen too, with no assigned-sub-terminal awareness at all.
		// Widening visibility here to include assigned sub-screens (tried
		// once, reverted) made an order with items ONLY on an assigned
		// sub-terminal "belong" on this board while having nothing any
		// renderer would actually draw inside it - a blank tile. A pass
		// terminal's own board only ever shows its own screen's items;
		// closeOrderForPassScreen's wider section-reach is a closing-time
		// action, not a display-time one.
		return order.items.some((mainitem) => KDS.screenMatches(mainitem.screen.name, kdsscreen) && mainitem.status === 'OPEN');
	}
	// A non-pass terminal's Close Order only ever touches its own screen's
	// items (closeOrderScreenDatabase), so on a multi-screen order it can
	// genuinely finish its own part while other screens are still working
	// theirs - the order stays OPEN overall until every screen is done.
	// Without this OPEN-status check, this terminal's tile for that order
	// never leaves the board: it already has nothing left to bump (its own
	// item shows correctly as complete/checked), but the tile itself just
	// sits there indefinitely since the ORDER hasn't fully closed yet -
	// reading as "Close Order isn't working" when there's actually nothing
	// left for this screen to do. Recall Orders (a separate view, scoped by
	// completed items on this screen) is unaffected and still finds it.
	return order.items.some((mainitem) => KDS.screenMatches(mainitem.screen.name, kdsscreen) && mainitem.status === 'OPEN');
};
KDS.HTML_update_order_time = function(orderid, thetime){
	KDS.updateHTML('timer-' + orderid, thetime);
};
// Two main items can share the exact same display name (e.g. a split
// transfer landing two "Ribeye Steak" rows on one tile) while sitting in
// different states - one not yet started, one in production, one complete.
// HTML_compress_mainitems/HTML_GENERATE_MAIN_ITEMS used to key their
// compression buckets and the `done` dedup set by name alone, so whichever
// of these same-named items rendered first "claimed" the name and every
// other state's copy was silently skipped - not shown anywhere on the tile,
// yet still counted as OPEN by KDS.close_order's incomplete check. Keying
// by name *and* production/complete bucket keeps same-name-same-state items
// compressing into one row (unchanged behavior) while same-name-different-
// state items each get their own slot.
KDS.HTML_compress_key = function(mainitem) {
	let bucket = 'other';
	if (mainitem.status === 'OPEN') {
		bucket = mainitem.in_production === true ? 'inprod' : 'notprod';
	} else if (mainitem.status === 'COMPLETE') {
		bucket = 'complete';
	}
	return mainitem.name + '_' + bucket;
};

KDS.HTML_compress_mainitems = function(order) {
	let open = [];
	let kiddies = {};
	let voids = {};
	let byStatus = { notProduced: {}, inProduction: {}, completed: {} };

	order.items.forEach((mainitem) => {
		//FLAG: enableallcenters
		if (KDS.screenMatches(mainitem.screen.name, kdsscreen) || KDS.flagcheck("enableallcenters")) {
			// Kitchen-wide not-produced/in-production/completed counts and
			// item breakdown for the summary panel - this runs once per
			// order on every update cycle (see KDS.messages_incoming), same
			// lifecycle as summary_by_status below, so it stays a live
			// running total across every order currently on this screen.
			let statusKey = null;
			if (mainitem.status === 'OPEN') {
				if (mainitem.in_production === true) {
					KDS.summary_counts.inProduction += 1;
					statusKey = 'inProduction';
				} else {
					KDS.summary_counts.notProduced += 1;
					statusKey = 'notProduced';
				}
			} else if (mainitem.status === 'COMPLETE') {
				KDS.summary_counts.completed += 1;
				statusKey = 'completed';
			}
			// Kiddies items are deliberately excluded here too, matching
			// the scope they've always had in this summary (never surfaced
			// in the panel, only kept for the per-order kiddies grouping
			// below).
			if (statusKey !== null && mainitem.iskiddies !== 1) {
				let bucket = byStatus[statusKey];
				if (mainitem.name in bucket) {
					bucket[mainitem.name].qty += Number(mainitem.qty);
				} else {
					bucket[mainitem.name] = { qty: Number(mainitem.qty), sub: {} };
				}
				mainitem.subitems.forEach((subitem) => {
					let subBucket = bucket[mainitem.name].sub;
					if (subitem.name in subBucket) {
						subBucket[subitem.name].qty += Number(subitem.qty);
					} else {
						subBucket[subitem.name] = { qty: Number(subitem.qty) };
					}
				});
			}
			if (mainitem.status === 'OPEN') {
				let compressKey = KDS.HTML_compress_key(mainitem);
				if (mainitem.iskiddies === 1) {
					if (compressKey in kiddies) {
						kiddies[compressKey].qty += Number(mainitem.qty);
						mainitem.subitems.forEach((subitem) => {
							kiddies[compressKey].sub.push(subitem);
						});
					} else {
						kiddies[compressKey] = {};
						kiddies[compressKey].qty = Number(mainitem.qty);
						kiddies[compressKey].sub = [];
						mainitem.subitems.forEach((subitem) => {
							kiddies[compressKey].sub.push(subitem);
						});
					}
				} else {
					if (compressKey in open) {
						open[compressKey].qty += Number(mainitem.qty);
						mainitem.subitems.forEach((subitem) => {
							open[compressKey].sub.push(subitem);
						});
					} else {
						open[compressKey] = {};
						open[compressKey].qty = Number(mainitem.qty);
						open[compressKey].sub = [];
						mainitem.subitems.forEach((subitem) => {
							open[compressKey].sub.push(subitem);
						});
					}
				}
			} else if (mainitem.status === 'VOID') {
				if (mainitem.name in voids) {
					voids[mainitem.name].qty += Number(mainitem.qty);
					voids[mainitem.name].sub += mainitem.subitems;
				} else {
					voids[mainitem.name] = {};
					voids[mainitem.name].qty = Number(mainitem.qty);
					voids[mainitem.name].sub = mainitem.subitems;
				}
			}
		}
	});
	if (undefined !== KDS.kdsconfig.flags.sortsubitems && KDS.kdsconfig.flags.sortsubitems){
		for (var key in open) {
			open[key].sub.sort( (a,b) => (a.name > b.name) ? 1 : -1);
		}
		for (var key in kiddies) {
			kiddies[key].sub.sort( (a,b) => (a.name > b.name) ? 1 : -1);
		}
	}
	// Merge this order's per-status breakdown into the kitchen-wide totals
	// the summary panel reads from.
	['notProduced', 'inProduction', 'completed'].forEach((statusKey) => {
		let globalBucket = KDS.summary_by_status[statusKey];
		for (const [name, value] of Object.entries(byStatus[statusKey])) {
			if (name in globalBucket) {
				globalBucket[name].qty += value.qty;
			} else {
				globalBucket[name] = { qty: value.qty, sub: {} };
			}
			for (const [subName, subValue] of Object.entries(value.sub)) {
				let globalSub = globalBucket[name].sub;
				if (subName in globalSub) {
					globalSub[subName].qty += subValue.qty;
				} else {
					globalSub[subName] = { qty: subValue.qty };
				}
			}
		}
	});
	// Original flat (not split by production status) breakdown - kept
	// alongside summary_by_status above so KDS.render_summary can switch
	// between the two instantly via the "summarybystatus" config flag,
	// without needing to recompute either on toggle.
	for (const [key, value] of Object.entries(open)) {
		if (key in KDS.summary_open) {
			KDS.summary_open[key].qty += value.qty;
		} else {
			KDS.summary_open[key] = { qty: value.qty };
		}
		value.sub.forEach(sub => {
			if (sub.name in KDS.summary_sub) {
				KDS.summary_sub[sub.name].qty += sub.qty;
			} else {
				KDS.summary_sub[sub.name] = { qty: sub.qty };
			}
		});
	}
	return {
		kiddies: kiddies,
		voids: voids,
		open: open
	};
};

// Inserts a brand-new order tile's markup into #cardss at its correctly
// sorted position by arrival time, instead of always appending to the end
// (oldest-first, since orders were simply appended in discovery order) or
// always prepending (which would just reverse that, not actually sort
// anything once orders arrive out of a single predictable order - e.g.
// after a page reload, when every currently-open order arrives at once).
// The oldest parseable created value among items belonging to this screen
// (or every item if enableallcenters) - see HTML_GENERATE_ORDER_ITEMS's
// matching "oldestOnThisScreen" for the full reasoning (same computation,
// duplicated here since this function runs before that one, for a tile's
// very first insertion). Returns 0 if this screen has no parseable item
// timestamp at all - callers must fall back to order.id, never to 0/epoch
// or Date.now() directly (see HTML_INSERT_ORDER_SORTED's own comment).
KDS.HTML_oldestItemTimeOnScreen = function(order) {
	let oldest = 0;
	order.items.forEach((mainitem) => {
		if (!mainitem.screen) {
			return;
		}
		if (KDS.screenMatches(mainitem.screen.name, kdsscreen) || KDS.flagcheck("enableallcenters")) {
			let created = Date.parse(KDS.fixDate(mainitem.created || ''));
			if (created > 0 && (oldest === 0 || created < oldest)) {
				oldest = created;
			}
		}
	});
	return oldest;
};

// Tags the new node with data-timerank so later insertions - and this one -
// can find the correct position by comparing against every sibling already
// on the board, the same insertBefore technique HTML_GENERATE_ORDER_ITEMS
// already uses for Sort Orders By Production State's data-boardrank.
KDS.HTML_INSERT_ORDER_SORTED = function(order, html) {
	let cardssEl = document.getElementById('cardss');
	if (!cardssEl) {
		return;
	}
	let wrapper = document.createElement('div');
	wrapper.innerHTML = html;
	let node = wrapper.firstElementChild;
	if (!node) {
		return;
	}
	// Sorts by HTML_oldestItemTimeOnScreen, matching the tile's own
	// displayed timer (see HTML_GENERATE_ORDER_ITEMS) - falls back to
	// order.id, not a parsed timestamp of any kind, if this screen has no
	// parseable item timestamp at all. An id is a plain integer, assigned
	// by this terminal's own database in strict arrival order - always
	// present, never malformed, nothing to parse. A date string can be
	// missing, empty, or in a format that fails to parse, and defaulting a
	// failure to EITHER end is itself wrong: falling back to epoch
	// (oldest) shoves a broken order straight to the front as if it were
	// the most urgent thing on the board; falling back to "right now"
	// (newest) buries it at the back where staff may never notice it
	// needs attention at all - both silently mis-sort a possibly-real
	// order instead of surfacing the problem. Using id sidesteps the
	// failure mode entirely rather than picking a "least-bad" fallback.
	let timeKey = KDS.HTML_oldestItemTimeOnScreen(order) || Number(order.id) || 0;
	node.setAttribute('data-timerank', String(timeKey));
	let oldestFirst = KDS.flagcheck('showoldestfirst');
	let target = null;
	for (let sib = cardssEl.firstElementChild; sib; sib = sib.nextElementSibling) {
		let sibTime = Number(sib.getAttribute('data-timerank')) || 0;
		// Newest-first (default): stop at the first sibling older than us.
		// Oldest-first: stop at the first sibling younger than us.
		if (oldestFirst ? (sibTime > timeKey) : (sibTime < timeKey)) {
			target = sib;
			break;
		}
	}
	cardssEl.insertBefore(node, target);
};

KDS.HTML_GENERATE_ORDER = function(order) 
{	
	let lDivId = "index_" + order.id;	
	let str = '<div class="yesyesyes" id="index_' + order.id + '">';
	// ondblclick unconditionally here (not gated with an if on passmode
	// like the dialog's own CLOSE ORDER button) is safe as-is: close_order
	// already refuses to do anything on a non-pass-mode terminal, the same
	// way the bumpbar KEY_CLOSE_ORDER hotkey relies on that same internal
	// guard rather than duplicating the check at every call site.
	// fixedtilesize (KDS Config -> Fixed Tile Size) keeps every tile the
	// same height on the board regardless of item count, scrolling
	// internally instead of shrinking/growing - see the matching
	// #cardss .cardwrapper.fixed-tile-size rule in wimpy.css.
	str += '<div class="card m-1 nonononono cardwrapper' + (KDS.flagcheck("fixedtilesize") ? ' fixed-tile-size' : '') + '" id="elem_' + order.id + '" onclick="KDS.handleTileClick(' + order.id + ')" ondblclick="KDS.close_order(' + order.id + ')" data-id="' + order.id + '">';
	str += '<div style="padding:20px">';
	str += '<div class="row">';
	str += '<div class="column doleft headitem">Chk ' + order.order_forday + '</div>';
	str += '<div id="timer-' + order.id + '" class="column  doright headitem">' + '00:00' + '</div>';
	str += '</div>';
	str += '<div class="row">';
	str += '<div id="waiter_' + order.id + '" class="column column-40 doleft headitemlarger">' + order.waiter + '</div>';
	str += '<div id="table_' + order.id + '" class="column column-60 doright headitemlarger">' + order.table + '</div>';
	str += '</div>';
	str += '<div style="clear:both;"></div>';
	// Outside .tile-collapsible-body deliberately - this is the one piece
	// of item-level status that should still be visible while a tile is
	// collapsed, since that's exactly when the item list itself (which
	// these counts are drawn from) is hidden. Populated/kept current by
	// HTML_GENERATE_ORDER_ITEMS below, same as the item list itself.
	str += '<div id="tilecounts-' + order.id + '" class="tile-status-counts"></div>';
	// Full-width bar (reuses the .sectionheading visual language) instead of
	// a small corner icon - big enough to tap reliably on a touch screen and
	// the text itself communicates what tapping it does, rather than relying
	// on a chevron glyph. stopPropagation so it never also counts as the
	// tile-wide click/double-click that opens the dialog or (in pass mode)
	// closes the order. Text is kept in sync with the collapsed state by
	// KDS.toggleTileCollapse. Gated behind the "Enable Order Tile
	// Collapse/Expand" config flag (default true - this bar was always on
	// before that flag existed, so an existing install keeps seeing it
	// until someone explicitly turns it off); when disabled, the bar simply
	// isn't drawn at all, so .tile-collapsible-body below is never toggled
	// and the tile always shows its full item list.
	//
	// Fixed Tile Size replaces this bar entirely with a running "N items on
	// ticket" count instead (populated/kept current by
	// HTML_GENERATE_ORDER_ITEMS, same as tilecounts- above) - collapsing
	// doesn't free up any board space once every tile is already a fixed
	// height, so the toggle stops being useful there and the item count is
	// more worth the space.
	if (KDS.flagcheck("fixedtilesize")) {
		str += '<div id="itemcounter-' + order.id + '" class="tile-collapse-toggle tile-item-counter"></div>';
	} else if (KDS.kdsconfig.flags.enabletilecollapse !== false) {
		str += '<div id="tiletoggle-' + order.id + '" class="tile-collapse-toggle" onclick="KDS.toggleTileCollapse(event, ' + order.id + ')">Collapse Order</div>';
	}
	str += '<div class="tile-collapsible-body">';

	if (KDS.kdsconfig.flags.showheadings) {
		if (order.profile === 'restaurant') {
			str += '<center><h3 class="card-title profileitem" onclick="" style="">' + order.profile + '</h3></center>';
		} else if (order.profile === 'mr delivery') {
			str += '<center><h3 class="card-title profileitem" onclick="" style="color:#6dcff6;">' + order.profile + '</h3></center>';
		} else if (order.profile === 'uber eats') {
			str += '<center><h3 class="card-title profileitem" onclick="" style="color:purple;">' + order.profile + '</h3></center>';
		} else if (order.profile === 'drive thru') {
			str += '<center><h3 class="card-title profileitem drivethrucolor" onclick="">' + order.profile + '</h3></center>';
		} else if (order.profile === 'take away') {
			str += '<center><h3 class="card-title profileitem takeawaycolor" onclick="">' + order.profile + '</h3></center>';
		} else {
			str += '<center><h3 class="card-title profileitem" onclick="" style="color:#0b4d00;">' + order.profile + '</h3></center>';
		}
	}

	if (order.transfer_note) {
		// A line item transfer spins up its own tile at the destination
		// table (see kds-poller's engine.go), tagged with TransferNote
		// rather than overwriting Profile - so the normal heading above
		// (e.g. "sit down") stays intact, and this banner calls out the
		// transfer as its own distinct line right above the item(s).
		str += '<div class="transfernotice">' + order.transfer_note + '</div>';
	}

	str += '<div id="items_' + order.id + '" class="mainitems_wrapper">';
	str += '</div>';

	str += '</div>'; // closes tile-collapsible-body
	str += '</div>';
	str += '</div>';
	str += '</div>';

	let shouldbetop = KDS.HTML_CHECK_PRIORITY(order);
	// order.id === KDS.lastrecalled is a one-shot "just recalled, jump to
	// front" flag - consume it here regardless of which branch below
	// actually places the tile, so it can never linger and wrongly match a
	// later order that happens to reuse the same id.
	let wasJustRecalled = (order.id === KDS.lastrecalled);
	if (wasJustRecalled) {
		KDS.lastrecalled = 0;
	}

	if (KDS.flagcheck('showoldestfirst')) {
		// Show Oldest Orders First is a strict arrival-time ordering
		// promise - it overrides every other positioning rule (priority
		// keyword jump-to-front, just-recalled jump-to-front, and Sort
		// Orders By Production State's own reshuffle in
		// HTML_GENERATE_ORDER_ITEMS) so a tile's position always reflects
		// only when the order arrived, nothing else.
		KDS.HTML_INSERT_ORDER_SORTED(order, str);
	}
	else if (shouldbetop) {
		KDS.prependHTML('cardss', str);
	}
	else if (wasJustRecalled) {
		KDS.prependHTML('cardss', str);
	} else {
		// Board Behavior config: newest order first by default, with a
		// "Show Oldest Orders First" flag to flip it - see
		// HTML_INSERT_ORDER_SORTED. Governs where a tile is FIRST inserted;
		// Sort Orders By Production State (data-boardrank, in
		// HTML_GENERATE_ORDER_ITEMS) can reposition it further on later
		// updates, EXCEPT when Show Oldest Orders First is on - that flag
		// takes priority and suppresses the production-state reshuffle
		// entirely, so oldest-first stays a strict arrival-time order that
		// no later status change can undo.
		KDS.HTML_INSERT_ORDER_SORTED(order, str);
	}

	if (KDS.flagcheck("enablemulticolumn"))
	{
		let forthiscreen = order.items.filter((mainitem) => {
			//FLAG : enableallcenters
			return (KDS.screenMatches(mainitem.screen.name, kdsscreen) || KDS.flagcheck("enableallcenters"));
		});
		var numofitemsforthiscreen = forthiscreen.length;
		
		if (numofitemsforthiscreen >= 20) {
			KDS.addClass(lDivId, 'multicolumnitem3');
		} else if (numofitemsforthiscreen >= 10) {
			KDS.addClass(lDivId, 'multicolumnitem');
		}
	}
	

	KDS.itemtimers.push(order.id);
};

KDS.HTML_CHECK_PRIORITY = (order) => {
	//KDS.features.prioritykeywordenabled = false;
	//KDS.features.prioritykeywordtext = '';
	let shouldbefirst = false;
	if (KDS.features.prioritykeywordenabled) {
		if (KDS.features.prioritykeywordtext.length > 0) {
			let key = KDS.features.prioritykeywordtext.toLowerCase();
			//console.log(key);
			order.items.forEach((mainitem) => {
				//console.log(mainitem);
				if (!shouldbefirst && mainitem.name.includes(key)) {
					shouldbefirst = true;
					//console.log("KEY");
				}
				mainitem.subitems.forEach((subitem) => {
					//console.log(subitem);
					if (!shouldbefirst && subitem.name.includes(key)) {
						shouldbefirst = true;
						//console.log("KEY");
					}					
				});
			});
		}		
	}	
	return shouldbefirst;
};

KDS.HTML_CHECK_ITEM_PRIORITY = (order) => {
	var lRearrange = [];
	var lNormal = [];
	var lItem = [];
	for (let i = 0; i < order.items.length; i++) 
	{
		var lItemName = order.items[i].name.toString().toLowerCase().trim();
		if (KDS.features.priorityitemcontainertext1.trim() !== '' && lItemName == KDS.features.priorityitemcontainertext1.trim())
		{
			lRearrange.push(i);			
		}
		else if (KDS.features.priorityitemcontainertext2.trim() !== '' && lItemName == KDS.features.priorityitemcontainertext2.trim())
		{
			lRearrange.push(i);
		}
		else if (KDS.features.priorityitemcontainertext3.trim() !== '' && lItemName == KDS.features.priorityitemcontainertext3.trim())
		{
			lRearrange.push(i);
		}
		else
		{
			lNormal.push(i);
		}		
	}
	lRearrange.forEach((index) => 
	{		
		let lTempItem = order.items[index];
		lItem.push(lTempItem);
	});

	lNormal.forEach((index) => 
	{		
		let lTempItem = order.items[index];
		lItem.push(lTempItem);		
	});

	order.items = lItem;

	return order;
};

KDS.HTML_UPDATE_MAIN_ORDER = function(order) {
	KDS.HTML_UPDATE_TABLE_WAITER(order.id , order.waiter);
	KDS.HTML_UPDATE_TABLE_TABLE(order.id , order.table);
};

KDS.HTML_GENERATE_ORDER_ITEMS = function(order, numofitemsforthiscreen , checkprio = false , updatetimers = false) {
	// An item transferred away to its own tile (see kds-poller's engine.go)
	// is marked TRANSFERRED rather than VOID - unlike a real void, it was
	// never cancelled, it just isn't prepared here anymore, so it shouldn't
	// show up on this tile at all (not even with a badge).
	order.items = order.items.filter((mainitem) => mainitem.status !== 'TRANSFERRED');

	let done = {};
	let str = '';
	let itemsadded = 0;

	for (let i = 0; i < order.items.length; i++) {
		var abc = order.items[i].name.toString().toLowerCase();
		var remove_after = abc.indexOf('[');
		if (remove_after > 0){
			abc =  abc.substring(0, remove_after - 1);
		}
		
		order.items[i].name = abc;
		for (let j = 0; j < order.items[i].subitems.length; j++) {
			var abc = order.items[i].subitems[j].name.toString().toLowerCase();
			var remove_after= abc.indexOf('[');
			if (remove_after > 0){
				abc =  abc.substring(0, remove_after - 1);
			}							
			order.items[i].subitems[j].name = abc;
		}
	}	

	order = KDS.HTML_CHECK_ITEM_PRIORITY(order);

	let subs = KDS.HTML_compress_mainitems(order);
	
	// oldestOnThisScreen anchors both the tile's displayed timer (see
	// HTML_update_order_time below) and its sort key to whenever an item
	// belonging to THIS screen first showed up here - not order.created
	// (the order's global first-ever creation, which could be on a
	// completely different screen this one never saw), and not "youngest"
	// (the most recent item, which used to make the timer wrongly reset
	// every time another item landed on an already-open order).
	//
	// This is what "reset only if the order wasn't already on this
	// screen" actually means: if this screen already had an item pending
	// when a new one arrived, the OLDEST item here predates the new
	// arrival, so the timer/sort correctly doesn't move. If this screen
	// never had this order before (e.g. only a different screen's items
	// existed so far), the oldest item here IS the new arrival, so it
	// correctly reads as freshly arrived FOR THIS SCREEN - confirmed live:
	// an order whose drinks sat on the bar/milkshake screen for an hour,
	// then had food added that landed on this (food) screen for the first
	// time - this screen should show that food as new, while the
	// drinks screen (which had it the whole time) keeps counting from
	// when it actually first arrived there.
	let oldestOnThisScreen = 0;
	let youngest = 0;
	let notProducedCount = 0;
	let inProductionCount = 0;
	let completeCount = 0;
	let sentToTableCount = 0;
	let totalItemsOnTicket = 0;
	order.items.forEach((mainitem) => {
		//FLAG : enableallcenters
		if (KDS.screenMatches(mainitem.screen.name, kdsscreen) || KDS.flagcheck("enableallcenters")) {
			totalItemsOnTicket += 1;
			mainitem.created = KDS.fixDate(mainitem.created);
			let itemCreated = Date.parse(mainitem.created);
			if (itemCreated > youngest) {
				youngest = itemCreated;
			}
			if (itemCreated > 0 && (oldestOnThisScreen === 0 || itemCreated < oldestOnThisScreen)) {
				oldestOnThisScreen = itemCreated;
			}
			if (mainitem.status === 'OPEN') {
				if (mainitem.in_production === true) {
					inProductionCount += 1;
				} else {
					notProducedCount += 1;
				}
			} else if (mainitem.status === 'COMPLETE') {
				completeCount += 1;
				if (mainitem.sent_to_table === true) {
					sentToTableCount += 1;
				}
			}
		}
	});
	// Only once every item on this screen is COMPLETE (nothing left not-
	// produced or in-production) does "Send to Table" make sense to offer -
	// computed here (rather than left to its existing home further down)
	// so the complete-items render pass below can use it to decide whether
	// to draw the button on each row.
	let isOrderComplete = notProducedCount === 0 && inProductionCount === 0 && completeCount > 0;

	// Kept visible even when the tile itself is collapsed (see
	// HTML_GENERATE_ORDER) - same colors as the board's own sort-order
	// language (Board Behavior config, sectionheading-* classes) so a
	// glance at a collapsed tile still shows where it sits in that order.
	KDS.updateHTML('tilecounts-' + order.id,
		'<span class="tile-status-pill tile-status-pill-notproduced">' + notProducedCount + ' NOT PROD</span>' +
		'<span class="tile-status-pill tile-status-pill-inprod">' + inProductionCount + ' IN PROD</span>' +
		'<span class="tile-status-pill tile-status-pill-complete">' + completeCount + ' COMPLETE</span>'
	);

	// Fixed Tile Size's replacement for the collapse/expand bar (see
	// HTML_GENERATE_ORDER, which only renders the itemcounter- element when
	// this same flag is on) - guarded the same way so this never tries to
	// update an element that isn't on the page.
	if (KDS.flagcheck("fixedtilesize")) {
		KDS.updateHTML('itemcounter-' + order.id, totalItemsOnTicket + (totalItemsOnTicket === 1 ? ' item on ticket' : ' items on ticket'));
	}

	// Render in passes rather than in one pass in original order, grouped by
	// status/production state, each under its own clearly separated heading
	// (VOID/COURSE entries are special item types outside the production
	// workflow and stay ungrouped, unheaded, at the top) - so an item
	// visibly moves into the right section the moment its state changes,
	// instead of just picking up an inline badge wherever it already sat.
	let renderGroup = (filterFn, showSendToTable = false) => {
		let groupHtml = '';
		order.items.forEach((mainitem) => {
			if ((KDS.screenMatches(mainitem.screen.name, kdsscreen) || KDS.flagcheck("enableallcenters")) && !(KDS.HTML_compress_key(mainitem) in done) && filterFn(mainitem)) {
				if (!KDS.kdsconfig.flags.compressed){
					groupHtml += KDS.HTML_GENERATE_MAIN_ITEMS(mainitem, itemsadded, numofitemsforthiscreen, null , null, showSendToTable);
				} else {
					groupHtml += KDS.HTML_GENERATE_MAIN_ITEMS(mainitem, itemsadded, numofitemsforthiscreen, subs, done, showSendToTable);
				}
			}
		});
		return groupHtml;
	};

	// Not-yet-started items stay at the top with no heading of their own,
	// same as VOID/COURSE - only IN PRODUCTION and COMPLETED get pulled into
	// their own clearly bordered sections below.
	if (KDS.kdsconfig.flags.productionmode === false) {
		// Production Mode off (KDS config) - there's no "in production" state
		// for anything to be pulled into, so skip the split entirely instead
		// of rendering a section box with no visible badges inside it (which
		// could otherwise still happen off a stale in_production value from
		// before the toggle was switched off).
		str += renderGroup((mainitem) => mainitem.status !== 'COMPLETE');
	} else {
		str += renderGroup((mainitem) => !(mainitem.status === 'OPEN' && mainitem.in_production === true) && mainitem.status !== 'COMPLETE');

		let inProductionHtml = renderGroup((mainitem) => mainitem.status === 'OPEN' && mainitem.in_production === true);
		if (inProductionHtml.length > 0) {
			str += '<hr class="sectiondivider" />';
			str += '<div class="sectiongroup sectiongroup-inprod"><div class="sectionheading sectionheading-inprod">' + inProductionCount + ' IN PRODUCTION</div>' + inProductionHtml + '</div>';
		}
	}

	// "Send to Table" only ever appears once the whole order is complete
	// (isOrderComplete, computed above) - a completed item still sitting
	// alongside not-produced/in-production siblings on the same ticket
	// hasn't actually left the kitchen's hands yet. Once sent, an item
	// moves out of this section entirely and into its own "SENT TO TABLE"
	// section at the bottom of the tile (below), rather than just picking
	// up a badge in place - same "moves to where it belongs" language as
	// the IN PRODUCTION/COMPLETED split above.
	let completeHtml = renderGroup((mainitem) => mainitem.status === 'COMPLETE' && mainitem.sent_to_table !== true, true);
	if (completeHtml.length > 0) {
		str += '<hr class="sectiondivider" />';
		str += '<div class="sectiongroup sectiongroup-complete"><div class="sectionheading sectionheading-complete">' + (completeCount - sentToTableCount) + ' COMPLETED</div>' + completeHtml + '</div>';
	}

	let sentToTableHtml = renderGroup((mainitem) => mainitem.status === 'COMPLETE' && mainitem.sent_to_table === true, false);
	if (sentToTableHtml.length > 0) {
		str += '<hr class="sectiondivider" />';
		str += '<div class="sectiongroup sectiongroup-senttotable"><div class="sectionheading sectionheading-senttotable">' + sentToTableCount + ' SENT TO TABLE</div>' + sentToTableHtml + '</div>';
	}

	let cardEl = document.getElementById('elem_' + order.id);
	let indexEl = document.getElementById('index_' + order.id);
	// Tiles all stay in the single #cardss board (its normal float-based
	// layout is what actually wraps/sizes each tile correctly - splitting
	// this into separate containers broke that). Progress is instead
	// expressed as a sort position within #cardss: in-production first,
	// then not-produced, then completed - mirrored under pass mode, where
	// completed orders are the more urgent, repeated action there instead.
	// Completed only once nothing is left not-produced or in-production;
	// in-production otherwise takes priority over not-produced (matching
	// the single-color border below) so an order with some items started
	// and some not still reads as "being worked on" rather than "not
	// started". (isOrderComplete itself is computed earlier, above the
	// item render passes, so the COMPLETED section can use it too.)
	let isOrderInProduction = !isOrderComplete && inProductionCount > 0;
	if (cardEl) {
		cardEl.classList.remove('order-inproduction', 'order-complete');
		if (isOrderComplete) {
			cardEl.classList.add('order-complete');
		} else if (isOrderInProduction) {
			cardEl.classList.add('order-inproduction');
		}
	}
	// "Show Oldest Orders First" is a strict arrival-time ordering promise -
	// Sort Orders By Production State must not be allowed to reshuffle tiles
	// out of that order underneath it. Without this guard, the two ran
	// independently (see the comment in HTML_GENERATE_ORDER above this
	// function): a tile's first insertion respected oldest-first, but every
	// later status change (an item entering/leaving production, an order
	// completing) re-ran this block regardless, silently re-sorting the
	// whole board by production state instead and undoing oldest-first the
	// moment any order's state changed - which is most of the time a board
	// is actually in use.
	if (indexEl) {
		// Kept current every cycle, regardless of insertion path or which
		// sort mode is active - KDS.HTML_resort_board's oldest-first branch
		// depends on this being fresh on every tile, not just ones that
		// happened to go through HTML_INSERT_ORDER_SORTED.
		//
		// Sorts by oldestOnThisScreen (see its own comment above) - the
		// same value the tile's displayed timer now uses - so the board's
		// left-to-right order always agrees with what's actually printed
		// on each tile. Falls back to order.id (never a parsed timestamp
		// directly, never 0/epoch or Date.now()) if this screen somehow
		// has no parseable item timestamp of its own at all: an id is a
		// plain integer assigned in strict arrival order, nothing to parse
		// and nothing that can be malformed, and using it here can only
		// ever affect the rare all-timestamps-broken case, never the
		// normal one this comment is about.
		indexEl.setAttribute('data-timerank', String(oldestOnThisScreen || Number(order.id) || 0));
	}
	if (indexEl && KDS.flagcheck('sortbyprodstate') && !KDS.flagcheck('showoldestfirst')) {
		// Only stamp this tile's own rank here - actually repositioning it
		// used to happen right here too, one tile at a time, against
		// whatever data-boardrank its CURRENT DOM siblings happened to
		// already carry. But every order in the periodic update gets run
		// through this same function in a single pass (see
		// KDS.messages_incoming's js.forEach), in whatever order the
		// backend happened to return them in - NOT already sorted by rank.
		// So a tile processed early in that pass could insert itself
		// relative to a sibling that hasn't had ITS OWN rank refreshed yet
		// this same cycle, using stale data - not guaranteed to converge on
		// a correct order, and able to drift further wrong over repeated
		// cycles instead of self-correcting. KDS.HTML_resort_board (called
		// once per cycle, after every tile's rank is up to date) replaces
		// this with a single stable full-board sort instead.
		let passReversed = document.body.classList.contains('passmode-boardorder');
		let baseRank = isOrderComplete ? 2 : (isOrderInProduction ? 0 : 1);
		let rank = passReversed ? (2 - baseRank) : baseRank;
		indexEl.setAttribute('data-boardrank', String(rank));
	}

	// Anchored to oldestOnThisScreen (see its own comment above), not
	// order.created directly and not "youngest" - the displayed elapsed
	// time must reset when an order shows up on THIS screen for the first
	// time (a different screen's items don't count), but never reset just
	// because another item lands on top of ones already pending here.
	// Falls back to order.created, then youngest, only if this screen
	// somehow has no parseable item timestamp of its own at all.
	//
	// Uses time_difference_now_full, NOT time_difference_now - confirmed
	// live: the plain version always forces seconds to ":00" and re-seeds
	// from that on every re-render (any item on the order changing at
	// all, not just this screen's own), so even though oldestOnThisScreen
	// was computing the correct, unmoving minute-level anchor throughout,
	// the seconds visibly snapped back on every re-render - looking
	// exactly like a reset that wasn't actually happening underneath. See
	// time_difference_now_full's own comment; this is precisely the
	// "timer re-seeded from scratch repeatedly" case it exists for.
	let t = KDS.time_difference_now_full(oldestOnThisScreen || Date.parse(KDS.fixDate(order.created || '')) || youngest);
	if (updatetimers){
		KDS.HTML_update_order_time(order.id , t);
	}


	if (undefined !== order.consumer && order.consumer.length > 1) {
		str += '<div class="row extrainformation">';
		str += '<div class="column docenter  headitemlarger">CONSUMER: '+ order.consumer + '</div>';
		str += '</div>';
	}
	if (undefined !== order.delivery && order.delivery.length > 1) {
		str += '<div class="row extrainformation">';
		str += '<div class="column docenter  headitemlarger">DELIVERY: ' + order.delivery + '</div>';
		str += '</div>';
	}
	str += '</div>';
	str += '<div style="clear:both;"></div></div>';
	str += '';
	KDS.updateHTML('items_' + order.id, str);
	if (checkprio){
		let shouldbetop = KDS.HTML_CHECK_PRIORITY(order);
		if (shouldbetop) {
			KDS.prependTO( 'index_' + order.id , "cardss");
		};
	}
};

KDS.HTML_GENERATE_VOID = function(mainitem, itemsadded) {
	//console.log("HTML_GENERATE_VOID");
	let thehtml = '';
	itemsadded += 1;
	thehtml += '<div class="row wrapperdiv ordervoid">';
	thehtml += '<div class="column doleft fullcolumn">' + mainitem.qty + ' ' + mainitem.name + '</div>';
	thehtml += '<span class="production-tag production-tag-void">VOID</span>';
	thehtml += '</div>';
	mainitem.subitems.forEach((subitem) => {
		itemsadded += 1;
		thehtml += '<div class="row textleft ordervoid subitemwrapper">';
		thehtml += '<div class="column">' + subitem.qty + ' X ' + subitem.name + '</div>';
		thehtml += '</div>';
	});
	return thehtml;
};

KDS.HTML_GENERATE_COURSE = function(mainitem, itemsadded) {
	//console.log("HTML_GENERATE_VOID");
	let thehtml = '';
	itemsadded += 1;
	thehtml += '<div class="row wrapperdiv ordercourse">';	
	thehtml += '<div class="column column-10s0">' + mainitem.name + '</div>';		
	thehtml += '</div>';	
	return thehtml;
};

KDS.HTML_GENERATE_MAIN_ITEMS = function(mainitem, itemsadded, numofitemsforthiscreen, subs, done, showSendToTable = false){

	let thehtml = '';
	if (mainitem.status === 'OPEN') {
		itemsadded += 1;
		thehtml += KDS.HTML_GENERATE_MAINITEM_LINE(mainitem, subs, (mainitem.iskiddies === 1), numofitemsforthiscreen);
		if (KDS.kdsconfig.flags.compressed) {
			let compressed = {};
			let compressKey = KDS.HTML_compress_key(mainitem);
			if (compressKey in subs['open']) {
				done[compressKey] = true;
				subs['open'][compressKey].sub.forEach((subitem) => {
					compressed[subitem.name] = (compressed[subitem.name] || 0) + Number(subitem.qty);
				});
			} else if (compressKey in subs['kiddies']) {
				done[compressKey] = true;
				if (subs['kiddies'][compressKey].sub.length > 0) {
					subs['kiddies'][compressKey].sub.forEach((subitem) => {
						compressed[subitem.name] = (compressed[subitem.name] || 0) + Number(subitem.qty);
					});
				}
			} else {
				mainitem.subitems.forEach((subitem) => {
					compressed[subitem.name] = (compressed[subitem.name] || 0) + Number(subitem.qty);
				});
			}
			thehtml += KDS.HTML_GENERATE_SUBITEMS(compressed, itemsadded);
		} else {
			thehtml += KDS.HTML_GENERATE_SUBITEMS(mainitem, itemsadded);
		}
		//FLAG : enableallcenters
	} else if ( (KDS.screenMatches(mainitem.screen.name, kdsscreen) || KDS.flagcheck("enableallcenters")) && mainitem.status === 'VOID') {
		thehtml += KDS.HTML_GENERATE_VOID(mainitem, itemsadded);
		//FLAG : enableallcenters
	} else if ( (KDS.screenMatches(mainitem.screen.name, kdsscreen) || KDS.flagcheck("enableallcenters")) && mainitem.status === 'COURSE') {
		thehtml += KDS.HTML_GENERATE_COURSE(mainitem, itemsadded);
	} else {
		if (KDS.kdsconfig.flags.checkboxes && mainitem.status === 'COMPLETE') {
			thehtml += KDS.HTML_GENERATE_MAINITEM_LINE(mainitem, subs, (mainitem.iskiddies === 1), numofitemsforthiscreen, true, showSendToTable);
			thehtml += KDS.HTML_GENERATE_SUBITEMS(mainitem, itemsadded, true);
		}
	}
	return thehtml;
};

// KDS Config -> Item Highlight Colors (conf.kdsfeatures.itemcolors, an array
// of {name, color}) lets an operator flag specific items - allergen items,
// whatever a site wants to call out - in a color of their own choosing
// rather than the fixed handful of classname-based colors below (makegreen,
// ordervoid). Name match is case-insensitive and exact (mainitem.name is
// already lowercased well before this point, in HTML_GENERATE_ORDER_ITEMS's
// own normalization pass) - returns an inline style string (empty if no
// match) so it can simply be appended to the wrapping div's existing
// class-based styling and win by CSS specificity, without having to touch
// that classname logic at all.
//
// A kids item (iskiddies) is deliberately exempt from this override - kids
// meals must always render in the fixed .orderkiddies blue so kitchen staff
// can spot them at a glance, regardless of whatever a site has separately
// configured a color for that same item name (e.g. flagging it for an
// allergen too). Without this, a matching config entry would silently win
// via the !important below and the item would stop looking like a kids meal
// at all.
KDS.getItemColorStyle = function(mainitem) {
	if (mainitem.iskiddies === 1) {
		return '';
	}
	let itemcolors = (KDS.kdsconfig.kdsfeatures && Array.isArray(KDS.kdsconfig.kdsfeatures.itemcolors)) ? KDS.kdsconfig.kdsfeatures.itemcolors : [];
	for (let i = 0; i < itemcolors.length; i++) {
		if (itemcolors[i].name && itemcolors[i].name.toLowerCase() === mainitem.name.toLowerCase()) {
			// !important because a plain inline style isn't actually
			// guaranteed to win here - .ordervoid (see wimpy.css) marks its
			// own background-color !important, which otherwise beats an
			// inline style with no !important of its own. Without this, a
			// suborder/refund item would keep showing its built-in color
			// instead of the one configured here.
			return ' style="background-color:' + itemcolors[i].color + ' !important;"';
		}
	}
	return '';
};

KDS.HTML_GENERATE_MAINITEM_LINE = function(mainitem, subs, iskiddies = false, numofitemsforthiscreen, openitem = false, showSendToTable = false) {
	//console.log("HTML_GENERATE_MAINITEM_LINE");
	//console.log(subs);
	let thehtml = '';
	//let thex = ' X ';
	let thex = ' ';
	let classname = 'mainitemcolor';
	if (mainitem.suborder > 0 && numofitemsforthiscreen > 1) {
		classname = 'makegreen';
	} else if (iskiddies) {
		classname = 'orderkiddies';
	} else if(mainitem.name.includes('refund')){
		classname = 'ordervoid';
	}
	let productionModeEnabled = KDS.kdsconfig.flags.productionmode !== false;
	if (productionModeEnabled && mainitem.in_production === true) {
		classname += ' iteminproduction';
	}
	thehtml += '<div class="row wrapperdiv ' + classname + '"' + KDS.getItemColorStyle(mainitem) + '>';
	if (openitem){
		if (KDS.kdsconfig.flags.leftalign) {
			thehtml += '<div class="column doneitem doleft fullcolumn">' + mainitem.qty + thex + mainitem.name + '</div>';
		} else {
			thehtml += '<div class="column doneitem docenter fullcolumn column-100">' + mainitem.qty + thex + mainitem.name + '</div>';
		}
	} else {
		if (KDS.kdsconfig.flags.leftalign) {
			if (KDS.kdsconfig.flags.compressed) {
				//console.log("tring : " + mainitem.name);
				let compressKey = KDS.HTML_compress_key(mainitem);
				if (iskiddies){
					thehtml += '<div class="column doleft fullcolumn">' + subs['kiddies'][compressKey].qty + thex + mainitem.name + '</div>';
				} else {
					thehtml += '<div class="column doleft fullcolumn">' + subs['open'][compressKey].qty + thex + mainitem.name + '</div>';
				}
			} else {
				thehtml += '<div class="column doleft fullcolumn">' + mainitem.qty + thex + mainitem.name + '</div>';
			}
		} else {
			if (KDS.kdsconfig.flags.compressed) {
				let compressKey = KDS.HTML_compress_key(mainitem);
				if (iskiddies){
					thehtml += '<div class="column docenter fullcolumn column-100">' + subs['kiddies'][compressKey].qty + thex + mainitem.name + '</div>';
				} else {
					thehtml += '<div class="column docenter fullcolumn column-100">' + subs['open'][compressKey].qty + thex + mainitem.name + '</div>';
				}
			} else {
				thehtml += '<div class="column docenter fullcolumn column-100">' + mainitem.qty + thex + mainitem.name + '</div>';
			}
		}
	}
	if (mainitem.status === 'COMPLETE') {
		thehtml += '<span class="production-tag production-tag-complete">COMPLETE</span>';
	} else if (productionModeEnabled && mainitem.in_production === true) {
		thehtml += '<span class="production-tag">IN PROD</span>';
		thehtml += KDS.HTML_production_timer(mainitem, 'prodtimer-main-');
	}
	// openitem is only ever true for a COMPLETE item's own board-tile row
	// (see HTML_GENERATE_MAIN_ITEMS) - line-item specific, so this shows on
	// any complete-and-not-yet-sent item regardless of whether its siblings
	// elsewhere on the same ticket are still open (see HTML_GENERATE_ORDER_
	// ITEMS). stopPropagation so tapping it never also counts as the
	// tile-wide click/double-click that opens the dialog or closes the order.
	if (openitem && showSendToTable) {
		thehtml += '<button class="send-to-table-btn" onclick="event.stopPropagation(); KDS.send_to_table(' + mainitem.id + ');">Send to Table</button>';
	}
	thehtml += '</div>';
	return thehtml;
};

KDS.HTML_GENERATE_SUBITEMS = function(compressed, itemsadded, openitem = false){
	let thehtml = '';
	//let thex = ' X ';
	let thex = ' ';
	if (openitem){
		compressed.subitems.forEach((subitem) => {
			itemsadded += 1;
			if (KDS.kdsconfig.flags.leftalign) {
				thehtml += '<div class="row textleft subitemwrapper">';
			} else {
				thehtml += '<div class="row docenter subitemwrapper">';
			}
			thehtml += '<div class="column doneitem subitemcolor">' + subitem.qty + thex + subitem.name + '</div>';
			thehtml += '</div>';
		});
	} else {
		if (KDS.kdsconfig.flags.compressed) {
			for (const [key, value] of Object.entries(compressed)) {
				itemsadded += 1;				
				if (KDS.kdsconfig.flags.leftalign) {
					thehtml += '<div class="row textleft subitemwrapper">';
				} else {
					thehtml += '<div class="row docenter subitemwrapper">';
				}
				//console.log(key.includes("instruction"));
				if (key.includes("instruction:")){
					//PROD INSTRUCTION
					console.log("PROD");
					let newkey = key.replace("instruction:", "#");
					thehtml += '<div class="column" style="background-color:#CCFFFF;">&nbsp;&nbsp;->&nbsp;&nbsp;' + newkey + '</div>';
				} else {
					thehtml += '<div class="column">&nbsp;&nbsp;' + value + ' ' + key + '</div>';
				}
				//thehtml += '<div class="column">' + value + ' X ' + key + '</div>';
				thehtml += '</div>';
			}
		} else {
			compressed.subitems.forEach((subitem) => {
				itemsadded += 1;
				if (KDS.kdsconfig.flags.leftalign) {
					thehtml += '<div class="row textleft subitemwrapper">';
				} else {
					thehtml += '<div class="row docenter subitemwrapper">';
				}
				if (subitem.name.includes("instruction:")){
					//PROD INSTRUCTION
					console.log("PROD");
					let newkey = subitem.name.replace("instruction:", "#");
					thehtml += '<div class="column" style="background-color:#CCFFFF;">&nbsp;&nbsp;->&nbsp;&nbsp;' + newkey + '</div>';
				} else {
					thehtml += '<div class="column subitemcolor">&nbsp;&nbsp;' + subitem.qty + thex + subitem.name + '</div>';
				}
				//thehtml += '<div class="column">' + value + ' X ' + key + '</div>';
				thehtml += '</div>';
				//thehtml += '<div class="column subitemcolor">' + subitem.qty + thex + subitem.name + '</div>';
				//thehtml += '</div>';
			});
		}
	}
	return thehtml;
};

KDS.message_single_generate = function(element) {
	// Cached so KDS.close_order can check for not-yet-completed items
	// without a separate round trip - refreshed every time this dialog
	// (re)renders, so it's always current for whichever order is open.
	KDS.activeOrderData = element;
	var belongonthisscreen = false;
	var str = '<div id="main_order_' + element.id + '" class="card m-1 nonononono yesyesyes cardwrapper">';
	str += '<div style="padding:10px">';

	str += '<div class="row">';
	str += '<div class="column doleft headitem">Chk ' + element.order_forday + '</div>';
	str += '<div class="column doright headitem">N/A</div>';
	str += '</div>';

	str += '<div class="row">';
	str += '<div class="column column-40 doleft headitemlarger">' + element.waiter + '</div>';
	str += '<div class="column column-60 doright headitemlarger">' + element.table + '</div>';
	str += '</div>';
	str += '';
	str += '<div style="clear:both;"></div>';
	str += '<center><h3 class="card-title profileitem" onclick="" style="color:#000000;">' + element.profile + '</h3></center>';
	str += '<div style="padding:0px 5px 0px 5px;">';
	let itemsadded = 0;
	let forthiscreen = element.items.filter((mainitem) => {
		//FLAG : enableallcenters
		return (KDS.screenMatches(mainitem.screen.name, kdsscreen) || KDS.flagcheck("enableallcenters"));
	});
	var numofitemsforthiscreen = forthiscreen.length;
	if (numofitemsforthiscreen >= 20) {
		KDS.addClass('makesingle', 'multicolumn3');
	} else if (numofitemsforthiscreen >= 10) {
		KDS.addClass('makesingle', 'multicolumn');
	}
	element.items.forEach((mainitem) => {
		//FLAG : enableallcenters
		if (KDS.screenMatches(mainitem.screen.name, kdsscreen) || KDS.flagcheck("enableallcenters")) {
			belongonthisscreen = true;
			if (!KDS.kdsconfig.flags.checkboxes) {
				if (mainitem.status === 'OPEN') {
					itemsadded += 1;
					if (mainitem.iskiddies === 1) {
						str += '<div class="row wrapperdiv wrapperdiv-inline doleft orderkiddies"' + KDS.getItemColorStyle(mainitem) + '>';
						str += '<div class="column fullcolumn">' + mainitem.qty + ' X ' + mainitem.name + '</div>';
						str += '<div class="itemproductioncontrol-wrap">' + KDS.HTML_production_control(mainitem) + KDS.HTML_production_timer(mainitem) + '</div>';
						str += '</div>';
					} else {
						if (mainitem.suborder > 0 && numofitemsforthiscreen > 1) {
							if (mainitem.name.includes('refund')) {
								str += '<div class="row wrapperdiv wrapperdiv-inline ordervoid"' + KDS.getItemColorStyle(mainitem) + '>';
							} else {
								str += '<div class="row wrapperdiv wrapperdiv-inline makegreen"' + KDS.getItemColorStyle(mainitem) + '>';
							}
						} else {
							if (mainitem.name.includes('refund')) {
								str += '<div class="row wrapperdiv wrapperdiv-inline ordervoid"' + KDS.getItemColorStyle(mainitem) + '>';
							} else {
								str += '<div class="row wrapperdiv wrapperdiv-inline mainitemcolor"' + KDS.getItemColorStyle(mainitem) + '>';
							}
						}
						str += '<div class="column doleft fullcolumn">' + mainitem.qty + ' X ' + mainitem.name + '</div>';
						str += '<div class="itemproductioncontrol-wrap">' + KDS.HTML_production_control(mainitem) + KDS.HTML_production_timer(mainitem) + '</div>';
						str += '</div>';
					}
					mainitem.subitems.forEach((subitem) => {
						itemsadded += 1;
						str += '<div class="row subitemwrapper">';
						str += '<div class="column doleft subitemcolor">' + subitem.qty + ' X ' + subitem.name + '</div>';
						str += '</div>';
					});
				} else if (mainitem.status === 'VOID') {
					itemsadded += 1;
					str += '<div class="row wrapperdiv ordervoid">';
					str += '<div class="column doleft fullcolumn">' + mainitem.qty + ' ' + mainitem.name + '</div>';
					str += '<span class="production-tag production-tag-void">VOID</span>';
					str += '</div>';
					mainitem.subitems.forEach((subitem) => {
						itemsadded += 1;
						str += '<div class="row ordervoid subitemwrapper">';
						str += '<div class="column">' + subitem.qty + ' X ' + subitem.name + '</div>';
						str += '</div>';
					});
				} else {
				}
			} else {
				if (mainitem.status === 'OPEN' || mainitem.status === 'COMPLETE') {
					itemsadded += 1;
					if (mainitem.iskiddies === 1) {
						str += '<div class="row wrapperdiv wrapperdiv-inline orderkiddies">';
						str += '<div class="column fullcolumn">' + mainitem.qty + ' X ' + mainitem.name + '</div>';
						str += '<div class="itemproductioncontrol-wrap">' + KDS.HTML_production_control(mainitem) + KDS.HTML_production_timer(mainitem) + '</div>';
						if (mainitem.status === 'OPEN'){
							str += '<div class="itemclosecheckbox-wrap"><input class="itemclosecheckbox" type="checkbox" id="close_item_' + mainitem.id + '" onclick="KDS.close_item(' + mainitem.id + ',' + element.id + ');"></div>';
						} else {
							str += '<div class="itemclosecheckbox-wrap" style="text-decoration: line-through;"><input class="itemclosecheckbox" type="checkbox" checked="true" id="close_item_' + mainitem.id + '" onclick="KDS.close_item(' + mainitem.id + ',' + element.id + ');"></div>';
						}
						str += '</div>';
					} else {
						if (mainitem.suborder > 0 && numofitemsforthiscreen > 1) {
							if (mainitem.name.includes('refund')) {
								str += '<div class="row wrapperdiv wrapperdiv-inline ordervoid"' + KDS.getItemColorStyle(mainitem) + '>';
							} else {
								str += '<div class="row wrapperdiv wrapperdiv-inline makegreen"' + KDS.getItemColorStyle(mainitem) + '>';
							}
						} else {
							if (mainitem.name.includes('refund')) {
								str += '<div class="row wrapperdiv wrapperdiv-inline ordervoid"' + KDS.getItemColorStyle(mainitem) + '>';
							} else {
								str += '<div class="row wrapperdiv wrapperdiv-inline mainitemcolor"' + KDS.getItemColorStyle(mainitem) + '>';
							}
						}
						if (mainitem.status === 'OPEN'){
							str += '<div class="column doleft fullcolumn">' + mainitem.qty + ' X ' + mainitem.name + '</div>';
							str += '<div class="itemproductioncontrol-wrap">' + KDS.HTML_production_control(mainitem) + KDS.HTML_production_timer(mainitem) + '</div>';
							str += '<div class="itemclosecheckbox-wrap"><input class="itemclosecheckbox" type="checkbox" id="close_item_' + mainitem.id + '" onclick="KDS.close_item(' + mainitem.id + ',' + element.id + ');"></div>';
							str += '</div>';
						} else {
							//str += '<div style="text-decoration: line-through;" class="column doleft">' + mainitem.qty + ' X ' + mainitem.name + '</div>';
							//str += '</div>';
							str += '<div class="column doleft fullcolumn">' + mainitem.qty + ' X ' + mainitem.name + '</div>';
							str += '<div class="itemproductioncontrol-wrap">' + KDS.HTML_production_control(mainitem) + '</div>';
							str += '<div class="itemclosecheckbox-wrap" style="text-decoration: line-through;"><input class="itemclosecheckbox" type="checkbox" checked="true" id="close_item_' + mainitem.id + '" onclick="KDS.close_item(' + mainitem.id + ',' + element.id + ');"></div>';
							str += '</div>';
						}
					}
					mainitem.subitems.forEach((subitem) => {
						itemsadded += 1;
						str += '<div class="row docenter subitemwrapper">';
						if (mainitem.status === 'OPEN'){
							str += '<div class="column doleft subitemcolor">' + subitem.qty + ' X ' + subitem.name + '</div>';
						} else {
							str += '<div style="text-decoration: line-through;" class="column doleft subitemcolor">' + subitem.qty + ' X ' + subitem.name + '</div>';
						}
						str += '</div>';
					});
				} else if (mainitem.status === 'VOID') {
					itemsadded += 1;
					str += '<div class="row wrapperdiv ordervoid">';
					str += '<div class="column doleft fullcolumn">' + mainitem.qty + ' ' + mainitem.name + '</div>';
					str += '<span class="production-tag production-tag-void">VOID</span>';
					str += '</div>';
					mainitem.subitems.forEach((subitem) => {
						itemsadded += 1;
						str += '<div class="row docenter ordervoid subitemwrapper">';
						str += '<div class="column">' + subitem.qty + ' X ' + subitem.name + '</div>';
						str += '</div>';
					});
				} else {
				}
			}
		} else {
			if (mainitem.status === 'COMPLETE') {
				str = str.replace('nonononono', 'otherelemsaredone');
			}
		}
	});
	str += '</div>';
	str += '<div class="row"><div class="column"></div></div>';
	str += '</div>';
	str += '</div>';
	str += '';

	if (belongonthisscreen) {
		KDS.updateHTML('makesingle', str);
	}
};

// Re-fetches and re-renders the currently open order-detail dialog, if the
// order being refreshed is the one currently shown. Used after a checkbox
// tick/untick so the struck-through state and production timers reflect the
// item's new status immediately, without requiring the dialog to be reopened.
KDS.refreshSingleOrderDialog = function(orderid) {
	if (KDS.activemodal && KDS.activemodalID === orderid) {
		KDS.getSingleOrder(orderid);
	}
};

KDS.flagcheck = (flag) => {
	if (KDS.kdsconfig.flags.hasOwnProperty(flag)){
		return KDS.kdsconfig.flags[flag];
	} else {
		return false;
	}
};

// A pass-mode terminal can always press Close Order - that's its whole job.
// A non-pass screen can only do it for itself (bumping just its own items,
// never anything on another screen) when nobody has been put in charge of
// bumping it instead - see client.passterminal (this section's own "Pass
// Terminal" config setting) and closeOrderScreenDatabase/
// closeOrderForPassTerminal in kds-webserver-main, which enforce the actual
// scoping server-side. Once a section picks a pass terminal, that pass
// terminal's own Close Order press is the only thing that can bump it.
KDS.canCloseOrderHere = () => {
	if (KDS.flagcheck("passmode")) {
		return true;
	}
	return !KDS.kdsconfig.client.passterminal;
};

///////////////////////////////////// END HTML ///////////////////////////////////////////////
module.exports = KDS;
