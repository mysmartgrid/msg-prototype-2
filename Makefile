build-all: .build/msgpc .build/msgpd .build/msgpdevd .build/msgdbd

godep-all:
	godep save . ./cmd/msgpd ./cmd/msgpdevd ./cmd/msgpc ./cmd/msgdbd

gofmt-all:
	find . -iname '*.go' -and -not -ipath './Godeps/*' |\
		xargs dirname |\
		sort |\
		uniq |\
		xargs godep go fmt

.build/msgpc:
	godep go build ./cmd/msgpc

.build/msgpd:
	godep go build ./cmd/msgpd

.build/msgpdevd:
	godep go build ./cmd/msgpdevd

.build/msgdbd:
	godep go build ./cmd/msgdbd
