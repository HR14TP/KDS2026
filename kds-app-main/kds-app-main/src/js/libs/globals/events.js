const LocalEvents = {
    EVENT_REGISTERED: "kds-local-event-registered",    
    EVENT_QUIT: "kds-local-event-quit",
    EVENT_STARTUP: "kds-local-startup",
    
    EVENT_KDS_ORDERS: "kds-local-event-kds-orders", // Orders incoming from the webserver
    EVENT_RECALL_ORDER: "kds-local-event-kds-recall-order",

    EVENT_GETORDERS: "kds-local-event-kds-getorders", // IPC - Ask webserver to send current orders
    NATS_GETORDERS: "getorders", // NATS - Ask webserver for all current orders

    EVENT_PLAY_SOUND: "kds-play-sound",
};
const LocalRequests = {
    REQUEST_HWID: "kds-local-request-hwid",
    REQUEST_CONFIG: "kds-local-request-config",
    REQUEST_CONFIG_SAVE: "kds-local-request-config-save",
    REQUEST_REGISTER: "kds-local-request-register",
    REQUEST_PASTORDERS: "kds-local-request-pastorders",      
    REQUEST_GETSINGLEORDER: "kds-local-request-getsingleorder", 
    REQUEST_CLOSEORDER: "kds-local-request-closeorder",
    REQUEST_RECALLORDER: "kds-local-request-recallorder",       
};
const GlobalEvents = {
    EVENT_REGISTERED: "kds-global-event-registered",
        
    EVENT_NEWORDER: "kds-global-event-neworder",
    EVENT_CLOSEORDER: "kds-global-event-closeorder",
    EVENT_CLOSEITEM: "kds-global-event-closeitem",
    EVENT_RECALLORDER: "kds-global-event-recallorder",

    EVENT_HOOKS_CLOSEORDER: "kds-global-event-hook-closeorder",
    EVENT_HOOKS_RECALLORDER: "kds-global-event-hook-recallorder",        
};

module.exports = { LocalEvents , LocalRequests , GlobalEvents }