// Copyright 2020 by Brian C. Lane <bcl@brianlane.com>. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sgp30_test

import (
	"fmt"
	"log"
	"time"

	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/host"

	"github.com/bcl/air-sensors/sgp30"
)

func Example() {
	// Make sure periph is initialized.
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	// Open a handle to the first available I²C bus:
	bus, err := i2creg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	defer bus.Close()

	d, err := sgp30.New(bus, ".sgp30_baseline", 30*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer d.Halt()

	sn, err := d.GetSerialNumber()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Serial Number: %X\n", sn)

	// Start measuring air quality
	if err = d.StartMeasurements(); err != nil {
		log.Fatal(err)
	}

	for {
		time.Sleep(1 * time.Second)
		if co2, tvoc, err := d.ReadAirQuality(); err != nil {
			log.Fatal(err)
		} else {
			fmt.Printf("CO2 : %d ppm\nTVOC: %d ppb\n", co2, tvoc)
		}
	}
}
