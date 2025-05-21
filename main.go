package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/openhue/openhue-go"
)

const lightId = "LIGHT_ID"

type HueAgent struct {
	home *openhue.Home
}

func NewHueAgent() *HueAgent {
	h, err := openhue.NewHome(openhue.LoadConfNoError())
	if err != nil {
		log.Fatalf("Error creating new home: %v\n", err)
	}
	return &HueAgent{
		home: h,
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file")
	}

	hueAgent := NewHueAgent()
	lightsMap, _ := hueAgent.home.GetLights()

	var myLight openhue.LightGet
	for k, v := range lightsMap {
		if k == os.Getenv(lightId) {
			myLight = v
		}
	}

	// light variables
	mirekValue := 153
	on := false
	brightness := float32(100.0)

	err := hueAgent.home.UpdateLight(*myLight.Id, openhue.LightPut{
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
