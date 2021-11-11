package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type URLParamKey string

type OpenWeatherOneCallCurrentConditions struct {
	DT        int64   `json:"dt"`
	FeelsLike float32 `json:"feels_like"`
	Weather   []struct {
		Main string `json:"main"`
	} `json:"weather"`
}

type OpenWeatherOneCallAlert struct {
	SenderName  string `json:"sender_name"`
	Event       string `json:"event"`
	Description string `json:"description"`
}

type OpenWeatherOneCallResponse struct {
	Current OpenWeatherOneCallCurrentConditions `json:"current"`
	Alerts  []OpenWeatherOneCallAlert           `json:"alerts"`
}

type MyWeatherResponse struct {
	Timestamp              string
	CurrentTemperatureFeel string   `json:"current_temperature_feel`
	CurrentConditions      []string `json:"current_conditions"`
	// This re-uses the type from OpenWeather, which should be
	// immediately changed if any changes to this data structure
	// are desired
	CurrentAlerts []OpenWeatherOneCallAlert `json:"alerts"`
}

// GetCurrentTemperatureFeel makes a subjective call about what counts
// as what temperature "feel". It uses the magic numbers 277K and 297K
// as boundaries, which are intrinsic to this opinionated function and
// not the program as a whole (so the magic numbers should stay here)
func GetCurrentTemperatureFeel(respParsed *OpenWeatherOneCallResponse) string {
	if respParsed.Current.FeelsLike < 277.0 {
		return "cold"
	} else if respParsed.Current.FeelsLike <= 297 {
		return "moderate"
	} else {
		return "hot"
	}
}

func GetCurrentConditions(respParsed *OpenWeatherOneCallResponse) []string {
	results := make([]string, 0, len(respParsed.Current.Weather))
	for _, weatherCondition := range respParsed.Current.Weather {
		results = append(results, weatherCondition.Main)
	}
	return results
}

func GetCurrentAlerts(respParsed *OpenWeatherOneCallResponse) []OpenWeatherOneCallAlert {
	return append(make([]OpenWeatherOneCallAlert, 0, len(respParsed.Alerts)), respParsed.Alerts...)
}

func main() {
	apiKey := os.Getenv("OPENWEATHERMAP_API_KEY")
	if apiKey == "" {
		os.Stderr.WriteString("Missing required environment variable OPENWEATHERMAP_API_KEY\n")
		os.Exit(1)
	}
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get(`/hello`,
		func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					log.Println(r)
					http.Error(w, http.StatusText(500), 500)
				}
			}()

			w.Write([]byte("hello 1!"))
		})

	r.With(latLonContext).Get(`/helloCtx`,
		func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					log.Println(r)
					http.Error(w, http.StatusText(500), 500)
				}
			}()

			w.Write([]byte("hello 2!"))
		})

	r.With(latLonContext).Get(`/test/lat/-40/lon/40`,
		func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					log.Println(r)
					http.Error(w, http.StatusText(500), 500)
				}
			}()

			w.Write([]byte("hello 3!"))
		})

	r.With(latLonContext).Get(`/test2/lat/{lat:[-+]?[0-9]+}/lon/{lon:[-+]?[0-9]+}`,
		func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					log.Println(r)
					http.Error(w, http.StatusText(500), 500)
				}
			}()

			w.Write([]byte("hello 3!"))
		})

	r.With(latLonContext).Get(`/weather/lat/{lat:[-+]?[0-9]+(?:\.[0-9]+)?}/lon/{lon:[-+]?[0-9]+(?:\.[0-9]+)?}`,
		func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					log.Println(r)
					http.Error(w, http.StatusText(500), 500)
				}
			}()

			// This begins the low-level flow of this flow; again, it
			// should definitely be extracted somewhere, but for now
			// we don't have enough information to know where
			resp, err := http.Get(fmt.Sprintf(
				"https://api.openweathermap.org/data/2.5/onecall?lat=%f&lon=%f&exclude=minutely,hourly,daily&cnt=0&appid=%s",
				r.Context().Value(URLParamKey("lat")).(float64),
				r.Context().Value(URLParamKey("lon")).(float64),
				apiKey,
			))
			panicIfErr(err)

			respBody, err := ioutil.ReadAll(resp.Body)
			panicIfErr(err)

			var respParsed OpenWeatherOneCallResponse
			err = json.Unmarshal(respBody, &respParsed)
			panicIfErr(err)

			myRespBytes, err := json.Marshal(MyWeatherResponse{
				Timestamp:              time.Unix(respParsed.Current.DT, 0).Format(time.RFC3339),
				CurrentTemperatureFeel: GetCurrentTemperatureFeel(&respParsed),
				CurrentConditions:      GetCurrentConditions(&respParsed),
				CurrentAlerts:          GetCurrentAlerts(&respParsed),
			})
			panicIfErr(err)

			w.Write(myRespBytes)
		})
	http.ListenAndServe(":8080", r)
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

func latLonContext(next http.Handler) http.Handler {
	// This should probably be refactored, but the application isn't big
	// enough yet to know the best way to organize things
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				http.Error(w, http.StatusText(400), 400)
			}
		}()

		lat, err := strconv.ParseFloat(chi.URLParam(r, "lat"), 64)
		panicIfErr(err)
		ctx := context.WithValue(r.Context(), URLParamKey("lat"), lat)

		lon, err := strconv.ParseFloat(chi.URLParam(r, "lon"), 64)
		panicIfErr(err)
		ctx = context.WithValue(ctx, URLParamKey("lon"), lon)

		log.Println(ctx)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
