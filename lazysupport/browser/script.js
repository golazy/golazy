
function timeout(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

function get() {
    var location = document.location;
    var uri = {
        spec: location.href,
        host: location.host,
        prePath: location.protocol + "//" + location.host, // TODO This is incomplete, needs username/password and port
        scheme: location.protocol.substr(0, location.protocol.indexOf(":")),
        pathBase: location.protocol + "//" + location.host + location.pathname.substr(0, location.pathname.lastIndexOf("/") + 1)
    };

    var readabilityObj = new Readability(uri, document);
    return readabilityObj.parse();

}


JSON.stringify(get())