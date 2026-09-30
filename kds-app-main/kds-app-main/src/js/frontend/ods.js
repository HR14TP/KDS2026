'use strict';
// Order Display Screen - a customer-facing "order ready" board. Deliberately
// a separate, much smaller script from kds.js: this screen doesn't open
// item dialogs, tick checkboxes, or run production timers, it just mirrors
// the same live order feed every kitchen board terminal already gets and
// sorts orders into what a viewer actually cares about: numbers ready for
// collection get the top of the screen, kitchen progress sits below it.
const NATS = require('nats');
const ipc = require('electron').ipcRenderer;

let odsapp = {
	config: null,
	nc: null,
	// Currently-open orders from the live feed, keyed by order.id - this is
	// exactly the same "updates" NATS payload the normal kitchen board
	// reads, just rendered differently and without the per-screen filter
	// (KDS.HTML_order_belongs_to_screen's equivalent) the kitchen board
	// applies, since this screen is meant to show every order regardless of
	// production center. Only ever holds orders that aren't fully
	// announced yet - see markComplete/handleUpdate.
	orders: {},
	// Persisted (survives restart) - see backend.js service-get/save-ods-state.
	completed: [],
	dismissed: [],
};

odsapp.initialize = function() {
	odsapp.ipcINIT();
	ipc.send('service-config', 'all');
	ipc.send('service-get-ods-state', 'all');
	odsapp.startClock();
	// Nothing on this screen depends on a NATS message to stay accurate -
	// the "Xm ago" labels drift out of date on their own. Cheap to just
	// re-render from whatever's already in memory every so often.
	setInterval(() => {
		odsapp.renderBoard();
		odsapp.renderCompleted();
	}, 20000);
};

odsapp.ipcINIT = function() {
	ipc.on('service-config-reply', (event, arg) => {
		odsapp.config = JSON.parse(arg);
		odsapp.natsInit();
	});
	ipc.on('service-get-ods-state-reply', (event, arg) => {
		let state = JSON.parse(arg);
		odsapp.completed = state.completed || [];
		odsapp.dismissed = state.dismissed || [];
		odsapp.renderCompleted();
	});
	ipc.on('config-updates-remotely', (event, arg) => {
		location.reload();
	});
};

odsapp.natsInit = async function() {
	try {
		odsapp.nc = await NATS.connect({ servers: '0.0.0.0:4222' });
		const updatesChannel = odsapp.nc.subscribe('updates');
		odsapp.incomingMessage(updatesChannel);
		if (odsapp.nc.protocol.connected) {
			odsapp.setConnectionStatus(true);
		}
		// Same nudge KDS.ajax_startup uses on the main board - asks the
		// local webserver to (re)publish the current order list on
		// "updates" immediately, rather than waiting for the next change.
		var xmlHttp = new XMLHttpRequest();
		xmlHttp.open('GET', 'http://127.0.0.1:2020/updated', true);
		xmlHttp.send(null);
	} catch (err) {
		console.log(err);
		odsapp.setConnectionStatus(false);
	}
};

odsapp.incomingMessage = async function(sub) {
	const sc = NATS.StringCodec();
	for await (const msg of sub) {
		try {
			let js = JSON.parse(sc.decode(msg.data));
			if (js.messagetype === 'update') {
				odsapp.setConnectionStatus(true);
				odsapp.handleUpdate(js.data);
			} else if (js.messagetype === 'reload') {
				location.reload();
			}
		} catch (err) {
			console.log(err);
		}
	}
};

odsapp.setConnectionStatus = function(connected) {
	let el = document.getElementById('ods_connection_status');
	if (el) {
		el.textContent = connected ? 'CONNECTED' : 'DISCONNECTED';
		el.classList.toggle('ods-status-connected', connected);
		el.classList.toggle('ods-status-disconnected', !connected);
	}
};

odsapp.startClock = function() {
	let tick = () => {
		let el = document.getElementById('ods_clock');
		if (el) {
			el.textContent = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
		}
	};
	tick();
	setInterval(tick, 1000);
};

// A ticket's void/transferred-away items don't represent anything a
// customer is waiting on, so they're excluded from the "how much of this
// ticket is ready" math below.
odsapp.eligibleItems = function(order) {
	return order.items.filter((item) => item.status !== 'VOID' && item.status !== 'TRANSFERRED');
};

