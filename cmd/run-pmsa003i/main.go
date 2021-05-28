// Copyright 2020 by Brian C. Lane <bcl@brianlane.com>. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"time"

	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/host"

	"github.com/bcl/air-sensors/pmsa003i"
)

func main() {
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

	d, err := pmsa003i.New(bus)
	if err != nil {
		log.Fatal(err)
	}

	for start := time.Now(); time.Since(start) < time.Second*30; {
		time.Sleep(1 * time.Second)

		// Read the PMSA003i sensor data
		r, err := d.ReadSensor()
		if err != nil {
			// Checksum failures could be transient
			fmt.Println(err)
		} else {
			fmt.Println()
			fmt.Printf("PM1.0  %3d μg/m3\n", r.EnvPm1)
			fmt.Printf("PM2.5  %3d μg/m3\n", r.EnvPm2_5)
			fmt.Printf("PM10   %3d μg/m3\n", r.EnvPm10)
			fmt.Println("Counters in 0.1L of air")
			fmt.Printf("%d > 0.3μm\n", r.Cnt0_3)
			fmt.Printf("%d > 0.5μm\n", r.Cnt0_5)
			fmt.Printf("%d > 1.0μm\n", r.Cnt1)
			fmt.Printf("%d > 2.5μm\n", r.Cnt2_5)
			fmt.Printf("%d > 5.0μm\n", r.Cnt5)
			fmt.Printf("%d > 10μm\n", r.Cnt10)
		}
	}
}
