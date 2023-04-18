const http = require("http");
const ws = require("ws");

const sink = () => {};

function getWebSocketDebuggerURL(url) {
  return new Promise((resolve, reject) => {
    let body = ""
    const req = http.get(url, res => {
      res.on("data", chunk => {
        body += chunk;
      });
      res.on("end", () => {
        const targets = JSON.parse(body);
        resolve(targets[0].webSocketDebuggerUrl);
      });
    });
    req.on("error", reject);
  });
}

class DebuggerClient {
  constructor(hostname, port) {
    this.hostname = hostname;
    this.port = port;

    this.client = null;
    this.msgId = 0;
    this.scripts = [];

    this.onOpen = this.onOpen.bind(this);
    this.onMessage = this.onMessage.bind(this);
    this.onError = this.onError.bind(this);

    this.onResponse = sink;

    this.init();
  }

  async init() {
    const baseURL = "http://" + this.hostname + ":" + this.port + "/json";
    const debuggerURL = await getWebSocketDebuggerURL(baseURL);
    this.client = new ws.WebSocket(debuggerURL);
    this.client.on("open", this.onOpen);
    this.client.on("message", this.onMessage);
    this.client.on("error", this.onError);
  }

  async onOpen() {
    await this.sendMessage({ method: "Runtime.enable" });
    await this.sendMessage({ method: "Debugger.enable" });
    await this.sendMessage({ method: "Runtime.runIfWaitingForDebugger" });
  }

  onMessage(buff) {
    const str = buff.toString("utf8");
    const msg = JSON.parse(str);
    if ("id" in msg) {
      this.onResponse(msg);
      this.onResponse = sink;
      return;
    }
    if ("method" in msg) {
      if (msg.method === "Debugger.scriptParsed") {
        return this.onScriptParsed(msg);
      }
      if (msg.method === "Debugger.paused") {
        return this.onPaused(msg);
      }
    }
    // console.log("unknown message", JSON.stringify(msg, null, 4));
  }

  onError(err) {
    console.error(err);
  }

  sendMessage(data) {
    const msg = { ...data, id: this.msgId++ };
    const str = JSON.stringify(msg);
    this.client.send(str);
    return new Promise(resolve => {
      this.onResponse = resolve;
    });
  }

  // =========================================

  onScriptParsed(msg) {
    this.scripts.push(msg.params);
  }

  async onPaused() {
    console.log("(ndb) list main.js:13")
    await this.getScriptSource("main.js", 13);
    console.log("(ndb) breakpoint main.js:13");
    await this.setBreakpoint("main.js", 13);
  }
  
  // =========================================
 
  async getScriptSource(filename, lineNumber) {
    const script = this.findScript(filename);
    if (script === undefined) {
      return;
    }
    const msg = await this.sendMessage({
      method: "Debugger.getScriptSource",
      params: { scriptId: script.scriptId }
    });
    const lines = msg.result.scriptSource.split("\n");
    const iFrom = Math.max(lineNumber - 5, 0);
    const iTo = Math.min(iFrom + 10, lines.length - 1);
    console.log("Showing: " + script.url + ":" + lineNumber);
    for (let i = iFrom; i <= iTo; i++) {
      const j = i + 1;
      let s = "";
      if (j === lineNumber ) {
        s = " -> ";
      } else {
        s = "    ";
      }
      // TODO calculate required pad size
      s += ("" + j).padStart(4, "  ");
      s += ":   ";
      s += lines[i];
      console.log(s);
    }
  }

  async setBreakpoint(filename, lineNumber) {
    const script = this.findScript(filename);
    if (script === undefined) {
      return;
    }
    await this.sendMessage({
      method: "Debugger.setBreakpoint",
      params: { location: { scriptId: script.scriptId, lineNumber } }
    });
  }

  // =========================================

  findScript(filename) {
    for (const script of this.scripts) {
      if (script.url.endsWith(filename)) {
        return script;
      }
    }
  }
}

new DebuggerClient("127.0.0.1", "9229");
