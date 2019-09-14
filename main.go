package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/memberlist"
	"github.com/nats-io/gnatsd/server"
	"github.com/nats-io/go-nats"
	"gopkg.in/alecthomas/kingpin.v2"
)

type peerlist []string

type rootConfig struct {
	writeWait       int
	pongWait        int
	pingPeriod      int
	peers           peerlist
	httpport        int
	natsport        int
	natsroutingport int
	clusterport     int
	debug           bool
}

var version string
var config rootConfig

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
	config = rootConfig{}
	var peersraw []string
	app := kingpin.New("meshchat", "Mesh Chat")
	app.UsageTemplate(kingpin.CompactUsageTemplate)
	app.Flag("writeWait", "NATS write wait").Default("3").IntVar(&config.writeWait)
	app.Flag("pongWait", "NATS pong wait").Default("3").IntVar(&config.pongWait)
	app.Flag("pingPeriod", "NATS ping period").Default("2").IntVar(&config.pingPeriod)
	app.Flag("httpport", "HTTP Port").Default("8080").IntVar(&config.httpport)
	app.Flag("natsport", "NATS Port").Default("4222").IntVar(&config.natsport)
	app.Flag("natsroutingport", "NATS routing Port").Default("4001").IntVar(&config.natsroutingport)
	app.Flag("clusterport", "Cluster port").Default("4444").IntVar(&config.clusterport)
	app.Flag("debug", "Debug on").BoolVar(&config.debug)
	app.Flag("peers", "Initial Peers List").StringsVar(&peersraw)
	app.Version(version)
	kingpin.MustParse(app.Parse(os.Args[1:]))
	config.peers = peerlist(peersraw)

	if config.debug {
		log.Printf("%+v", config)
	}

	// chat server internal
	cs := &ChatServer{}
	cs.clients = make(map[*websocket.Conn]bool)
	cs.broadcast = make(chan Message)
	cs.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	// cluster list
	clusteropts := memberlist.DefaultWANConfig()
	clusteropts.BindPort = config.clusterport
	clusteropts.AdvertisePort = config.clusterport
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
		Port:           config.natsport,
		NoLog:          !config.debug,
		NoSigs:         true,
		MaxControlLine: 2048,
		Trace:          false,
		Debug:          config.debug,
		Cluster: server.ClusterOpts{
			Host: "0.0.0.0",
			Port: config.natsroutingport,
		},
	}
	if len(config.peers) > 0 {
		var routes []*url.URL
		for _, i := range config.peers {
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
	log.Printf("Starting http on %d", config.httpport)
	err = http.ListenAndServe(":"+strconv.Itoa(config.httpport), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}
