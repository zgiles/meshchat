package chatserver

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats-server/server"
	"github.com/nats-io/nats.go"
)

type ChatServer struct {
	WriteWait     int
	PongWait      int
	PingPeriod    int
	Clients       map[*websocket.Conn]bool
	Broadcast     chan Message
	Upgrader      websocket.Upgrader
	Natsconnected bool
	Nc            *nats.Conn
	Ns            *server.Server
	Writemu       sync.Mutex
}

type Message struct {
	Name     string `json:"name"`
	Message  string `json:"message"`
	Pong     bool   `json:"pong"`
	Presence bool   `json:"presence"`
	Id       string `json:"id"`
}

func (cs *ChatServer) HandleChat(w http.ResponseWriter, r *http.Request) {
	log.Println("New Connection")
	ws, err := cs.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()
	cs.Clients[ws] = true

	remoteid := ws.RemoteAddr().String()
	remotename := "User" + remoteid

	ticker := time.NewTicker(time.Duration(cs.PingPeriod) * time.Second)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ticker.C:
				// log.Println("Sending a ping to %s\n", remoteid)
				ws.SetWriteDeadline(time.Now().Add(time.Duration(cs.WriteWait) * time.Second))
				if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Println(err)
					return
				}
			}
		}
	}()
	ws.SetPongHandler(func(in string) error {
		// log.Printf("got a pong from %s\n", remoteid)
		msg := Message{
			Name: remotename,
			Id:   remoteid,
			Pong: true,
		}
		b, berr := json.Marshal(msg)
		if berr != nil {
			log.Println(berr)
		} else {
			// log.Printf("Sending NATs Pong: %s\n", msg.Name)
			cs.Nc.Publish("meshchat.broadcast", b)
		}

		return nil
	})

	log.Println("Waiting for messages")
	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		// log.Printf("New WS Message: %s - %s", msg.Name, msg.Message)
		if err != nil {
			log.Println("Error on ws: ", err)
			log.Println(msg)
			delete(cs.Clients, ws)
			break
		}
		//      if msg.Presence {
		remoteid = msg.Id
		remotename = msg.Name
		//      }
		if cs.Natsconnected {
			b, berr := json.Marshal(msg)
			if berr != nil {
				log.Println(berr)
			} else {
				log.Printf("Sending NATs Message: %s\n", msg.Message)
				cs.Nc.Publish("meshchat.broadcast", b)
			}
		}

		//cs.broadcast <- msg
	}
}

func (cs *ChatServer) Sendtolocal(msg Message) {
	cs.Writemu.Lock()
	defer cs.Writemu.Unlock()
	for client := range cs.Clients {
		client.SetWriteDeadline(time.Now().Add(time.Duration(cs.WriteWait) * time.Second))
		err := client.WriteJSON(msg)
		if err != nil {
			log.Println("Error writing to client: ", err)
			client.Close()
			delete(cs.Clients, client)
		}

	}

}

func (cs *ChatServer) HandleNatsMsg(m *nats.Msg) {
	var msg Message
	err := json.Unmarshal(m.Data, &msg)
	if err != nil {
		log.Println(err)
	} else {
		// log.Printf("New NATs Message: %s - %s", msg.Name, msg.Message)
		cs.Sendtolocal(msg)
	}
}
