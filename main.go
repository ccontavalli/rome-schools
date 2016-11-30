// This file was downloaded from:
//   https://github.com/googlemaps/google-maps-services-go/tree/master/examples/geocoding/cmdline
//
// And modified by ccontavalli@gmail.com to:
//   - parse .csv files extracted from public data provided from
//     http://urslazio.it.
//   - fetch all locations extracted by those .csv files.
//   - generate a .json file with a normalized list of schools extracted
//     from those files.
//
// The changes are ugly, main purpose of the work was to get clean and
// normalized data as quickly as possible.


// This is the original Copyright notice for the code:
//
// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main contains a simple command line tool for Geocoding API
// Documentation: https://developers.google.com/maps/documentation/geocoding/

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"encoding/json"
	"encoding/csv"
	"github.com/kr/pretty"
	"golang.org/x/net/context"
	"googlemaps.github.io/maps"
)

var (
	apiKey       = flag.String("key", "", "API Key for using Google Maps API.")
	clientID     = flag.String("client_id", "", "ClientID for Maps for Work API access.")
	signature    = flag.String("signature", "", "Signature for Maps for Work API access.")
	Address      = flag.String("Address", "", "The street Address that you want to geocode, in the format used by the national postal service of the country concerned.")
	components   = flag.String("components", "", "A component filter for which you wish to obtain a geocode.")
	bounds       = flag.String("bounds", "", "The bounding box of the viewport within which to bias geocode results more prominently.")
	language     = flag.String("language", "", "The language in which to return results.")
	region       = flag.String("region", "", "The region code, specified as a ccTLD two-character value.")
	latlng       = flag.String("latlng", "", "The textual latitude/longitude value for which you wish to obtain the closest, human-readable Address.")
	resultType   = flag.String("result_type", "", "One or more Address types, separated by a pipe (|).")
	locationType = flag.String("location_type", "", "One or more location types, separated by a pipe (|).")
)

func usageAndExit(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	fmt.Println("Flags:")
	flag.PrintDefaults()
	os.Exit(2)
}

func check(err error) {
	if err != nil {
		log.Fatalf("fatal error: %s", err)
	}
}

func Clean(record []string) []string {
	retval := make([]string, len(record), len(record))
	for i := 0; i < len(record); i++ {
		temp := record[i]
		temp = strings.Replace(temp, "\n", " ", -1)
		temp = strings.Replace(temp, "\r", " ", -1)
		temp = strings.TrimSpace(temp)

		retval[i] = temp
	}
	return retval
}

func Desired(record []string) bool {
	for i := 0; i < len(record); i++ {
		normalized := strings.ToLower(record[i])

		if strings.Contains(normalized, "infanzia") ||
			strings.Contains(normalized, "nido") {
			return true
		}
	}

	return false
}

type School struct {
	Origin *string `json:"origin"`

	Name    string `json:"name"`
	Contact string `json:"contact"`
	Address string `json:"address"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`

        Location *maps.GeocodingResult `json:"location"`

}

type Parser struct {
	Name  string
        Origin string
	parse func([]string) *School
}

func ParseNonParitarie(record []string) *School {
	record = Clean(record)
	if !Desired(record) {
		return nil
	}

	var Email, Phone string
        data := strings.Split(record[9], " ")
        for _, element := range(data) {
	  if strings.Contains(element, "@") {
		Email = Email + " " + element
	  } else {
		Phone = Phone + " " + element
	  }
        }

	// pretty.Println(record)
	return &School{
		Name:    record[5],
		Contact: record[6],
		Address: record[4] + ", " + record[1] + ", " + record[3],
		Email:   strings.TrimSpace(Email),
		Phone:   strings.TrimSpace(Phone),
	}
}

func ParseStraniere(record []string) *School {
	record = Clean(record)
	if !Desired(record) {
		return nil
	}

	// 12: LEGALE RAPPRESENTANTE
	// 13: DIRETTORE
	var Contact string
	if record[13] != "" {
		Contact = record[13]
	} else {
		Contact = record[12]
	}

	// 5: TEL SEDE LEGALE
	// 8: TEL SEDE OPERATIVA
	var Phone string
	if record[8] != "" {
		Phone = record[8]
	} else {
		Phone = record[5]
	}

	// 1: SEDE LEGALE
	// 6: SEDE OPERATIVA
	var Address string
	if record[6] != "" {
		Address = record[6] + ", " + record[7]
	} else {
		Address = record[1] + ", " + record[2] + ", " + record[3]
	}

	return &School{
		Name:    record[0],
		Contact: Contact,
		Email:   record[9],
		Phone:   Phone,
		Address: Address,
	}

}

func ParseParitarie(record []string) *School {
	record = Clean(record)
	if record[0] == "" || record[2] == "" {
		return nil
	}

	// pretty.Println(record)
	return &School{
		Name:    record[2],
		Address: record[4] + ", " + record[3] + ", " + record[5],
		Email:   record[6],
		Phone:   record[7],
	}
}

func Printer(record []string) *School {
	pretty.Println(record)
	return nil
}

func ReadSchools() []*School {
	var parsers = []Parser{
		{"ELENCONONPARITARIELAZIO2016_2017.csv", "non-paritarie", ParseNonParitarie},
		{"INFANZIA_Paritarie_2015_2016.csv", "paritarie", ParseParitarie},
		{"scuole_straniere_2016.csv", "straniere", ParseStraniere},
	}

	schools := make([]*School, 0)
	for index := range parsers {
		parser := &parsers[index]
		fd, err := os.Open("./data/" + parser.Name)
		if err != nil {
			log.Fatal(err)
		}

		cr := csv.NewReader(fd)
		cr.FieldsPerRecord = -1
		for {
			record, err := cr.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Println(err)
				continue
			}

			parsed := parser.parse(record)
			if parsed != nil {
				parsed.Origin = &parser.Origin
				schools = append(schools, parsed)
			}
		}
	}
	// pretty.Println(schools)
	return schools
}