odsapp.pushedCount = function(order) {
	return odsapp.eligibleItems(order).filter((item) => item.pushed_to_ods === true).length;
};

odsapp.allEligibleItemsPushed = function(order) {
	let eligible = odsapp.eligibleItems(order);
	if (eligible.length === 0) {
		return false;
	}
	return odsapp.pushedCount(order) === eligible.length;
};

// True whenever the kitchen has actively started at least one item - used
// purely to give a ticket a visibly "cooking" accent versus one that's
// simply queued and not started yet, within the In Production column.
odsapp.hasActiveProduction = function(order) {
	return odsapp.eligibleItems(order).some((item) => item.in_production === true);
};

odsapp.byCreatedAsc = function(a, b) {
	return new Date(a.created) - new Date(b.created);
};

// "2m", "just now", "1h 04m" - used both for how long a ticket has been
// waiting (In Production / Partially Ready) and how long ago something
// became ready, so staff can spot anything that's been sitting too long.
odsapp.timeAgo = function(dateStr) {
	let diffMs = Date.now() - new Date(dateStr).getTime();
	if (!isFinite(diffMs) || diffMs < 0) {
		diffMs = 0;
	}
	let mins = Math.floor(diffMs / 60000);
	if (mins < 1) {
		return 'just now';
	}
	if (mins < 60) {
		return mins + 'm';
	}
	let hrs = Math.floor(mins / 60);
	return hrs + 'h ' + String(mins % 60).padStart(2, '0') + 'm';
};

odsapp.handleUpdate = function(orders) {
	let newIds = {};
	let newByReference = {};
	orders.forEach((order) => {
		newIds[order.id] = true;
		newByReference[order.reference] = order;
	});

	// Any order that dropped out of the live feed entirely was bumped/closed
	// at the pass station (this screen has no other way to close an order) -
	// move it to Complete. Guarded by reference: if an order with the same
	// reference is still present under a different id, this wasn't actually
	// closed (e.g. it was re-ingested), so don't announce it as ready.
	Object.keys(odsapp.orders).forEach((idKey) => {
		if (!newIds[idKey]) {
			let dropped = odsapp.orders[idKey];
			if (!newByReference[dropped.reference]) {
				odsapp.markComplete(dropped);
			}
			delete odsapp.orders[idKey];
		}
	});

	let stillOpen = {};
	orders.forEach((order) => {
		// A staff member removed this one from the board (odsapp.dismissOrder)
		// - stays off this screen even though it's still genuinely open on
		// the kitchen's own board, until the dismissal itself is cleared.
		if (odsapp.dismissed.indexOf(order.reference) !== -1) {
			return;
		}
		if (odsapp.allEligibleItemsPushed(order)) {
			// Every item on this ticket is complete - it's already announced
			// automatically (see KDS close_item), so treat it as ready even
			// though it may still be technically OPEN in the kitchen's own
			// database (nobody bumped it yet).
			odsapp.markComplete(order);
		} else {
			stillOpen[order.id] = order;
		}
	});
	odsapp.orders = stillOpen;
	odsapp.renderBoard();
};

odsapp.markComplete = function(order) {
	// Already showing in Complete, or a staff member already cleared this
	// exact ticket - don't resurrect it just because the feed said the same
	// thing again.
	if (odsapp.dismissed.indexOf(order.reference) !== -1) {
		return;
	}
	if (odsapp.completed.some((entry) => entry.reference === order.reference)) {
		return;
	}
	let entry = {
		reference: order.reference,
		order_forday: order.order_forday,
		table: order.table,
		created: order.created,
		items: odsapp.eligibleItems(order).map((item) => ({ name: item.name, qty: item.qty })),
		completed_at: new Date().toISOString(),
	};
	odsapp.completed.unshift(entry);
	odsapp.persistState();
	odsapp.renderCompleted();
};

