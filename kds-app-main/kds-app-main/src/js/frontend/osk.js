// Lightweight on-screen keyboard for the client config screen. Kiosk
// terminals here don't reliably have a physical keyboard attached, so text
// fields (client name, IP addresses, Twilio tokens, ...) need a touch way
// to type. Delegated to the whole document rather than wired per-field, so
// every current and future text/number input on this page gets it for free.
(function () {
	var keyboardEl = null;
	var activeInput = null;
	var shiftOn = false;
	var hideTimer = null;

	var LETTER_ROWS = [
		['1', '2', '3', '4', '5', '6', '7', '8', '9', '0'],
		['q', 'w', 'e', 'r', 't', 'y', 'u', 'i', 'o', 'p'],
		['a', 's', 'd', 'f', 'g', 'h', 'j', 'k', 'l'],
		['z', 'x', 'c', 'v', 'b', 'n', 'm'],
	];
	var SYMBOL_ROW = ['.', ',', '-', '_', '@', ':', '/'];

	function isTypeable(el) {
		if (!el || el.disabled || el.readOnly) {
			return false;
		}
		if (el.tagName === 'TEXTAREA') {
			return true;
		}
		if (el.tagName !== 'INPUT') {
			return false;
		}
		var type = (el.type || 'text').toLowerCase();
		return type === 'text' || type === 'number' || type === 'password' || type === 'tel' || type === 'search' || type === 'email';
	}

	function build() {
		var kb = document.createElement('div');
		kb.id = 'onscreenkeyboard';
		kb.className = 'onscreenkeyboard';
		var html = '';
		LETTER_ROWS.forEach(function (row, i) {
			html += '<div class="osk-row">';
			if (i === 3) {
				html += '<button type="button" class="osk-key osk-key-wide" data-osk="shift">&#8679;</button>';
			}
			row.forEach(function (ch) {
				html += '<button type="button" class="osk-key" data-osk="char" data-char="' + ch + '">' + ch + '</button>';
			});
			if (i === 3) {
				html += '<button type="button" class="osk-key osk-key-wide" data-osk="backspace">&#9003;</button>';
			}
			html += '</div>';
		});
		html += '<div class="osk-row">';
		SYMBOL_ROW.forEach(function (ch) {
			html += '<button type="button" class="osk-key osk-key-symbol" data-osk="char" data-char="' + ch + '">' + ch + '</button>';
		});
		html += '<button type="button" class="osk-key osk-key-space" data-osk="space">SPACE</button>';
		html += '<button type="button" class="osk-key osk-key-done" data-osk="done">DONE</button>';
		html += '</div>';
		kb.innerHTML = html;
		document.body.appendChild(kb);
		// Keys must never actually take focus themselves - taking it away
		// from the field being typed into would blur it (closing the
		// keyboard and losing the insertion point) on every single tap.
		kb.addEventListener('mousedown', function (e) { e.preventDefault(); });
		kb.addEventListener('touchstart', function (e) { e.preventDefault(); }, { passive: false });
		kb.addEventListener('click', onKeyClick);
		return kb;
	}

	function updateShiftVisual() {
		if (!keyboardEl) {
			return;
		}
		keyboardEl.classList.toggle('osk-shift', shiftOn);
		var chars = keyboardEl.querySelectorAll('[data-osk="char"]');
		for (var i = 0; i < chars.length; i++) {
			var ch = chars[i].getAttribute('data-char');
			if (ch.length === 1 && ch.match(/[a-z]/i)) {
				chars[i].textContent = shiftOn ? ch.toUpperCase() : ch.toLowerCase();
			}
		}
	}

	function insertAtCursor(text) {
		if (!activeInput) {
			return;
		}
		var start = activeInput.selectionStart;
		var end = activeInput.selectionEnd;
		if (start === null || end === null) {
			start = end = activeInput.value.length;
		}
		var value = activeInput.value;
		activeInput.value = value.slice(0, start) + text + value.slice(end);
		var newPos = start + text.length;
		activeInput.setSelectionRange(newPos, newPos);
		// Let the page's own change/input handlers (validation, save-on-type
		// wiring, etc.) see this exactly like a real keystroke would.
		activeInput.dispatchEvent(new Event('input', { bubbles: true }));
	}

	function backspace() {
		if (!activeInput) {
			return;
		}
		var start = activeInput.selectionStart;
		var end = activeInput.selectionEnd;
		var value = activeInput.value;
		if (start === null || end === null) {
			start = end = value.length;
		}
		if (start === end && start > 0) {
			start -= 1;
		}
		activeInput.value = value.slice(0, start) + value.slice(end);
		activeInput.setSelectionRange(start, start);
		activeInput.dispatchEvent(new Event('input', { bubbles: true }));
	}

	function onKeyClick(e) {
		var key = e.target.closest('[data-osk]');
		if (!key || !activeInput) {
			return;
		}
		var action = key.getAttribute('data-osk');
		if (action === 'char') {
			var ch = key.getAttribute('data-char');
			insertAtCursor(shiftOn && ch.match(/[a-z]/i) ? ch.toUpperCase() : ch);
		} else if (action === 'space') {
			insertAtCursor(' ');
		} else if (action === 'backspace') {
			backspace();
		} else if (action === 'shift') {
			shiftOn = !shiftOn;
			updateShiftVisual();
		} else if (action === 'done') {
			hide();
			activeInput.blur();
			activeInput = null;
		}
	}

	function show(input) {
		clearTimeout(hideTimer);
		activeInput = input;
		if (!keyboardEl) {
			keyboardEl = build();
		}
		keyboardEl.classList.add('osk-open');
		document.body.classList.add('osk-active');
		setTimeout(function () {
			input.scrollIntoView({ block: 'center', behavior: 'smooth' });
		}, 60);
	}

	function hide() {
		if (keyboardEl) {
			keyboardEl.classList.remove('osk-open');
		}
		document.body.classList.remove('osk-active');
	}

	document.addEventListener('focusin', function (e) {
		if (isTypeable(e.target)) {
			show(e.target);
		}
	});

	document.addEventListener('focusout', function (e) {
		if (!isTypeable(e.target)) {
			return;
		}
		// A tap on a keyboard key never moves focus (mousedown/touchstart
		// above prevent that), so the only way this fires with the
		// keyboard still relevant is focus moving straight to another
		// typeable field - give that a moment to arrive before hiding.
		clearTimeout(hideTimer);
		hideTimer = setTimeout(function () {
			if (!isTypeable(document.activeElement)) {
				hide();
				activeInput = null;
			}
		}, 80);
	});
})();
