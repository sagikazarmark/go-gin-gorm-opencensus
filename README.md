# Go OpenCensus example for Gin and Gorm

[![Build Status](https://img.shields.io/travis/com/sagikazarmark/go-gin-gorm-opencensus.svg?style=flat-square)](https://travis-ci.com/sagikazarmark/go-gin-gorm-opencensus)
[![Go Report Card](https://goreportcard.com/badge/github.com/hashicorp/go-gin-gorm-opencensus?style=flat-square)](https://goreportcard.com/report/github.com/hashicorp/go-gin-gorm-opencensus)
[![GolangCI](https://golangci.com/badges/github.com/hashicorp/go-gin-gorm-opencensus.svg)](https://golangci.com/r/github.com/hashicorp/go-gin-gorm-opencensus)

This repository serves as an example for configuring [OpenCensus](http://opencensus.io/) to
instrument applications written using [Gin](https://gin-gonic.github.io/gin/) framework
and [Gorm](http://gorm.io/) ORM.


## Requirements

- Go 1.11
- Docker (with Compose)
- Dep 0.5.0 (make installs it for you)
- cURL or Postman for calling the API
- the following ports free: 3306, 6831, 8080, 9090, 14268, 16686 (alternative: edit `docker-compose.override.yml` manually)


## Usage

1. Set up the project: `make up`
2. Run the application: `make run`
3. Open `http://localhost:16686` in your browser (Jaeger UI)
4. Open `http://localhost:9090` in your browser (Prometheus UI)

When you are done playing with the project you can easily destroy everything it created with `make down`.
(It removes everything except `.env` and `.env.test`)


## Calling the API

The easiest way to run the example is using Postman:

[![Run in Postman](https://run.pstmn.io/button.svg)](https://app.getpostman.com/run-collection/b2c8fd4eee98b396b5d8)

Alternatively you can send simple HTTP requests with any tool you like.
For example using cURL:

```bash
$ curl http://localhost:8080/people -d '{"first_name": "John", "last_name": "Doe"}'
$ curl http://localhost:8080/hello/John
```


## License

The MIT License (MIT). Please see [License File](LICENSE) for more information.
