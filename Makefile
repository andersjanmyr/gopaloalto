
.PHONY:
build:
	go build

.PHONY:
run:
	go run main.go 0 data/haarcascade_frontalface_default.xml
