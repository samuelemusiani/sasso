build: build-front build-server

build-front:
  echo "Building front-end..."
  cd frontend && npm install && npm run build

build-server:
  echo "Building server..."
  go build -o sasso ./server
