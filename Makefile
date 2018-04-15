docker-build:
	rm -rf deploy
	mkdir deploy

	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o deploy/coding-guru .
	curl -o deploy/ca-certificates.crt https://raw.githubusercontent.com/bagder/ca-bundle/master/ca-bundle.crt

	sudo docker build -t coding-guru .
	rm -rf deploy
