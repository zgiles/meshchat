.PHONY: all meshchat cleanpublic cleanbindata makepublic copyfrontend makebindata frontend

VERSION := $(shell git describe --tags --always --dirty)
PKG := "github.com/zgiles/meshchat/cmd/meshchat"
LDFLAG := -ldflags "-w -extldflags -static -X main.version=${VERSION}"

default: meshchat

all: frontend meshchat

meshchat: makebindata
	GOOS=darwin go build -o meshchat-darwin ${LDFLAG} ${PKG}
	GOOS=linux GOARCH=amd64 go build -o meshchat-amd64 ${LDFLAG} ${PKG}
	GOOS=linux GOARCH=arm GOARM=5 go build -o meshchat-arm5 ${LDFLAG} ${PKG}

frontend:
	cd frontend; npm run build

cleanbindata:
	rm -f cmd/meshchat/bindata.go

makebindata: cleanbindata copyfrontend
	cd frontend; go-bindata-assetfs build/...       
	cp frontend/bindata.go cmd/meshchat/

