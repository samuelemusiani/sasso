# Sasso

## Develop

Sasso needs a PostgreSQL database. Use the `docker-compose-test.yml` file to start a test database.
```bash
docker compose -f docker-compose-test.yml up -d
```

To run the server with the default config:
```bash
go run ./... ./server/config/config.toml  
```

To run the frontend you must enter the `frontend` directory and:
```bash
npm run dev
```
