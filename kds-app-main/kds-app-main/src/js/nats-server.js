'use strict';
const { connect,JSONCodec,StringCodec } = require("nats");

let natsServer = {
    _jcodec: JSONCodec,
    _strcodec: StringCodec,
    _connection: null,
    globalInit: () => {
        natsServer._jcodec = new JSONCodec();
        natsServer._strcodec = new StringCodec();  
    },
    backendInit: async () => {
        try {
            natsServer.globalInit();
            natsServer._connection = await connect({
                servers: "nats.tabletpos.co.za:4222",
                token: "KDS2021@Tabletpos",                            
            });
            natsServer.handleClose();
            natsServer.timerFunctions();            
            return true;                    
        }catch(err){
            console.log(`error connecting : ${JSON.stringify(err)}`);
            return false;
        }
    },
    backendInitCheck: async () => {
        console.log("ping");
        if (natsServer._connection.isClosed()) {
            try {         
                console.log("pong");   
                natsServer.globalInit();
                natsServer._connection = await connect({
                    servers: "nats.tabletpos.co.za:4222",
                    token: "KDS2021@Tabletpos",                            
                });  
                natsServer.handleClose(); 
                return true;
            }catch(err){
                console.log(`error connecting : ${JSON.stringify(err)}`);
                return false;
            }
        }
    },
    subScribe:(topic , callback) => {
        console.log("SUBSCRIBED TO : " + topic);
        const eventTopic = natsServer._connection.subscribe(topic);
        callback(eventTopic);
    },
    doPublish:(topic , message) => {
        console.log("PUBLISHED TO : " + topic);
        natsServer._connection.publish(topic, natsServer._strcodec.encode(JSON.stringify(message)));
    },
    doRequest: async(topic , message = {} , callback) => {    
        console.log("REQUESTED FROM : " + topic);
        if (natsServer._connection !== null){    
            await natsServer._connection.request(topic, natsServer._strcodec.encode(JSON.stringify(message)), { timeout: 10000 }).then((m) => {
                try{
                    callback(m);
                }catch(err){
                    console.log(err);
                }                    
            }).catch((err) => {
                console.log(`problem with request: ${err.message}`);
            });  
        }
    },
    handleClose: async() => {        
        await natsServer._connection.closed().then((err) => {
            console.log("Connection is Closed");
            natsServer.backendInitCheck();          
        });
    },
    timerFunctions:() => {
        setInterval(natsServer.backendInitCheck, 60000);
    },
};

module.exports = natsServer;