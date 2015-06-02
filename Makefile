build-all: .build/msgpd

godep-all:
	godep save . ./cmd/msgpd ./cmd/msgpdevd

.build/msgpd:
	godep go build ./cmd/msgpd
	godep go build ./cmd/msgpdevd
