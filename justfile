build: build-front copy-front build-server

build-front:
  cd frontend && npm install && npm run build

copy-front:
  rm -rf ./server/_front
  cp -r ./frontend/dist ./server/_front

build-server:
  go build -o server ./server

check-format:
  test -z $(gofmt -l .)
  cd frontend && npx prettier --check ./src

frontend-lint:
  cd frontend && npx eslint .
