package http

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/vishen/go-chromecast/application"
	"github.com/vishen/go-chromecast/dns"
	"github.com/vishen/go-chromecast/tts"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

func (h *Handler) tts(w http.ResponseWriter, r *http.Request) {
	var payload TTSPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.logAlways("tts payload: %+v", payload)
	if payload.DeviceUuid == "" {
		if h.deviceUuid == "" {
			httpValidationError(w, "missing 'deviceUuid' in json payload or 'uuid' in program args")
			return
		}
		payload.DeviceUuid = h.deviceUuid
	}

	if payload.GoogleServiceAccount == "" {
		if h.googleServiceAccount == "" {
			httpValidationError(w, "missing 'googleServiceAccount' in json payload or 'google-service-account' in program args")
			return
		}
		payload.GoogleServiceAccount = h.googleServiceAccount
	}
	if payload.LanguageCode == "" {
		if h.languageCode == "" {
			httpValidationError(w, "missing 'languageCode' in json payload or 'language-code' in program args")
			return
		}
		payload.LanguageCode = h.languageCode
	}

	app, ok := getOrConnectApp(payload.DeviceUuid, h)
	if ok {
		play(app, &payload, w)
	} else {
		httpValidationError(w, "device uuid is not found")
		return
	}

}

func getOrConnectApp(deviceUUID string, h *Handler) (*application.Application, bool) {
	app, ok := h.app(deviceUUID)
	if ok {
		return app, ok
	} else {
		deviceAddr := ""
		devicePort := ""
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		devicesChan, err := dns.DiscoverCastDNSEntries(ctx, nil)
		if err != nil {
			h.log("error discovering entries: %v", err)
			return nil, false
		}

		for device := range devicesChan {
			// TODO: Should there be a lookup by name as well?
			if device.UUID == deviceUUID {
				deviceAddr = device.AddrV4.String()
				// TODO: This is an unnessecary conversion since
				// we cast back to int a bit later.
				devicePort = strconv.Itoa(device.Port)
			}
		}

		if deviceAddr == "" || devicePort == "" {
			h.log("'port' and 'addr' missing from query params and uuid device lookup returned no results")
			return nil, false
		}

		h.logAlways("connecting to addr=%s port=%s...", deviceAddr, devicePort)

		devicePortI, err := strconv.Atoi(devicePort)
		if err != nil {
			h.log("device port %q is not a number: %v", devicePort, err)
			return nil, false
		}

		applicationOptions := []application.ApplicationOption{
			application.WithDebug(h.verbose),
			application.WithCacheDisabled(true),
		}

		app := application.NewApplication(applicationOptions...)
		if err := app.Start(deviceAddr, devicePortI); err != nil {
			h.logAlways("unable to start application: %v", err)
			return nil, false
		}
		h.mu.Lock()
		h.apps[deviceUUID] = app
		h.mu.Unlock()
		return app, true
	}
}

func play(app *application.Application, payload *TTSPayload, w http.ResponseWriter) {

	fmt.Printf("play text=%s", payload.Text)
	googleServiceAccountJson, err := ioutil.ReadFile(payload.GoogleServiceAccount)
	if err != nil {
		httpValidationError(w, fmt.Sprintf("unable to open google service account file: %v\n", err))
		return
	}

	data, err := tts.Create(payload.Text, googleServiceAccountJson, payload.LanguageCode)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	f, err := ioutil.TempFile("", "go-chromecast-tts")
	if err != nil {
		fmt.Printf("unable to create temp file: %v", err)
		return
	}
	defer os.Remove(f.Name())

	if _, err := f.Write(data); err != nil {
		httpValidationError(w, fmt.Sprintf("unable to write to temp file: %v\n", err))

		return
	}
	if err := f.Close(); err != nil {
		httpValidationError(w, fmt.Sprintf("unable to close temp file: %v\n", err))
		return
	}

	if err := app.Load(f.Name(), "audio/mp3", false, false, false); err != nil {
		httpValidationError(w, fmt.Sprintf("unable to load media to device: %v\n", err))
		return
	}
	return
}

type TTSPayload struct {
	Text                 string
	DeviceUuid           string
	GoogleServiceAccount string
	LanguageCode         string
}
