GO15VENDOREXPERIMENT=1
export GO15VENDOREXPERIMENT

build-all: .build/msgpc .build/msgpd .build/msgpdevd .build/msgdbd

install-deps:
	glide install

update-deps:
	glide up

gofmt-all:
	find . -iname '*.go' -and -not -ipath './Godeps/*' |\
		xargs dirname |\
		sort |\
		uniq |\
		xargs godep go fmt

.build/msgpc:
	go build ./cmd/msgpc

.build/msgpd:
	go build ./cmd/msgpd

.build/msgpdevd:
	go build ./cmd/msgpdevd

.build/msgdbd:
	go build ./cmd/msgdbd
