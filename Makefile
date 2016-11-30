
rome: main.go
	go build

push:
	rsync -avz -e ssh ./static/ root@pertini:/opt/http/rabexc.org/sub/rome/http/

server:
	python -m SimpleHTTPServer 8000


