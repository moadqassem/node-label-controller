IMAGE_REPO=moadqassem/node-label-controller
VERSION=latest

build:
	CGO_ENABLED=0 go build -ldflags="-w -s" -v

run: build
	./node-label-controller --config config/config.json
image:
	docker build --rm=true -t $(IMAGE_REPO):$(VERSION) .

docker-push: image
	docker push $(IMAGE_REPO):$(VERSION)

vendor:
	go mod vendor

install:
	./install.sh
