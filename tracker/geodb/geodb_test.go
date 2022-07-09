package geodb

import (
	"github.com/pirsch-analytics/pirsch/v4"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestGet(t *testing.T) {
	// this test is disabled if the license key is empty
	licenseKey := os.Getenv("GEOLITE2_LICENSE_KEY")

	if licenseKey == "" {
		return
	}

	assert.NoError(t, Get("geodb", licenseKey))
	_, err := os.Stat(pirsch.GeoLite2Filename)
	assert.NoError(t, err)
	_, err = os.Stat(geoLite2TarGzFilename)
	assert.True(t, os.IsNotExist(err))
}

func TestGeoDB_CountryCode(t *testing.T) {
	db, err := NewGeoDB(Config{
		File: filepath.Join("GeoIP2-City-Test.mmdb"),
	})
	assert.NoError(t, err)
	countryCode, city := db.CountryCodeAndCity("81.2.69.142")
	assert.Equal(t, "gb", countryCode)
	assert.Equal(t, "London", city)
}
