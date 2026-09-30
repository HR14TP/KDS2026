'use-strict';
let KDSAJAX = {
    name: 'KDSAJAX',
    id: 'KDSAJAX',
};
KDSAJAX.getSingleOrder = function(id) {
    KDS.ajax_getsingleorder(id);
};
KDSAJAX.ajax_startup = function(id) {
    var xmlHttp = new XMLHttpRequest();
    xmlHttp.open("GET", "http://127.0.0.1:2020/updated", true); // true for asynchronous
    xmlHttp.send(null);
};
KDSAJAX.ajax_getsingleorder = function(id) {
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
module.exports = KDSAJAX;
