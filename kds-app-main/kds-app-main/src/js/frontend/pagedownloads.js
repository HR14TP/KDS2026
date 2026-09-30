'use strict';

let pageDownloads = {
	name: 'pagedownloads',
	id: 'pagedownloads',
	func: {},
	vars: {}
};

pageDownloads.func.closeModal = () => {    
    window.close();  
}

module.exports = pageDownloads;