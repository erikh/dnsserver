# dnsserver - a simple DNS service toolkit

_(This repository is adapted from docker/dnsserver by the original author)_

This provides a very basic API for programming a DNS service that serves over
UDP only. A records and simple SRV records are currently supported, although
this may change in the future.

## Stability

This should be considered moderately stable software; it has been used in
several production environments in slightly modified form.

## Documentation

You can find comprehensive documentation in the source comments or at the URL below:

[http://godoc.org/github.com/erikh/dnsserver](http://godoc.org/github.com/erikh/dnsserver)

## Dependencies

- [github.com/miekg/dns](https://github.com/miekg/dns)

## License

We use the Apache 2.0 License as with all public Docker projects. You can read
this [here](https://github.com/docker/dnsserver/blob/master/LICENSE).

## Contribution Guidelines

- Fork the project and send a pull request
- Please do not modify any license or version information.
- Thanks!

## Maintainer

Erik Hollensbe <github@hollensbe.org>
