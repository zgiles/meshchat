package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/memberlist"
	"github.com/nats-io/gnatsd/server"
	"github.com/nats-io/go-nats"
)

var writeWait = 3 * time.Second
var pongWait = 3 * time.Second
var pingPeriod = 2 * time.Second

type peerlist []string

func (i *peerlist) String() string {
	r := ""
	for _, x := range *i {
		r = r + " " + x
	}
	return r
}

func (i *peerlist) Set(v string) error {
	*i = append(*i, v)
	return nil
}

type ChatServer struct {
	clients       map[*websocket.Conn]bool
	broadcast     chan Message
	upgrader      websocket.Upgrader
	natsconnected bool
	nc            *nats.Conn
	ns            *server.Server
	writemu       sync.Mutex
}

type Message struct {
	Name     string `json:"name"`
	Message  string `json:"message"`
	Pong     bool   `json:"pong"`
	Presence bool   `json:"presence"`
	Id       string `json:"id"`
}

func (cs *ChatServer) handleChat(w http.ResponseWriter, r *http.Request) {
	log.Println("New Connection")
	ws, err := cs.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()
	cs.clients[ws] = true

	remoteid := ws.RemoteAddr().String()
	remotename := "User" + remoteid

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ticker.C:
				// log.Println("Sending a ping to %s\n", remoteid)
				ws.SetWriteDeadline(time.Now().Add(writeWait))
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
			cs.nc.Publish("meshchat.broadcast", b)
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
			delete(cs.clients, ws)
			break
		}
		//	if msg.Presence {
		remoteid = msg.Id
		remotename = msg.Name
		//	}
		if cs.natsconnected {
			b, berr := json.Marshal(msg)
			if berr != nil {
				log.Println(berr)
			} else {
				log.Printf("Sending NATs Message: %s\n", msg.Message)
				cs.nc.Publish("meshchat.broadcast", b)
			}
		}

		//cs.broadcast <- msg
	}
}

func (cs *ChatServer) sendtolocal(msg Message) {
	cs.writemu.Lock()
	defer cs.writemu.Unlock()
	for client := range cs.clients {
		client.SetWriteDeadline(time.Now().Add(writeWait))
		err := client.WriteJSON(msg)
		if err != nil {
			log.Println("Error writing to client: ", err)
			client.Close()
			delete(cs.clients, client)
		}

	}

}

func (cs *ChatServer) handleNatsMsg(m *nats.Msg) {
	var msg Message
	err := json.Unmarshal(m.Data, &msg)
	if err != nil {
		log.Println(err)
	} else {
		// log.Printf("New NATs Message: %s - %s", msg.Name, msg.Message)
		cs.sendtolocal(msg)
	}
}

func RunServer(opts server.Options) *server.Server {
	s, err := server.NewServer(&opts)
	if err != nil || s == nil {
		log.Printf("Couldnt start internal NATS Server: %v", err)
	}
	s.ConfigureLogger()
	go s.Start()
	return s
}

func main() {
	// flags
	httpport := flag.Int("httpport", 8080, "Port for HTTP/WS")
	natsport := flag.Int("natsport", 4222, "Port for NATS messagebus internal server")
	natsroutingport := flag.Int("natroutingport", 4001, "Port for NATS messagebus routing")
	clusterport := flag.Int("clusterport", 4444, "Cluster Port")
	debug := flag.Bool("debug", false, "Enable Debugging")
	var peers peerlist
	flag.Var(&peers, "peers", "peer list")
	flag.Parse()

	log.Println(*httpport)
	log.Println(*natsport)
	log.Println(*natsroutingport)
	log.Println(*clusterport)
	log.Println(peers)

	// chat server internal stuff
	cs := &ChatServer{}
	cs.clients = make(map[*websocket.Conn]bool)
	cs.broadcast = make(chan Message)
	cs.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	// cluster list
	clusteropts := memberlist.DefaultWANConfig()
	clusteropts.BindPort = *clusterport
	clusteropts.AdvertisePort = *clusterport
	list, err := memberlist.Create(clusteropts)
	if err != nil {
		log.Println("Failed to start memberlist.. oh well", err.Error())
	}
	_, err = list.Join([]string{})
	if err != nil {
		log.Println("Failed to join members", err.Error())
	}

	// start Nats
	opts := server.Options{
		Host:           "127.0.0.1",
		Port:           *natsport,
		NoLog:          !*debug,
		NoSigs:         true,
		MaxControlLine: 2048,
		Trace:          false,
		Debug:          *debug,
		Cluster: server.ClusterOpts{
			Host: "0.0.0.0",
			Port: *natsroutingport,
		},
	}
	if len(peers) > 0 {
		var routes []*url.URL
		for _, i := range peers {
			u, _ := url.Parse("nats-route://" + i)
			routes = append(routes, u)
		}
		opts.Routes = routes
	}
	cs.ns = RunServer(opts)

	// connect to nats
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Println("Couldn't connect to NATs, oh well")
	}
	nc.Subscribe("meshchat.broadcast", cs.handleNatsMsg)
	cs.nc = nc
	cs.natsconnected = true

	// publisher
	// go cs.messagepump()

	// server
	// fs := http.FileServer(http.Dir("public/"))
	fs := http.FileServer(assetFS())
	http.Handle("/", fs)
	http.HandleFunc("/ws", cs.handleChat)
	log.Printf("Starting http on %d", *httpport)
	err = http.ListenAndServe(":"+strconv.Itoa(*httpport), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}
