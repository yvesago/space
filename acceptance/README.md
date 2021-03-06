# Space / Acceptance tests

> A user management microservice; OAuth 2 provider

## Setup

```sh
$ brew install chromedriver
$ go get github.com/sclevine/agouti
$ go get github.com/onsi/ginkgo/ginkgo
$ go get github.com/onsi/gomega
$ go get github.com/sclevine/agouti/matchers
$ go get github.com/manveru/faker
```

## Testing

```sh
$ ENV=testing go test ./...
```

## Generate new test case

```sh
$ ginkgo generate --agouti {description-file}
```

## Limitations

Currently, there is no dependency management

## License

[MIT License](http://earaujoassis.mit-license.org/) &copy; Ewerton Assis
