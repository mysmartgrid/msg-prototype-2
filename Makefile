build-all: .build/msgpd

godep-all:
	godep save . ./cmd/msgpd

.build/msgpd:
	godep go build ./cmd/msgpd
