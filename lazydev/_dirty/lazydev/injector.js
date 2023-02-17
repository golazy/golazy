


class Commander {
    constructor(){
        this.setup()
    }

    setup() {
        this.ws = new WebSocket(`ws://${document.location.host}/golazy/commands`);
        this.ws.addEventListener('message', this.onMessage)
        this.ws.addEventListener('open', this.onOpen)
        this.ws.addEventListener('close', this.onClose)
        this.ws.addEventListener('error', this.onError)
    }

    onError = (event) => {
        debugger
    }

    onOpen = (event) => {
        debugger
    }
    onClose = (event) => {
        debugger
    }

    onMessage=(event) => {
        const msg = JSON.parse(event.data)
        switch (msg.Command) {
            case "reload":
                if(typeof Turbo !== 'undefined'){
                    Turbo.visit(document.location.pathname + document.location.search)
                } else {
                    document.location.reload()
                }
                break;
            case "tick":
                break;
            default:
                debugger
        }

    }
}

new Commander();