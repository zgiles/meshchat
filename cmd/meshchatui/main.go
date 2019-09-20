package main

import (
	"fmt"
	"fyne.io/fyne"
	fyneapp "fyne.io/fyne/app"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"

	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats-server/server"
	nats "github.com/nats-io/nats.go"
	"github.com/zgiles/meshchat/chatserver"
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

func (config *rootConfig) startmeshchat() chan interface{} {
	// chat server internal
	cs := &chatserver.ChatServer{
		WriteWait:  config.writeWait,
		PongWait:   config.pongWait,
		PingPeriod: config.pingPeriod,
	}
	cs.Clients = make(map[*websocket.Conn]bool)
	cs.Broadcast = make(chan chatserver.Message)
	cs.Upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
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
	cs.Ns = RunServer(opts)

	// connect to nats
	nc, err := nats.Connect("nats://localhost:" + strconv.Itoa(config.natsport))
	if err != nil {
		log.Println("Couldn't connect to NATs, oh well, will keep trying")
	}
	nc.Subscribe("meshchat.broadcast", cs.HandleNatsMsg)
	cs.Nc = nc
	cs.Natsconnected = true

	fs := http.FileServer(assetFS())
	mux := http.NewServeMux()
	mux.Handle("/", fs)
	mux.HandleFunc("/ws", cs.HandleChat)
	log.Printf("Starting http on %d", config.httpport)
	server := &http.Server{Addr: ":" + strconv.Itoa(config.httpport), Handler: mux}
	cancelchan := make(chan interface{})
	go func() {
		err = server.ListenAndServe()
		if err != nil {
			log.Println("ListenAndServe: ", err)
		}
		for c := range cs.Clients {
			c.Close()
		}
		cs.Nc.Drain()
		cs.Nc.Close()
		cs.Ns.Shutdown()
	}()
	go func() {
		<-cancelchan
		log.Println("Cancelchan hit")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := server.Shutdown(ctx); err != nil {
			log.Println(err)
		}
		cancel()
	}()
	return cancelchan
}

func main() {
	config = rootConfig{}
	// var peersraw []string
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
	// app.Flag("peers", "Initial Peers List").StringsVar(&peersraw)
	app.Version(version)
	kingpin.MustParse(app.Parse(os.Args[1:]))
	// config.peers = peerlist(peersraw)

	if config.debug {
		log.Printf("%+v", config)
	}

	var cancel chan interface{}

	// GUI
	guiapp := fyneapp.New()
	w := guiapp.NewWindow("Meshchat")
	w.Resize(fyne.Size{500, 300})

	peer1 := widget.NewEntry()
	peer1.SetPlaceHolder("Peer 1 (optional)")
	peer2 := widget.NewEntry()
	peer2.SetPlaceHolder("Peer 2 (optional)")
	peer3 := widget.NewEntry()
	peer3.SetPlaceHolder("Peer 3 (optional)")
	form := &widget.Form{}
	form.Append("Peer 1", peer1)
	form.Append("Peer 2", peer2)
	form.Append("Peer 3", peer3)

	runninglabel := widget.NewLabel("Not Running")

	startbutton := widget.NewButton("Start Meshchat", func() {
		peer1.ReadOnly = true
		peer1.Hidden = true
		peer2.ReadOnly = true
		peer2.Hidden = true
		peer3.ReadOnly = true
		peer3.Hidden = true
		for _, str := range []string{peer1.Text, peer2.Text, peer3.Text} {
			if str != "" {
				config.peers = append(config.peers, str)
			}
		}
		runninglabel.SetText("Running")
		fmt.Println("Entry", config.peers)
		cancel = config.startmeshchat()
	})
	startbutton.Style = widget.PrimaryButton

	stopbutton := widget.NewButton("Stop Meshchat", func() {
		close(cancel)
		runninglabel.SetText("Not Running")
		peer1.ReadOnly = false
		peer1.Hidden = false
		peer2.ReadOnly = false
		peer2.Hidden = false
		peer3.ReadOnly = false
		peer3.Hidden = false
	})
	// stopbutton.Disable()

	quitbutton := widget.NewButton("Quit", func() {
		guiapp.Quit()
	})

	buttonrow := fyne.NewContainerWithLayout(layout.NewGridLayout(3),
		startbutton,
		stopbutton,
		quitbutton,
	)

	w.SetContent(widget.NewVBox(
		widget.NewLabel("Meshchat..."),
		widget.NewVBox(form),
		buttonrow,
		runninglabel,
	))

	w.ShowAndRun()

	// end GUI

}
