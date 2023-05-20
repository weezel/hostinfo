package geoip

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"fmt"
	"io/ioutil"
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
	//go:embed assets/GeoLite2-City.zip
	cityDataZip []byte
	//go:embed assets/GeoLite2-Country.zip
	countryDataZip []byte
	//go:embed assets/GeoLite2-ASN.zip
	asnDataZip []byte
)

func extractZip(data []byte, filename string) ([]byte, error) {
	reader := bytes.NewReader(data)
	z, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("extracting zip failed: %w", err)
	}

	fHandle, err := z.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}

	zipContent, err := ioutil.ReadAll(fHandle)
	if err != nil {
		return nil, fmt.Errorf("couldn't read zip contents of file %s: %w", filename, err)
	}

	return zipContent, nil
}

func New() *GeoInfo {
	cityData, err := extractZip(cityDataZip, "GeoLite2-City.mmdb")
	if err != nil {
		panic(err)
	}
	dbCity, err := geoip2.FromBytes(cityData)
	if err != nil {
		panic(err)
	}

	countryData, err := extractZip(countryDataZip, "GeoLite2-Country.mmdb")
	if err != nil {
		panic(err)
	}
	dbCountry, err := geoip2.FromBytes(countryData)
	if err != nil {
		panic(err)
	}

	asnData, err := extractZip(asnDataZip, "GeoLite2-ASN.mmdb")
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
