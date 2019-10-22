# ss-apiserver
This is a simple restful API server that translates requests to an upstream `ss-manager` instance. This program requires netcat (check with `which nc`). 

## Usage
Run `ss-manager` first:

```bash
ss-manager --manager-address 127.0.0.1:1080 [...]
```

Then run `ss-apiserver`:

```bash
ss-apiserver -hostname 127.0.0.1 -port 1080 -listen 8080
```

Now you can control `ss-manager` using restful API:

```bash
# returns "ok" if ss-apiserver is running
curl http://localhost:8080/healthcheck

# get all ports and traffic stats
curl http://localhost:8080/ports

# add a port. this creates a new ss-server instance.
curl -X POST http://localhost:8080/ports/12345 -d MyPassword

# you can't change the password once the port is created. to modify, remove that port and add again.
curl -X DELETE http://localhost:8080/ports/12345
```

You can substitute `localhost` above to your host's domain.


## Compile

You need a golang development environment (v1.12)

```bash
cd /go/src
export SS_API_BRANCH_NAME="v1.0"
SS_API_SRC_DIR="/go/src/github.com/imacks/ss-apiserver"
SS_API_GIT_DIR="https://github.com/imacks/ss-apiserver.git/"
mkdir -p "$SS_API_SRC_DIR"
cd "$SS_API_SRC_DIR"
git clone "$SS_API_GIT_DIR" --recursive --single-branch --branch "$SS_API_BRANCH_NAME" ./
git checkout "tags/${SS_API_BRANCH_NAME}"
export GIT_DESCRIBE=$(git describe --tags)
export LDFLAGS="-X main.VERSION=${GIT_DESCRIBE} -s -w"
export GCFLAGS=""
export CGO_ENABLED=0
export GOOS="linux"
export GOARCH="amd64"
export GO111MODULE=on
go mod download
go build -v -ldflags "$LDFLAGS" -gcflags "$GCFLAGS" -o /go/bin/ss-apiserver
```