// Removes an order from the In Production / Partially Ready columns without
// it ever having been marked complete - for a stale test ticket, a
// duplicate, or anything staff just don't want cluttering this screen
// anymore. This only affects what ODS shows: the order is untouched on the
// kitchen's own board, and staff can still complete/close it there as
// normal. Shares the same "dismissed" list Complete's Clear uses, so it
// also won't reappear here later even once it's actually finished - a
// confirmation guards against a stray tap silently dropping tracking on a
// ticket that's still genuinely being cooked.
odsapp.dismissOrder = function(orderId, orderForday, reference) {
	if (!confirm('Remove order ' + orderForday + ' from this screen? It will not reappear here, even once complete.')) {
		return;
	}
	delete odsapp.orders[orderId];
	odsapp.dismissed.push(reference);
	if (odsapp.dismissed.length > 500) {
		odsapp.dismissed = odsapp.dismissed.slice(odsapp.dismissed.length - 500);
	}
	odsapp.persistState();
	odsapp.renderBoard();
};

odsapp.clearCompleted = function(reference) {
	odsapp.completed = odsapp.completed.filter((entry) => entry.reference !== reference);
	odsapp.dismissed.push(reference);
	// Unbounded growth guard - only the most recent clears matter for
	// suppressing a re-add, not a full history.
	if (odsapp.dismissed.length > 500) {
		odsapp.dismissed = odsapp.dismissed.slice(odsapp.dismissed.length - 500);
	}
	odsapp.persistState();
	odsapp.renderCompleted();
	odsapp.hideSummary();
};

odsapp.clearAllCompleted = function() {
	odsapp.completed.forEach((entry) => {
		if (odsapp.dismissed.indexOf(entry.reference) === -1) {
			odsapp.dismissed.push(entry.reference);
		}
	});
	odsapp.completed = [];
	if (odsapp.dismissed.length > 500) {
		odsapp.dismissed = odsapp.dismissed.slice(odsapp.dismissed.length - 500);
	}
	odsapp.persistState();
	odsapp.renderCompleted();
	odsapp.hideSummary();
};

odsapp.persistState = function() {
	ipc.send('service-save-ods-state', JSON.stringify({
		completed: odsapp.completed,
		dismissed: odsapp.dismissed,
	}));
};

odsapp.htmlescape = function(str) {
	return String(str).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
};

// ========================= RENDERING ========================= //

odsapp.renderBoard = function() {
	let inProduction = [];
	let partial = [];
	Object.values(odsapp.orders).forEach((order) => {
		if (odsapp.pushedCount(order) > 0) {
			partial.push(order);
		} else {
			inProduction.push(order);
		}
	});
	inProduction.sort(odsapp.byCreatedAsc);
	partial.sort(odsapp.byCreatedAsc);
	odsapp.renderColumn('ods_inproduction', inProduction, 'Nothing in production', (order) => odsapp.tileHTML(order, false));
	odsapp.renderColumn('ods_partial', partial, 'Nothing partially ready', (order) => odsapp.tileHTML(order, true));
};

odsapp.renderColumn = function(containerId, list, emptyMessage, itemRenderer) {
	let container = document.getElementById(containerId);
	odsapp.setCount(containerId + '_count', list.length);
	if (!container) {
		return;
	}
	if (list.length === 0) {
		container.innerHTML = '<div class="ods-empty">' + emptyMessage + '</div>';
		return;
	}
	container.innerHTML = list.map(itemRenderer).join('');
};

odsapp.setCount = function(elementId, count) {
	let el = document.getElementById(elementId);
	if (el) {
		el.textContent = count;
	}
};

