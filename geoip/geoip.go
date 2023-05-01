package geoip

import (
	_ "embed"
	"fmt"
	"log"
	"net"

	"github.com/oschwald/geoip2-golang"
)

type GeoInfo struct {
	dbCity      *geoip2.Reader
	dbCountry   *geoip2.Reader
	dbASN       *geoip2.Reader
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	City        string `json:"city,omitempty"`
	ASN         string `json:"asn,omitempty"`
}

var (
	//go:embed assets/GeoLite2-City.mmdb
	cityData []byte
	//go:embed assets/GeoLite2-Country.mmdb
	countryData []byte
	//go:embed assets/GeoLite2-ASN.mmdb
	asnData []byte
)

func New() *GeoInfo {
	dbCity, err := geoip2.FromBytes(cityData)
	if err != nil {
		panic(err)
	}

	dbCountry, err := geoip2.FromBytes(countryData)
	if err != nil {
		panic(err)
	}

	dbASN, err := geoip2.FromBytes(asnData)
	if err != nil {
		panic(err)
	}

	return &GeoInfo{
		dbCity:    dbCity,
		dbCountry: dbCountry,
		dbASN:     dbASN,
	}
}

func (g *GeoInfo) Close() {
	if g.dbCity != nil {
		if err := g.dbCity.Close(); err != nil {
			log.Printf("Failed to close geo city db: %v", err)
		}
	}

	if g.dbCountry != nil {
		if err := g.dbCountry.Close(); err != nil {
			log.Printf("Failed to close geo country db: %v", err)
		}
	}

	if g.dbASN != nil {
		if err := g.dbASN.Close(); err != nil {
			log.Printf("Failed to close geo ASN db: %v", err)
		}
	}
}

// GetGeoData parses geodata and if some fields fails, it tries to fetch other information
func (g GeoInfo) GetGeoData(ip string) <-chan GeoInfo {
	ret := make(chan GeoInfo, 1)

	go func() {
		ipAddr := net.ParseIP(ip)

		geoInfo, err := g.dbCity.City(ipAddr)
		if err != nil {
			log.Printf("failed to parse city: %v", err)
		}

		asn, err := g.dbASN.ASN(ipAddr)
		if err != nil {
			log.Printf("failed to parse ASN: %v", err)
		}

		ret <- GeoInfo{
			Country:     geoInfo.Country.Names["en"],
			CountryCode: geoInfo.Country.IsoCode,
			City:        geoInfo.City.Names["en"],
			ASN: fmt.Sprintf("%d (%s)",
				asn.AutonomousSystemNumber,
				asn.AutonomousSystemOrganization),
		}
	}()

	return ret
}
