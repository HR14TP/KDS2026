const os = require('os');

generic = {
    name: 'generic',
    id: 'generic',
    platform: os.platform(),
    arch: os.arch(),
};
generic.getOSname = (filename) => {
    if (generic.platform === "linux") {
        return filename;
    } else if (generic.platform === "win32"){
        return filename + ".exe";
    } else if (generic.platform === "win64"){
        return filename + ".exe";
    } else if (generic.platform === "darwin"){
        return filename;
    } else {
        return filename;
    }
};
generic.getArchFilename = () => {
    console.log(generic.platform);
    console.log(generic.arch);
    if (generic.platform === "win32" || generic.platform === "win64") {
        return "windows";
    } else if (generic.arch === "x64" && generic.platform === "linux") {
        return "linux";
    } else if (generic.arch === "x64" && generic.platform === "darwin") {
        return "osx";
    } else if (generic.arch === "arm64" && generic.platform === "darwin") {
        return "osx";
    } else {
        return "arm7";
    }
};
generic.getNatsFilename = () => {
    if (generic.platform === "win32" || generic.platform === "win64") {
        return "windows.exe";
    } else if (generic.arch === "x64" && generic.platform === "linux") {
        return "linux";
    } else if (generic.arch === "x64" && generic.platform === "darwin") {
        return "osx";
    } else if (generic.arch === "arm64" && generic.platform === "darwin") {
        return "osx";
    } else if (generic.arch === "arm" && generic.platform === "linux") {
        return "arm7";
    } else {
        return "arm7";
    }
};
module.exports = generic;