func GeoCode(client *maps.Client, Address string) ([]maps.GeocodingResult, error) {
	r := &maps.GeocodingRequest{
		Address:  Address,
		Language: *language,
		Region:   *region,
	}

	parseComponents(*components, r)
	parseBounds(*bounds, r)
	parseLatLng(*latlng, r)
	parseResultType(*resultType, r)
	parseLocationType(*locationType, r)

	resp, err := client.Geocode(context.Background(), r)
	return resp, err
}

func main() {
	flag.Parse()

	var client *maps.Client
	var err error
	if *apiKey != "" {
		client, err = maps.NewClient(maps.WithAPIKey(*apiKey))
	} else if *clientID != "" || *signature != "" {
		client, err = maps.NewClient(maps.WithClientIDAndSignature(*clientID, *signature))
	} else {
		usageAndExit("Please specify an API Key, or Client ID and Signature.")
	}
	check(err)

	if *Address != "" {
		resp, err := GeoCode(client, *Address)
		check(err)

		for _, r := range resp {
			log.Printf("%s %0.3f %0.3f\n", r.FormattedAddress, r.Geometry.Location.Lat, r.Geometry.Location.Lng)
		}
		pretty.Println(resp)
	} else {
		schools := ReadSchools()
		for index, school := range schools {
			resp, err := GeoCode(client, school.Address)
			if err != nil {
				log.Printf("[%d] ERROR with %*v %v\n", index, school, err)
				continue
			}

			if len(resp) < 1 {
				log.Printf("[%d] ERROR with %*v - no result\n", index, school)
				continue
			}

                        school.Location = &resp[0]
                        log.Printf("[%d] DETECTED LOCATION FOR: %s", index, school.Name)
		}
                jschools, err := json.MarshalIndent(struct { Schools []*School `json:"schools"` } {schools}, "", "  ")
                if err != nil {
                  log.Printf("ERROR in json marshaling: %s\n", err)
                }
                os.Stdout.Write(jschools)
	}
}

func parseComponents(components string, r *maps.GeocodingRequest) {
	if components != "" {
		c := strings.Split(components, "|")
		for _, cf := range c {
			i := strings.Split(cf, ":")
			switch i[0] {
			case "route":
				r.Components[maps.ComponentRoute] = i[1]
			case "locality":
				r.Components[maps.ComponentLocality] = i[1]
			case "administrative_area":
				r.Components[maps.ComponentAdministrativeArea] = i[1]
			case "postal_code":
				r.Components[maps.ComponentPostalCode] = i[1]
			case "country":
				r.Components[maps.ComponentCountry] = i[1]
			}
		}
	}
}

func parseBounds(bounds string, r *maps.GeocodingRequest) {
	if bounds != "" {
		b := strings.Split(bounds, "|")
		sw := strings.Split(b[0], ",")
		ne := strings.Split(b[1], ",")

		swLat, err := strconv.ParseFloat(sw[0], 64)
		if err != nil {
			log.Fatalf("Couldn't parse bounds: %#v", err)
		}
		swLng, err := strconv.ParseFloat(sw[1], 64)
		if err != nil {
			log.Fatalf("Couldn't parse bounds: %#v", err)
		}
		neLat, err := strconv.ParseFloat(ne[0], 64)
		if err != nil {
			log.Fatalf("Couldn't parse bounds: %#v", err)
		}
		neLng, err := strconv.ParseFloat(ne[1], 64)
		if err != nil {
			log.Fatalf("Couldn't parse bounds: %#v", err)
		}

		r.Bounds = &maps.LatLngBounds{
			NorthEast: maps.LatLng{Lat: neLat, Lng: neLng},
			SouthWest: maps.LatLng{Lat: swLat, Lng: swLng},
		}
	}
}

func parseLatLng(latlng string, r *maps.GeocodingRequest) {
	if latlng != "" {
		l := strings.Split(latlng, ",")
		lat, err := strconv.ParseFloat(l[0], 64)
		if err != nil {
			log.Fatalf("Couldn't parse latlng: %#v", err)
		}
		lng, err := strconv.ParseFloat(l[1], 64)
		if err != nil {
			log.Fatalf("Couldn't parse latlng: %#v", err)
		}
		r.LatLng = &maps.LatLng{
			Lat: lat,
			Lng: lng,
		}
	}
}

func parseResultType(resultType string, r *maps.GeocodingRequest) {
	if resultType != "" {
		r.ResultType = strings.Split(resultType, "|")
	}
}

func parseLocationType(locationType string, r *maps.GeocodingRequest) {
	if locationType != "" {
		for _, l := range strings.Split(locationType, "|") {
			switch l {
			case "ROOFTOP":
				r.LocationType = append(r.LocationType, maps.GeocodeAccuracyRooftop)
			case "RANGE_INTERPOLATED":
				r.LocationType = append(r.LocationType, maps.GeocodeAccuracyRangeInterpolated)
			case "GEOMETRIC_CENTER":
				r.LocationType = append(r.LocationType, maps.GeocodeAccuracyGeometricCenter)
			case "APPROXIMATE":
				r.LocationType = append(r.LocationType, maps.GeocodeAccuracyApproximate)
			}
		}

	}
}
