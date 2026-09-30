dialog = {
	name: 'dialogbox',
	id: 'dialogbox'
};

dialog.init = function() {
	var width = screen.width + document.documentElement.scrollLeft;
	var left = parseInt(screen.width / 2 - 380, 10);
	var top = parseInt(10, 10);
	var right = 10;
	var layer = document.createElement('div');
	layer.style.zIndex = 490;
	layer.id = 'popupframe';
	layer.style.display = 'none';
	layer.style.position = 'absolute';
	layer.style.top = '0px';
	layer.style.left = '0px';
	layer.style.height = document.documentElement.scrollHeight + 'px';
	layer.style.width = width + 'px';
	layer.style.backgroundColor = 'gray';
	layer.style.opacity = '.6';
	layer.style.filter += 'progid:DXImageTransform.Microsoft.Alpha(opacity=50)';
	document.body.appendChild(layer);
	var div = document.createElement('div');
	div.style.zIndex = 491;
	div.id = 'popupbox';
	div.style.display = 'none';
	div.style.height = 'auto';
	div.style.position = navigator.userAgent.indexOf('MSIE 6') > -1 ? 'absolute' : 'fixed';
	div.style.top = top + 'px';
	div.style.right = right + 'px';
	div.style.width = 'auto';
	div.style.backgroundColor = 'white';
	div.style.border = '3px solid #213744';
	div.style.padding = '5px';
	div.style.overflowY = 'scroll';
	div.style.overflowX = 'hidden';
	div.style.maxHeight = screen.height - 200 + 'px';
	document.body.appendChild(div);
	var p = document.createElement('p');
	div.appendChild(p);
	var SearchBarDiv = document.createElement('div');
	SearchBarDiv.id = 'divSearchBar';
	SearchBarDiv.style.display = 'none';
	SearchBarDiv.innerHTML = '';
	SearchBarDiv.style.textAlign = 'center';
	SearchBarDiv.style.float = 'left';
	div.appendChild(SearchBarDiv);
};

dialog.resetcontent = function() {
	document.getElementById('divSearchBar').innerHTML = '';
};

dialog.hide = function() {
	document.getElementById('popupframe').style.display = 'none';
	document.getElementById('popupbox').style.display = 'none';
	document.getElementById('divSearchBar').style.display = 'none';
	dialog.originalresize();
};

dialog.show = function() {
	document.getElementById('popupframe').style.display = 'block';
	document.getElementById('popupbox').style.display = 'block';
	document.getElementById('divSearchBar').style.display = 'block';
};

dialog.showcustom = function(innercontent) {
	var width = screen.width + document.documentElement.scrollLeft;
	document.getElementById('divSearchBar').innerHTML = innercontent;
	document.getElementById('popupframe').style.display = 'block';
	document.getElementById('popupframe').style.width = width + 'px';
	document.getElementById('popupbox').style.display = 'block';
	document.getElementById('divSearchBar').style.display = 'block';
};

dialog.resize = function() {
	var left = parseInt(screen.width / 2 - 70, 10);
	var top = parseInt(screen.height / 2 - 40, 10);
	document.getElementById('popupbox').style.top = top + 'px';
	document.getElementById('popupbox').style.left = left + 'px';
	document.getElementById('popupbox').style.width = '150px';
	document.getElementById('popupbox').style.height = '80px';
	document.getElementById('divSearchBar').style.width = 'auto';
	document.getElementById('divSearchBar').style.height = 'auto';
};

dialog.resizeXY = function(width, height) {
	var left = parseInt(screen.width / 2 - width / 2, 10);
	var top = parseInt(screen.height / 2 - height / 2, 10);
	document.getElementById('popupbox').style.top = top + 'px';
	document.getElementById('popupbox').style.left = left + 'px';
	document.getElementById('popupbox').style.width = width + 'px';
	document.getElementById('popupbox').style.height = height + 'px';
	document.getElementById('divSearchBar').style.width = 'auto';
	document.getElementById('divSearchBar').style.height = 'auto';
};

// Sizes the popup as a share of the viewport instead of fixed pixels, so it
// scales sensibly whether this KDS is running on a small kiosk touchscreen
// or a full desktop monitor, and stays correct if the window itself is
// resized - resizeXY's fixed px box doesn't adapt to either. Capped with
// max-width/max-height so it doesn't stretch into unreadably wide lines on
// a large, low-density display.
dialog.resizeRelative = function(widthVW, heightVH, maxWidthPx, maxHeightPx) {
	var box = document.getElementById('popupbox');
	box.style.top = '50%';
	box.style.left = '50%';
	box.style.right = 'auto';
	box.style.transform = 'translate(-50%, -50%)';
	box.style.width = widthVW + 'vw';
	box.style.height = heightVH + 'vh';
	box.style.maxWidth = (maxWidthPx || 1400) + 'px';
	box.style.maxHeight = (maxHeightPx || 900) + 'px';
	document.getElementById('divSearchBar').style.width = '100%';
	document.getElementById('divSearchBar').style.height = '100%';
};

dialog.originalresize = function() {
	var left = parseInt(screen.width / 2 - 380, 10);
	var top = parseInt(10, 10);
	var right = 10;
	document.getElementById('popupbox').style.top = top + 'px';
	document.getElementById('popupbox').style.right = right + 'px';
	document.getElementById('popupbox').style.width = 'auto';
	document.getElementById('popupbox').style.height = 'auto';
};

dialog.showspinner = function() {
	var html = '<div style="text-align:center;" class="lds-hourglass"></div>';	    
    dialog.resetcontent();
    dialog.resizeXY(100,150);
    dialog.showcustom(html);
};

dialog.showSpinnerDialog = (message) => {
	var html = '<div><strong id="statusfield">Loading Please Wait...</strong></div><div class="lds-hourglass"></div>';
	dialog.resetcontent();
	dialog.resizeXY(200,200);
	dialog.showcustom(html);
};

dialog.keyPressedCallback = (event) => {

};

dialog.keyPressed = (event , cb) => {
	//console.log(event);
	//console.log(cb);
	document.removeEventListener("keypress", dialog.keyPressed);	
	dialog.keyPressedCallback(event);
	dialog.hide();
};

dialog.showSpinnerCustomDialog = (message, cb = null) => {
	var html = '<div><strong id="statusfield">'+message+'</strong></div><div class="lds-hourglass"></div>';
	dialog.resetcontent();
	dialog.resizeXY(300,200);
	dialog.showcustom(html);
	if (cb !== null) {
		dialog.keyPressedCallback = cb;
	}	
	document.addEventListener("keypress", dialog.keyPressed);	
};

dialog.askPassword = (label , cb) => {
	var html = '<div><label>' + label + '</label>';
	html += '<input type="password" id="dialogpassword" />';
	html += '<button onclick="dialog.askPasswordCallback();" class="button button-light-blue">OK</button><br />';
	html += '<button onclick="dialog.hide();" class="button button-red">CANCEL</button></div>';
	dialog.resetcontent();
	dialog.resizeXY(200,250);
	dialog.showcustom(html);
	dialog.askPasswordCallback = () => {
		let thepass = document.getElementById('dialogpassword').value;
		cb(thepass);
	};
};