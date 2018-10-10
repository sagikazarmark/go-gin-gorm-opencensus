# Go OpenCensus example for Gin and Gorm

[![Build Status](https://img.shields.io/travis/com/sagikazarmark/go-gin-gorm-opencensus.svg?style=flat-square)](https://travis-ci.com/sagikazarmark/go-gin-gorm-opencensus)
[![Go Report Card](https://goreportcard.com/badge/github.com/sagikazarmark/go-gin-gorm-opencensus?style=flat-square)](https://goreportcard.com/report/github.com/sagikazarmark/go-gin-gorm-opencensus)
[![GolangCI](https://golangci.com/badges/github.com/sagikazarmark/go-gin-gorm-opencensus.svg)](https://golangci.com/r/github.com/sagikazarmark/go-gin-gorm-opencensus)

This repository serves as an example for configuring [OpenCensus](http://opencensus.io/) to
instrument applications written using [Gin](https://gin-gonic.github.io/gin/) framework
and [Gorm](http://gorm.io/) ORM.


## Requirements

- Go 1.11
- Docker (with Compose)
- Dep 0.5.0 (make installs it for you)


## Usage

1. Set up the project: `make up`
2. Run the application: `make run`

When you are done playing with the project you can easily destroy everything it created with `make down`.
(It removes everything except `.env` and `.env.test`)


## License

The MIT License (MIT). Please see [License File](LICENSE) for more information.
