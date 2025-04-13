# Chirpy Web Server Guided Project from **Boot.dev**
## It is a simple microblogging platform API built with Go. It supports user registration, posting short messages("Chirps") and webhooks integration

### Used technology:
* Go
* PostgreSQL
* Goose
* JWT
* SQLC
* github.com/google/uuid
* GoDotEnv

### Instalation guide
* git clone https://github.com/FallenL3vi/WebServer
* create file .env
* set up enviormental variables:
    DB_URL = "postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable"
    PLATFORM = "dev"
    SECRET_JWT = "JWT_HERE"
    POLKA_KEY = "POLKA_KEY"
* sudo service postgresql start
* go to sql/schema
* goose postgres "postgres://postgres:postgres@localhost:5432/chirpy" up
* go back to root of project
* go build -o out && ./out


### List of endpoints

* POST /admin/reset

* POST /api/users

* POST /api/chirps

* GET /api/chirps/{chirpID}

* GET /api/chirps

* POST /api/login

* POST /api/refresh

* POST /api/revoke

* PUT /api/users

* DELETE /api/chirps/{chirpID}

* POST /api/polka/webhooks