/*
 Copyright 2020 Padduck, LLC
  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at
  	http://www.apache.org/licenses/LICENSE-2.0
  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
*/

package daemon

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/pufferpanel/pufferpanel/v3"
	"github.com/pufferpanel/pufferpanel/v3/config"
	"github.com/pufferpanel/pufferpanel/v3/logging"
	"github.com/pufferpanel/pufferpanel/v3/messages"
	"github.com/pufferpanel/pufferpanel/v3/servers"
	"io"
	path2 "path"
	"reflect"
	"runtime/debug"
	"strings"
)

func listenOnSocket(conn *pufferpanel.Socket, server *servers.Server, scopes []*pufferpanel.Scope) {
	defer func() {
		if err := recover(); err != nil {
			logging.Error.Printf("Error with websocket connection for server %s: %s\n%s", server.Id(), err, debug.Stack())
		}
	}()

	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			logging.Error.Printf("error on reading from websocket: %s", err)
			return
		}
		if msgType != websocket.TextMessage {
			continue
		}
		mapping := make(map[string]interface{})

		err = json.Unmarshal(data, &mapping)
		if err != nil {
			logging.Error.Printf("error on decoding websocket message: %s", err)
			continue
		}

		messageType := mapping["type"]
		if message, ok := messageType.(string); ok {
			switch strings.ToLower(message) {
			case "replay":
				{
					if pufferpanel.ContainsScope(scopes, pufferpanel.ScopeServerLogs) {
						console, _ := server.GetEnvironment().GetConsole()
						_ = conn.WriteMessage(messages.Console{Logs: console})
					}
				}
			case "stat":
				{
					if pufferpanel.ContainsScope(scopes, pufferpanel.ScopeServerStat) {
						results, err := server.GetEnvironment().GetStats()
						msg := messages.Stat{}
						if err != nil {
							msg.Cpu = 0
							msg.Memory = 0
						} else {
							msg.Cpu = results.Cpu
							msg.Memory = results.Memory
						}
						_ = conn.WriteMessage(msg)
					}
				}
			case "status":
				{
					if pufferpanel.ContainsScope(scopes, pufferpanel.ScopeServerStatus) {
						running, err := server.IsRunning()
						if err != nil {
							running = false
						}
						msg := messages.Status{Running: running}
						_ = conn.WriteMessage(msg)
					}
				}
			case "start":
				{
					if pufferpanel.ContainsScope(scopes, pufferpanel.ScopeServerStart) {
						_ = server.Start()
					}
					break
				}
			case "stop":
				{
					if pufferpanel.ContainsScope(scopes, pufferpanel.ScopeServerStop) {
						_ = server.Stop()
					}
				}
			case "install":
				{
					if pufferpanel.ContainsScope(scopes, pufferpanel.ScopeServerInstall) {
						_ = server.Install()
					}
				}
			case "kill":
				{
					if pufferpanel.ContainsScope(scopes, pufferpanel.ScopeServerKill) {
						_ = server.Kill()
					}
				}
			case "reload":
				{
					if pufferpanel.ContainsScope(scopes, pufferpanel.ScopeServerReload) {
						_ = servers.Reload(server.Id())
					}
				}
			case "ping":
				{
					_ = conn.WriteMessage(messages.Pong{})
				}
			case "console":
				{
					if pufferpanel.ContainsScope(scopes, pufferpanel.ScopeServerSendCommand) {
						cmd, ok := mapping["command"].(string)
						if ok {
							if run, _ := server.IsRunning(); run {
								_ = server.GetEnvironment().ExecuteInMainProcess(cmd)
							}
						}
					}
				}
			case "file":
				{
					action, ok := mapping["action"].(string)
					if !ok {
						break
					}
					path, ok := mapping["path"].(string)
					if !ok {
						break
					}

					switch strings.ToLower(action) {
					case "get":
						{
							if !pufferpanel.ContainsScope(scopes, pufferpanel.ScopeServerFileGet) {
								break
							}

							editMode, ok := mapping["edit"].(bool)
							handleGetFile(conn, server, path, ok && editMode)
						}
					case "delete":
						{
							if !pufferpanel.ContainsScope(scopes, pufferpanel.ScopeServerFileEdit) {
								break
							}

							err := server.DeleteItem(path)
							if err != nil {
								_ = conn.WriteMessage(messages.FileList{Error: err.Error()})
							} else {
								//now get the root
								handleGetFile(conn, server, path2.Dir(path), false)
							}
						}
					case "create":
						{
							if !pufferpanel.ContainsScope(scopes, pufferpanel.ScopeServerFileEdit) {
								break
							}

							err := server.CreateFolder(path)

							if err != nil {
								_ = conn.WriteMessage(messages.FileList{Error: err.Error()})
							} else {
								handleGetFile(conn, server, path, false)
							}
						}
					}
				}
			default:
				_ = conn.WriteJSON(map[string]string{"error": "unknown command"})
			}
		} else {
			logging.Error.Printf("message type is not a string, but was %s", reflect.TypeOf(messageType))
		}
	}
}

func handleGetFile(conn *pufferpanel.Socket, server *servers.Server, path string, editMode bool) {
	data, err := server.GetItem(path)
	if err != nil {
		_ = conn.WriteMessage(messages.FileList{Error: err.Error()})
		return
	}

	defer pufferpanel.Close(data.Contents)

	if data.FileList != nil {
		_ = conn.WriteMessage(messages.FileList{FileList: data.FileList, CurrentPath: path})
	} else if data.Contents != nil {
		//if the file is small enough, we'll send it over the websocket
		if editMode && data.ContentLength < int64(config.WebSocketFileLimit.Value()) {
			var buf bytes.Buffer
			_, _ = io.Copy(&buf, data.Contents)
			_ = conn.WriteMessage(messages.FileList{Contents: buf.Bytes(), Filename: data.Name})
		} else {
			_ = conn.WriteMessage(messages.FileList{Url: path, Filename: data.Name})
		}
	}
}
