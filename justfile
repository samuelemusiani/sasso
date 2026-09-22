ldflags := '-X main.version=`git describe --tags --always --dirty --abbrev=100` -X main.branch=`git rev-parse --abbrev-ref HEAD`'

build: build-front copy-front build-server build-router build-vpn

build-server-front: build-front copy-front build-server

build-front:
  cd frontend && npm install && npm run build

copy-front:
  rm -rf ./server/_front
  cp -r ./frontend/dist ./server/_front

build-server:
  go build -ldflags "{{ldflags}}" -o server-bin ./server

build-router:
  go build -ldflags "{{ldflags}}" -o router-bin ./router

build-vpn:
  go build -ldflags "{{ldflags}}" -o vpn-bin ./vpn

check-format:
  test -z $(gofmt -l .)
  cd frontend && npx prettier --check ./src

frontend-lint:
  cd frontend && npx eslint .

test:
  cd frontend && npm test
