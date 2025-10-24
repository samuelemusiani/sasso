build: build-front copy-front build-server build-router build-vpn

build-front:
  cd frontend && npm install && npm run build

copy-front:
  rm -rf ./server/_front
  cp -r ./frontend/dist ./server/_front

build-server:
  go build -o server-bin ./server

build-router:
  go build -o router-bin ./router

build-vpn:
  go build -o vpn-bin ./vpn

check-format:
  test -z $(gofmt -l .)
  cd frontend && npx prettier --check ./src

frontend-lint:
  cd frontend && npx eslint .

test:
  cd frontend && npm test
