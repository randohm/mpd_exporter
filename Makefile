NAME=mpd_exporter
VERSION=0.0.1
OSARCH=darwin-amd64 linux-amd64 linux-arm64
TARGETS=${NAME}.darwin-amd64 ${NAME}.linux-amd64 ${NAME}.linux-arm64

${NAME}: main.go
	go build -o ${NAME}

all: ${NAME} ${TARGETS}

${NAME}.darwin-amd64: main.go
	GOOS=darwin GOARCH=amd64 go build -o ${NAME}-${VERSION}.darwin-amd64

${NAME}.linux-amd64: main.go
	GOOS=linux GOARCH=amd64 go build -o ${NAME}-${VERSION}.linux-amd64

${NAME}.linux-arm64: main.go
	GOOS=linux GOARCH=arm64 go build -o ${NAME}-${VERSION}.linux-arm64

clean:
	rm -f ${NAME} ${TARGETS}
