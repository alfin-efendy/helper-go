package restapi

import (
	"net"
	"net/url"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/truemail-rb/truemail-go"
)

var isUrl validator.Func = func(fl validator.FieldLevel) bool {
	data, ok := fl.Field().Interface().(string)
	// remove www. from the url and trim the space
	data = strings.TrimSpace(strings.ToLower(strings.Replace(data, "www.", "", 1)))

	if ok {
		// parse the url
		u, err := url.Parse(data)
		if err != nil || u.Hostname() == "" {
			return true
		}

		// check if the url is valid
		if _, err := net.LookupIP(u.Hostname()); err != nil {
			return true
		}

		return false

	}
	return true
}

var isActiveEmail validator.Func = func(fl validator.FieldLevel) bool {
	data, ok := fl.Field().Interface().(string)

	if ok {
		// try to create configuration for validation the email address
		config, err := truemail.NewConfiguration(
			truemail.ConfigurationAttr{
				VerifierEmail: "alfin1993@gmail.com",
			},
		)

		if err != nil {
			return true
		}

		// Validation email address via MX record
		if truemail.IsValid(data, config, "mx") {
			return false
		}

	}

	return true
}

func enum(fl validator.FieldLevel) bool {
	// Dapatkan nilai yang akan divalidasi
	value := fl.Field().String()
	if value == "" {
		return false
	}

	// Dapatkan parameter enum dari tag (employee|customer|vendor)
	enumValues := strings.Split(fl.Param(), "/")
	if len(enumValues) == 0 {
		return false
	}

	// Cek apakah nilai ada dalam daftar valid
	for _, v := range enumValues {
		if value == strings.TrimSpace(v) {
			return true
		}
	}

	return false
}
