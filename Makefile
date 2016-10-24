GO15VENDOREXPERIMENT=1
export GO15VENDOREXPERIMENT

build-all: .build/msgpc .build/msgpd .build/msgpdevd .build/msgdbd 

install-deps:
	glide install

update-deps:
	glide up

gofmt-all:
	find . -iname '*.go' -and -not -ipath './vendor/*' |\
		xargs dirname |\
		sort |\
		uniq |\
		xargs gofmt -w -e -s

golint-all:
	find . -iname '*.go' -and -not -ipath './vendor/*' |\
		xargs dirname |\
		sort |\
		uniq |\
		while read package;\
		do\
			golint $$package;\
		done;

.build/msgpc:
	go build ./cmd/msgpc

.build/msgpd:
	go build ./cmd/msgpd

.build/msgpdevd:
	go build ./cmd/msgpdevd

.build/msgdbd:
	go build ./cmd/msgdbd

.build/msgampc:
	GOARCH=mips32 go build ./cmd/msgampc

