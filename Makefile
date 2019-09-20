.PHONY: all meshchat cleanpublic cleanbindata makepublic copyfrontend makebindata frontend

VERSION := $(shell git describe --tags --always --dirty)
LDFLAG := -ldflags "-w -extldflags -static -X main.version=${VERSION}"
PKG := "github.com/zgiles/meshchat/cmd/meshchat"
PKGUI := "github.com/zgiles/meshchat/cmd/meshchatui"

default: meshchat meshchatui

all: frontend meshchat

meshchat: makebindata
	GOOS=darwin go build -o meshchat-darwin ${LDFLAG} ${PKG}
	GOOS=linux GOARCH=amd64 go build -o meshchat-amd64 ${LDFLAG} ${PKG}
	GOOS=linux GOARCH=arm GOARM=5 go build -o meshchat-arm5 ${LDFLAG} ${PKG}

meshchatui: makebindata
	go build -o meshchatui -ldflags "-X main.version=${VERSION}" ${PKGUI}

frontend:
	cd frontend; npm run build

cleanbindata:
	rm -f cmd/meshchat/bindata.go

makebindata: cleanbindata copyfrontend
	cd frontend; go-bindata-assetfs build/...
	cp frontend/bindata.go cmd/meshchat/
	cp frontend/bindata.go cmd/meshchatui/

