package main

import (
	"fmt"
	"os"
	"os/user"

	"github.com/anatol/smart.go"
	"github.com/jaypipes/ghw"
	"github.com/syscll/tempconv"
)

func main() {
	user, err := user.Current()
	if user.Name != "root" {
		fmt.Println("This program must be run as root in order to access device information")
		os.Exit(1)
	} else if err != nil {
		panic(err)
	}

	block, err := ghw.Block()
	if err != nil {
		panic(err)
	}

	for _, disk := range block.Disks {
		if disk.BusPath == "unknown" {
			// some devices (like dmcrypt) do not support SMART interface
			continue
		} else if disk.Model == "Samsung_SSD_850_EVO_120GB" {
			// has different SMART IDs than the standard, figure out how to handle it
			continue
		}

		fmt.Println("Model:", disk.Model)
		fmt.Println("Serial Number:", disk.SerialNumber)
		fmt.Println("Device Name:", disk.Name)
		fmt.Println("WWN:", disk.WWN)

		dev, err := smart.Open("/dev/" + disk.Name)
		if err != nil {
			fmt.Printf("error: %v\n\n", err)
			continue
		}

		defer dev.Close()

		switch sm := dev.(type) {
		case *smart.SataDevice:
			data, readSmartDataErr := sm.ReadSMARTData()

			if readSmartDataErr != nil {
				panic(readSmartDataErr)
			}

			temp, _, _, _, attrErr := data.Attrs[194].ParseAsTemperature()
			if attrErr != nil {
				fmt.Printf("Error reading and parsing temperature: %v\n", attrErr)
			}

			fmt.Println("Temperature:", tempF(temp))
			fmt.Println("Power On Hours:", data.Attrs[9].ValueRaw)

		case *smart.ScsiDevice:
			a, err := sm.ReadGenericAttributes()
			if err != nil {
				panic(err)
			}

			fmt.Println("Temperature:", tempF(int(a.Temperature)))
			fmt.Println("Power On Hours:", a.PowerOnHours)

		case *smart.NVMeDevice:
			a, err := sm.ReadGenericAttributes()
			if err != nil {
				panic(err)
			}

			fmt.Println("Temperature:", tempF(int(a.Temperature))) // in Celsius
			fmt.Println("Power On Hours: ", a.PowerOnHours)
		}

		fmt.Println("")
	}

}

func tempF(temp int) string {
	tempF := tempconv.CelsiusToFahrenheit(tempconv.Celsius(temp))
	return fmt.Sprint(tempconv.Fahrenheit(tempF))
}
