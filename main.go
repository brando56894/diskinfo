package main

import (
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/anatol/smart.go"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
	"github.com/jessevdk/go-flags"
	"github.com/syscll/tempconv"
)

type Options struct {
	Model         bool `short:"m" long:"model" description:"Shows the drive model"`
	SerialNumber  bool `short:"s" long:"serialnumber" description:"Shows the drive's serial number"`
	DeviceName    bool `short:"d" long:"devicename" description:"Shows the drive's Linux device name"`
	WorldWideName bool `short:"w" long:"worldwidename" description:"Shows the drive's World Wide Name"`
	SmartInfo     bool `short:"i" long:"smartinfo" description:"Shows error related SMART info"`
	Temp          bool `short:"t" long:"temperature" description:"Shows the drive's temperature"`
	Sata          bool `short:"a" long:"sata" description:"Shows SATA drives"`
	Scsi          bool `short:"c" long:"scsi" description:"Shows SCSI/SAS drives"`
	Nvme          bool `short:"n" long:"nvme" description:"Shows NVME drives"`
	HideSSD       bool `short:"e" long:"hidessd" description:"Hides SATA SSDs"`
	HideHDD       bool `short:"f" long:"hidehdd" description:"Hides SATA HDDs"`
	All           bool `short:"l" long:"all" description:"Enables all flags"`
}

var options Options
var parser = flags.NewParser(&options, flags.Default)

func main() {
	if _, err := parser.Parse(); err != nil {
		switch flagsErr := err.(type) {
		case flags.ErrorType:
			if flagsErr == flags.ErrHelp {
				os.Exit(0)
			}
			fmt.Println("error: parser.Parse(): ", err)
			os.Exit(1)
		default:
			os.Exit(1)
		}
	}

	if options.All {
		options.Model = true
		options.SerialNumber = true
		options.DeviceName = true
		options.WorldWideName = true
		options.SmartInfo = true
		options.Temp = true
		options.Sata = true
		options.Scsi = true
		options.Nvme = true
	}

	checkUser()
	printInfo(options)

}

func checkUser() {
	user, err := user.Current()
	if user.Name != "root" {
		fmt.Println("This program must be run as root in order to access device information")
		os.Exit(1)
	} else if err != nil {
		fmt.Println("error: user.Current(): ", err)
	}
}

func printInfo(options Options) {
	block, err := ghw.Block()
	if err != nil {
		fmt.Println("error: ghw.Block(): ", err)
		panic(err)
	}

	for _, disk := range block.Disks {

		// some devices do not support SMART interface
		if disk.BusPath == "unknown" || disk.Model == "Card_Reader" || disk.DriveType.String() == "ODD" || strings.Contains(disk.Model, "Virtual") {
			continue
		} else if disk.Model == "Samsung_SSD_850_EVO_120GB" {
			// has different SMART IDs than the standard, figure out how to handle it
			continue
		} else if (disk.DriveType.String() == "SSD" && options.HideSSD) || (disk.DriveType.String() == "HDD" && options.HideHDD) {
			continue
		}

		dev, err := smart.Open("/dev/" + disk.Name)
		if err != nil {
			fmt.Println("error: smart.Open: ", err)
			fmt.Println("disk: ", disk.Model)
			continue
		}

		defer dev.Close()

		switch sm := dev.(type) {
		case *smart.SataDevice:
			if options.Sata {
				printIdentifyingInfo(*disk, options)
				data, readSmartDataErr := sm.ReadSMARTData()

				if readSmartDataErr != nil {
					fmt.Println("error: sm.ReadSMARTData(): ", readSmartDataErr)
				}

				if options.Temp {
					temp, _, _, _, attrErr := data.Attrs[194].ParseAsTemperature()
					if attrErr != nil {
						fmt.Println("error: data.Attrs[194].ParseAsTemperature: ", attrErr)
					}
					fmt.Println("Temperature:", tempF(temp))
				}

				if options.SmartInfo {
					fmt.Println("Power On Hours:", data.Attrs[9].ValueRaw)
					fmt.Println("Reallocated Sectors Count:", data.Attrs[5].ValueRaw)

					// Seagate drives report incorrect values so skip it
					if disk.Model == "ST18000NE000-2YY101" {
						continue
					} else {
						fmt.Println("Raw Read Error Rate:", data.Attrs[1].ValueRaw)
						fmt.Println("Seek Error Rate:", data.Attrs[7].ValueRaw)

					}
				}
			}

		case *smart.ScsiDevice:
			if options.Scsi {
				printIdentifyingInfo(*disk, options)
				notSata(sm, options.Temp, options.SmartInfo)
			}

		case *smart.NVMeDevice:
			if options.Nvme {
				printIdentifyingInfo(*disk, options)
				notSata(sm, options.Temp, options.SmartInfo)
			}
		}
	}
}

func printIdentifyingInfo(disk block.Disk, options Options) {
	fmt.Println("")
	if options.Model {
		fmt.Println("Model:", disk.Model)
	}
	if options.SerialNumber {
		fmt.Println("Serial Number:", disk.SerialNumber)
	}
	if options.DeviceName {
		fmt.Println("Device Name:", disk.Name)
	}
	if options.WorldWideName {
		fmt.Println("WWN:", disk.WWN)
	}
}

// Support for non-SATA SMART info is limited
func notSata(sm smart.Device, showTemp, smartInfo bool) {
	a, err := sm.ReadGenericAttributes()
	if err != nil {
		fmt.Println("error: sm.ReadGenericAttributes(): ", err)
	}

	if showTemp {
		fmt.Println("Temperature:", tempF(int(a.Temperature)))
	}

	if smartInfo {
		fmt.Println("Power On Hours:", a.PowerOnHours)
	}

}

func tempF(temp int) string {
	tempF := tempconv.CelsiusToFahrenheit(tempconv.Celsius(temp))
	return fmt.Sprint(tempconv.Fahrenheit(tempF))
}
