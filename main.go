package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"github.com/openhue/openhue-go"
)

type contextKey string

const (
	lightId string = "LIGHT_ID"

	lightStatePayloadKey contextKey = "lightStatePayload"
)

func (agent *HueAgent) updateBulbState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	body, ok := r.Context().Value(lightStatePayloadKey).(LightStateRequest)
	if !ok {
		resp := prepareHTTPResponse(http.StatusInternalServerError, "Missing validation payload in context", nil)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(resp)
		return
	}

	var myLight openhue.LightGet
	lightsMap, err := agent.home.GetLights()
	if err != nil {
		// this happened during a power cut
		log.Printf("Error occurred while getting lights: %s\n", err)
		return
	}

	for k, v := range lightsMap {
		if k == os.Getenv(lightId) {
			myLight = v
		}
	}

	if err := agent.home.UpdateLight(*myLight.Id, openhue.LightPut{
		On: &openhue.On{On: &body.On},
		ColorTemperature: &openhue.ColorTemperature{
			Mirek: &body.Mirek,
		},
		Dimming: &openhue.Dimming{Brightness: &body.Brightness},
	}); err != nil {
		resp := prepareHTTPResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to update light state. Error: %s", err), nil)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(resp)
		return
	}

	resp := prepareHTTPResponse(http.StatusOK, "Successfully updated light state", nil)
	log.Println("Successfully updated light state")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

func NewHueAgent() *HueAgent {
	h, err := openhue.NewHome(openhue.LoadConfNoError())
	if err != nil {
		log.Fatalf("Error creating new home: %v\n", err)
	}
	return &HueAgent{
		home: h,
		Wg:   &sync.WaitGroup{},
	}
}

func validationMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload LightStateRequest

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			resp := prepareHTTPResponse(http.StatusBadRequest, "Invalid JSON payload", nil)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(resp)
			return
		}

		if payload.Mirek < 153 || payload.Mirek > 370 {
			resp := prepareHTTPResponse(http.StatusBadRequest, "Invalid mirek value", nil)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(resp)
			return
		}

		if payload.Brightness < 0 || payload.Brightness > 100 {
			resp := prepareHTTPResponse(http.StatusBadRequest, "Invalid brightness value", nil)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(resp)
			return
		}

		ctx := context.WithValue(r.Context(), lightStatePayloadKey, payload)
		next(w, r.WithContext(ctx))
	}
}

func NewRequestMultiplexer(agent *HueAgent) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/light/state", validationMiddleware(agent.updateBulbState))
	return mux
}

func createHTTPServer(agent *HueAgent) *http.Server {
	return &http.Server{
		Addr:    ":9000",
		Handler: NewRequestMultiplexer(agent),
	}
}

func startHTTPServer(server *http.Server, agent *HueAgent) {
	agent.Wg.Add(1)
	go func() {
		defer agent.Wg.Done()
		log.Printf("HTTP server listening on port %s\n", server.Addr)

		err := server.ListenAndServe()
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return
			}
			log.Fatalf("ListenAndServe Error: %s\n", err)
		}
	}()
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file")
	}

	agent := NewHueAgent()
	httpServer := createHTTPServer(agent)
	startHTTPServer(httpServer, agent)
	agent.Wg.Wait()
}
