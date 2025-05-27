package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"github.com/openhue/openhue-go"
)

const lightId = "LIGHT_ID"

func (agent *HueAgent) updateBulbState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var body LightStateRequest

	bodyBytes, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		resp := prepareHTTPResponse(http.StatusInternalServerError, "Failed to read request body", nil)
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

	// hard-coding these values as I'm not sure if random user-input values for these attributes are safe for the bulb
	body.Mirek = 153
	body.Brightness = float32(100.0)

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

func NewRequestMultiplexer(agent *HueAgent) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/light/state", agent.updateBulbState)
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