odsapp.tileHTML = function(order, isPartial) {
	let eligible = odsapp.eligibleItems(order);
	let pushed = odsapp.pushedCount(order);
	let cooking = odsapp.hasActiveProduction(order);
	let pct = eligible.length > 0 ? Math.round((pushed / eligible.length) * 100) : 0;
	let classes = ['ods-tile'];
	classes.push(isPartial ? 'ods-tile-partial' : (cooking ? 'ods-tile-cooking' : 'ods-tile-queued'));
	let refAttr = odsapp.htmlescape(order.reference).replace(/'/g, "\\'");
	let html = '<div class="' + classes.join(' ') + '">';
	html += '<button class="ods-tile-dismiss" onclick="odsapp.dismissOrder(' + order.id + ',\'' + odsapp.htmlescape(order.order_forday) + '\',\'' + refAttr + '\')" title="Remove from this screen">&times;</button>';
	html += '<div class="ods-tile-head">';
	html += '<div class="ods-tile-number">' + odsapp.htmlescape(order.order_forday) + '</div>';
	if (isPartial) {
		html += '<div class="ods-tile-progress">' + pushed + ' / ' + eligible.length + '</div>';
	} else if (cooking) {
		html += '<div class="ods-tile-badge ods-tile-badge-cooking">COOKING</div>';
	} else {
		html += '<div class="ods-tile-badge ods-tile-badge-queued">QUEUED</div>';
	}
	html += '</div>';
	if (isPartial) {
		html += '<div class="ods-tile-progress-track"><div class="ods-tile-progress-fill" style="width:' + pct + '%"></div></div>';
	}
	html += '<div class="ods-tile-items">';
	eligible.forEach((item) => {
		let sent = item.pushed_to_ods === true;
		html += '<div class="ods-item' + (sent ? ' ods-item-sent' : '') + '">';
		html += '<span class="ods-item-mark">' + (sent ? '&#10003;' : '&#9679;') + '</span>';
		html += odsapp.htmlescape(item.qty) + ' &times; ' + odsapp.htmlescape(item.name);
		html += '</div>';
	});
	html += '</div>';
	html += '<div class="ods-tile-age">Waiting ' + odsapp.timeAgo(order.created) + '</div>';
	html += '</div>';
	return html;
};

odsapp.renderCompleted = function() {
	let container = document.getElementById('ods_completed');
	odsapp.setCount('ods_completed_count', odsapp.completed.length);
	if (!container) {
		return;
	}
	let list = odsapp.completed.slice().sort(odsapp.byCreatedAsc);
	if (list.length === 0) {
		container.innerHTML = '<div class="ods-empty ods-empty-band">No orders ready yet</div>';
		return;
	}
	let html = '';
	list.forEach((entry) => {
		let refAttr = odsapp.htmlescape(entry.reference).replace(/'/g, "\\'");
		let isNew = (Date.now() - new Date(entry.completed_at).getTime()) < 8000;
		html += '<div class="ods-ready-tile' + (isNew ? ' ods-ready-new' : '') + '" onclick="odsapp.showSummary(\'' + refAttr + '\')">';
		html += '<button class="ods-ready-clear" onclick="event.stopPropagation();odsapp.clearCompleted(\'' + refAttr + '\')" title="Clear">&times;</button>';
		html += '<div class="ods-ready-number">' + odsapp.htmlescape(entry.order_forday) + '</div>';
		html += '<div class="ods-ready-age">' + odsapp.timeAgo(entry.completed_at) + '</div>';
		html += '</div>';
	});
	container.innerHTML = html;
};

// ========================= ORDER SUMMARY MODAL ========================= //

odsapp.showSummary = function(reference) {
	let entry = odsapp.completed.find((e) => e.reference === reference);
	if (!entry) {
		return;
	}
	let html = '<div class="ods-modal-head">';
	html += '<div class="ods-modal-number">Order ' + odsapp.htmlescape(entry.order_forday) + '</div>';
	html += '<button class="ods-modal-close" onclick="odsapp.hideSummary()">&times;</button>';
	html += '</div>';
	if (entry.table) {
		html += '<div class="ods-modal-sub">Table: ' + odsapp.htmlescape(entry.table) + '</div>';
	}
	html += '<div class="ods-modal-items">';
	entry.items.forEach((item) => {
		html += '<div class="ods-modal-item">' + odsapp.htmlescape(item.qty) + ' &times; ' + odsapp.htmlescape(item.name) + '</div>';
	});
	html += '</div>';
	html += '<button class="ods-modal-clear-button" onclick="odsapp.clearCompleted(\'' + odsapp.htmlescape(reference).replace(/'/g, "\\'") + '\')">CLEAR THIS ORDER</button>';

	document.getElementById('ods_modal_content').innerHTML = html;
	document.getElementById('ods_modal_overlay').classList.add('ods-modal-open');
};

odsapp.hideSummary = function() {
	let overlay = document.getElementById('ods_modal_overlay');
	if (overlay) {
		overlay.classList.remove('ods-modal-open');
	}
};
