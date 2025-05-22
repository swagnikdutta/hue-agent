package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"github.com/openhue/openhue-go"
)

const lightId = "LIGHT_ID"

type HueAgent struct {
	Wg   *sync.WaitGroup
	home *openhue.Home
}

func (agent *HueAgent) calendarBulbHandler(w http.ResponseWriter, r *http.Request) {
	bulbState := r.PathValue("state")
	if bulbState != "on" && bulbState != "off" {
		errMsg := fmt.Sprintf("Invalid value %q for bulb state. Allowed values: 'on', 'off'\n", bulbState)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	lightsMap, _ := agent.home.GetLights()
	var myLight openhue.LightGet
	for k, v := range lightsMap {
		if k == os.Getenv(lightId) {
			myLight = v
		}
	}

	// light variables
	mirekValue := 153
	brightness := float32(100.0)
	on := false
	if bulbState == "on" {
		on = true
	}

	err := agent.home.UpdateLight(*myLight.Id, openhue.LightPut{
		On: &openhue.On{On: &on},
		ColorTemperature: &openhue.ColorTemperature{
			Mirek: &mirekValue,
		},
		Dimming: &openhue.Dimming{Brightness: &brightness},
	})
	if err != nil {
		fmt.Printf("Error updating light: %v\n", err)
	}
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
	mux.HandleFunc("/calendar/{state}", agent.calendarBulbHandler)
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
