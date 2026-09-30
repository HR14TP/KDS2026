'use strict';
const { connect,JSONCodec,StringCodec } = require("nats");

let natslocal = {
    _jcodec: JSONCodec,
    _strcodec: StringCodec,
    _connection: null,
    globalInit: () => {
        natslocal._jcodec = new JSONCodec();
        natslocal._strcodec = new StringCodec();  
    },
    backendInit: async () => {
        try {
            natslocal._connection = await connect({servers: "0.0.0.0:4222"});
            //const eventTopic = natslocal._connection.subscribe("events");
            //eventsCallback(eventTopic);
            return true;
        }catch(err) {
            console.log(`error connecting : ${JSON.stringify(err)}`);
            return false;
        }
    },
    subScribe:(topic , callback) => {
        const eventTopic = natslocal._connection.subscribe(topic);
        callback(eventTopic);
    }
};

module.exports = natslocal;