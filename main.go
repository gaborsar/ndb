package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"nhooyr.io/websocket"
	"os"
	"strings"
)

type DebuggerTarget struct {
	Url                  string `json:"url"`
	WebSocketDebuggerUrl string `json:"webSocketDebuggerUrl"`
}

type RequestMessage struct {
	Id     int    `json:"id"`
	Method string `json:"method"`
}

type ResponseMessage struct {
	Id     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type ScriptParsedParams struct {
	ScriptId string `json:"scriptId"`
}

type ScriptSourceResult struct {
	ScriptSource string `json:"scriptSource"`
}

const (
	colorPrompt = "\033[0;36m"
	colorNumber = "\033[0;32m"
	colorNone   = "\033[0m"
)

func main() {
	res, err := http.Get("http://127.0.0.1:9229/json")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var targets []DebuggerTarget
	err = json.Unmarshal(body, &targets)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if len(targets) == 0 {
		os.Exit(1)
	}
	target := targets[0]

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, _, err := websocket.Dial(ctx, target.WebSocketDebuggerUrl, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	d := NewDebugger(ctx, conn)
	d.Start()

	select {
	case <-d.done:
		break
	case err = <-d.err:
		fmt.Println(err)
		break
	}

	err = conn.Close(websocket.StatusNormalClosure, "")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type Debugger struct {
	ctx             context.Context
	conn            *websocket.Conn
	id              int
	requestMessages map[int]RequestMessage
	scripts         []ScriptParsedParams
	done            chan struct{}
	err             chan error
}

func NewDebugger(ctx context.Context, conn *websocket.Conn) Debugger {
	return Debugger{
		ctx:             ctx,
		conn:            conn,
		requestMessages: map[int]RequestMessage{},
		scripts:         make([]ScriptParsedParams, 0),
		done:            make(chan struct{}, 1),
		err:             make(chan error, 1),
	}
}

func (d *Debugger) Start() {
	d.SendMessage(RequestMessage{Method: "Debugger.enable"})
	d.SendMessage(RequestMessage{Method: "Runtime.enable"})
	d.SendMessage(RequestMessage{Method: "Runtime.runIfWaitingForDebugger"})
	go d.WaitForMessages()
	go d.WaitForCommands()
}

func (d *Debugger) SendMessage(msg RequestMessage) {
	d.id++
	msg.Id = d.id
	d.requestMessages[msg.Id] = msg
	b, err := json.Marshal(msg)
	if err != nil {
		d.err <- err
		return
	}
	d.conn.Write(d.ctx, websocket.MessageText, b)
}

func (d *Debugger) WaitForMessages() {
	for {
		_, r, err := d.conn.Reader(d.ctx)
		if err != nil {
			d.err <- err
			return
		}
		b, err := ioutil.ReadAll(r)
		if err != nil {
			d.err <- err
			return
		}
		var msg ResponseMessage
		err = json.Unmarshal(b, &msg)
		if err != nil {
			d.err <- err
			return
		}
		if msg.Id != 0 {
      req, ok := d.requestMessages[msg.Id]
      if ok {
        if req.Method == "Debugger.getScriptSource" {
        }
      }
		}
		if msg.Method == "Debugger.scriptParsed" {
			var params ScriptParsedParams
			err = json.Unmarshal(msg.Params, &params)
			if err != nil {
				d.err <- err
				return
			}
			d.scripts = append(d.scripts, params)
			// clearCurrentLine()
			// fmt.Println(params.Url, params.ScriptId)
			// printPrompt()
		}
	}
}

func (d *Debugger) WaitForCommands() {
	for {
		printPrompt()
		reader := bufio.NewReader(os.Stdin)
		str, err := reader.ReadString('\n')
		if err != nil {
			d.err <- err
			return
		}
		fields := strings.Fields(str)
		input := fields[0]
		if input == "exit" {
			break
		}
		switch input {
		case "sources":
			d.ListSourceFiles(fields[1:])
		}
	}
	d.done <- struct{}{}
}

func (d *Debugger) ListSourceFiles(args []string) {
	all := false
	for _, arg := range args {
		if arg == "--all" {
			all = true
		}
	}
	for _, script := range d.scripts {
		if all || !strings.HasPrefix(script.Url, "node:internal") {
			fmt.Printf(
				"%s%6s: %s%s\n",
				colorNumber,
				script.ScriptId,
				colorNone,
				script.Url,
			)
		}
	}
}

func (d *Debugger) ListSourceCode(args []string) {
}

func clearCurrentLine() {
	fmt.Print("\n\033[1A\033[K")
}

func printPrompt() {
	fmt.Printf("%s(ndb)%s ", colorPrompt, colorNone)
}
