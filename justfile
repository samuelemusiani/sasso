build: build-front build-server

build-front:
  cd frontend && npm install && npm run build

build-server:
  go build -o sasso ./server

check-format:
  test -z $(gofmt -l .)
  cd frontend && npx prettier --check ./src

frontend-lint:
  cd frontend && npm run lint
