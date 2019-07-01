.PHONY: all meshchat cleanpublic cleanbindata makepublic copyfrontend makebindata frontend

default: meshchat

all: frontend meshchat

meshchat: makebindata
	GOOS=linux GOARCH=amd64 go build -o meshchat-amd64
	GOOS=linux GOARCH=arm GOARM=5 go build -o meshchat-arm5

frontend:
	cd frontend; npm run build

cleanpublic:
	rm -Rf public

cleanbindata:
	rm -f bindata.go

makepublic: cleanpublic
	mkdir public

copyfrontend: makepublic
	cp -r frontend/build/* public/

makebindata: cleanbindata copyfrontend
	go-bindata-assetfs public/...

