# Hostinfo service

## Description

Similar to What is my IP and other services around there.
In short, returns information regarding to requesting IP address.
This application was created because many of the available services block Amazon IP ranges.

Following details are returned:

```json
{
  "addr": "127.0.0.1",
  "hostnames": "localhost",
  "src_port": "52952",
  "user_agent": "curl/8.0.1",
  "geo_location": {
    "asn": "0 ()"
  }
}
```

By default this will be build into a single binary which contains all the needed databases.
GeoIP databases aren't provided in this package to avoid licensing issues, just in to be sure.
Those can be found from the Internet with the names described in `Dependencies` section.

## Dependencies

* Go >1.18
* github.com/oschwald/geoip2-golang
* GeoIP databases (search from the Internet):
  * GeoLite2-City.mmdb
  * GeoLite2-Country.mmdb
  * GeoLite2-ASN.mmdb

## Prerequisites

Once databases are stored under [geoip/assets/](geoip/assets/) directory with the above mentioned names,
use `zip` to compress the files, like this:

```bash
cd geoip/assets
zip -9 GeoLite2-ASN.zip GeoLite2-ASN.mmdb
zip -9 GeoLite2-City.zip GeoLite2-City.mmdb
zip -9 GeoLite2-Country.zip GeoLite2-Country.mmdb
```

This almost divides the binary's final size by two.

Compiling the binary:

```bash
make build
```

## Usage

Example usage scenarios:

```bash
# Run on default port 8080
./dist/hostinfo
# Run on port 1234
./dist/hostinfo -p 1234
# Run on endpoint /foo
./dist/hostinfo -e foo
# Show version and build time
./dist/hostinfo -v
```
