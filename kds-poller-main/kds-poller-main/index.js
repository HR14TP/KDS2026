const fs = require('fs');
const AWS = require('aws-sdk');
require('dotenv').config();

const BUCKET_NAME = "kdsdailybackup";
const APP_NAME = "poller";
const FILE_PERMISSION = 'public-read'

console.log(process.env.AWS_ACCESS_KEY_ID);

const s3bucket = new AWS.S3({
    accessKeyId: process.env.AWS_ACCESS_KEY_ID,
    secretAccessKey: process.env.AWS_SECRET_ACCESS_KEY
});

function uploadToS3(fileName , baseDIR){
  const readStream = fs.createReadStream(fileName);

  const params = {
    Bucket: BUCKET_NAME,
    Key: "releases/" + baseDIR + "/" + fileName,
    Body: readStream,
    ACL: FILE_PERMISSION
  };

  return new Promise((resolve, reject) => {
    s3bucket.upload(params, function(err, data) {
      readStream.destroy();
      
      if (err) {
        return reject(err);
      }
      
      return resolve(data);
    });
  });
}

fs.readdir('files/', function (err, files) {
    console.log(files);
    files.forEach(function (file, index) {
        uploadToS3('files/' + file , APP_NAME).then(done => {
            console.log(done);            
        });
    });    
});