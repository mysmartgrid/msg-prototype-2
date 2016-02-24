# msg-prototype-2

## Requirements
- Go >=1.5
- Glide `go get github.com/Masterminds/glide`

  `go get github.com/Masterminds/glide`
  `export PATH=$PATH:${GOPATH}/bin`
  `git clone https://github.com/mysmartgrid/msg-prototype-2.git ${GOPATH}/src/github.com/mysmartgrid/msg-prototype-2`


- postgres >=9.5

  `sudo -s`
  `echo deb http://apt.postgresql.org/pub/repos/apt/ trusty-pgdg main > /etc/apt/sources.list.d/pgdg.list`
  `wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add -`
  `sudo apt-get update`
  `sudo apt-get install postgresql-9.5`


## Build
- Run `make install-deps`
- Then run `make build-all`

  `(cd ${GOPATH}/src/github.com/mysmartgrid/msg-prototype-2/;make install-deps)`
  `(cd ${GOPATH}/src/github.com/mysmartgrid/msg-prototype-2/;make)`

## Startup database

`sudo -u postgres -H createuser --createdb --pwprompt msgdb`
`sudo -u postgres -H createdb --owner=msgdb msgdb`
`psql -U msgdb -d msgdb -W -h localhost < src/github.com/mysmartgrid/msg-prototype-2/db/initdb.sql`


## Usage
- See https://github.com/mysmartgrid/msg-prototype-2/wiki
