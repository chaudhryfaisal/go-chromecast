NAME=go-chromecast
BIN_DIR=dist
GOOS=darwin
GOARCH=amd64
GOARM=
ARGS=ls

run:
	go run main.go $(ARGS)

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) \
	go build -o $(BIN_DIR)/$(NAME)-$(GOOS)-$(GOARCH)$(GOARM) main.go

build-all:
#	$(MAKE) build
	$(MAKE) build GOOS=linux GOARCH=arm GOARM=5
	$(MAKE) compress

push:
	scp ./dist/$(NAME)-linux-arm5 root@192.168.1.1:/jffs/chromecast/

compress:
	upx $(BIN_DIR)/* || true
