const crypto = require('crypto');
const path = require('path');
const fs = require('fs');

const kdshwid = {
    name: 'kdshwid',
    id: 'kdshwid',
    filePath: '',
    getHWID: () => {
        if (kdshwid.checkHWID()){
            const rawdata = fs.readFileSync(kdshwid.filePath);
            return JSON.parse(rawdata);                        
        } else {
            kdshwid.setHWID();
            const rawdata = fs.readFileSync(kdshwid.filePath);
            return JSON.parse(rawdata);
        }
    },
    checkHWID: () => {
        try{            
            return fs.existsSync(kdshwid.filePath);
        } catch(err) {
            console.log(err);
            return false;
        }
    },
    setHWID: () => {
        let hwid = {
            "hwid" : crypto.randomBytes(64).toString("hex")
        };        
        try{
            fs.writeFileSync(kdshwid.filePath, JSON.stringify(hwid, null, 4));            
            console.log("HWID Generated : " + kdshwid.filePath);
        }catch(err){
            console.log(err);
        } 
    },
    initHWID: (userDataPath) => {
        kdshwid.filePath = path.join(userDataPath,'config','kdshwid.json');
        console.log(kdshwid.filePath);
    }
};

module.exports = kdshwid;