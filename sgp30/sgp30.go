// Copyright 2020 by Brian C. Lane <bcl@brianlane.com>. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sgp30

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/sigurn/crc8"
	"periph.io/x/periph/conn"
	"periph.io/x/periph/conn/i2c"
)

const (
	SGP30_ADDR = 0x58
)

var (
	CRC8_SGP30 = crc8.MakeTable(crc8.Params{
		Poly:   0x31,
		Init:   0xFF,
		RefIn:  false,
		RefOut: false,
		XorOut: 0x00,
		Check:  0xA1,
		Name:   "CRC-8/SGP30",
	})
)

func checkCRC8(data []byte) bool {
	return crc8.Checksum(data[:], CRC8_SGP30) == 0x00
}

// New returns a SGP30 device struct for communicating with the device
//
// If baselineFile is passed the baseline calibration data will be read from the file
// at startup, and new data will be saved at baselineInterval interval when ReadAirQuality
// is called.
//
// eg. pass 30 * time.Second to save the baseline data every 30 seconds
func New(i i2c.Bus, baselineFile string, baselineInterval time.Duration) (*Dev, error) {
	d := &Dev{i2c: &i2c.Dev{Bus: i, Addr: SGP30_ADDR}}
	if _, err := d.GetSerialNumber(); err != nil {
		return nil, err
	}

	// Restore the baseline from the saved data if it exists
	if len(baselineFile) > 0 {
		d.baselineFile = baselineFile
		d.baselineInterval = baselineInterval
		d.lastSave = time.Now()
		// Restore the baseline data if it exists, ignore missing file
		if baseline, err := ioutil.ReadFile(baselineFile); err == nil {
			err = d.SetBaseline(baseline)
			if err != nil {
				return nil, err
			}
		}
	}
	return d, nil
}

type Dev struct {
	i2c              conn.Conn     // i2c device handle for the sgp30
	baselineFile     string        // Path and filename for storing baseline values
	baselineInterval time.Duration // How often to save the baseline data
	lastSave         time.Time     // Last time baseline was saved
	err              error
}

// Halt implements conn.Resource.
func (d *Dev) Halt() error {
	return nil
}

// GetSerialNumber returns the 48 bit serial number of the device
func (d *Dev) GetSerialNumber() (uint64, error) {
	// Send a 0x3682
	// Receive 3 words + 8 bit CRC on each
	var data [9]byte
	if err := d.i2c.Tx([]byte{0x36, 0x82}, data[:]); err != nil {
		return 0, fmt.Errorf("sgp30: Error while reading serial number: %w", err)
	}

	if !checkCRC8(data[0:3]) {
		return 0, fmt.Errorf("sgp30: serial number word 1 CRC8 failed on: %v", data[0:3])
	}
	if !checkCRC8(data[3:6]) {
		return 0, fmt.Errorf("sgp30: serial number word 2 CRC8 failed on: %v", data[3:6])
	}
	if !checkCRC8(data[6:9]) {
		return 0, fmt.Errorf("sgp30: serial number word 3 CRC8 failed on: %v", data[6:9])
	}

	return uint64(word(data[:], 0))<<24 + uint64(word(data[:], 3))<<16 + uint64(word(data[:], 6)), nil
}

// GetFeatures returns the 8 bit product type, and 8 bit product version
func (d *Dev) GetFeatures() (uint8, uint8, error) {
	// Send a 0x202f
	// Receive 1 word + 8 bit CRC
	var data [3]byte
	if err := d.i2c.Tx([]byte{0x20, 0x2f}, data[:]); err != nil {
		return 0, 0, fmt.Errorf("sgp30: Error while reading features: %w", err)
	}

	if !checkCRC8(data[0:3]) {
		return 0, 0, fmt.Errorf("sgp30: features CRC8 failed on: %v", data[0:3])
	}

	return data[0], data[1], nil
}

// StartMeasurements sends the Inlet Air Quality command to start measuring
// ReadAirQuality needs to be called every second after this has been sent
//
// Note that for 15s after the measurements have started the readings will return
// 400ppm CO2 and 0ppb TVOC
func (d *Dev) StartMeasurements() error {
	// Send a 0x2003
	if err := d.i2c.Tx([]byte{0x20, 0x03}, nil); err != nil {
		return fmt.Errorf("sgp30: Error starting air quality measurements: %w", err)
	}

	return nil
}

// ReadAirQuality returns the CO2 and TVOC readings as 16 bit values
// CO2 is in ppm and TVOC is in ppb
//
// If a baselineFile was passed to New the baseline data will be saved to disk every
// baselineInterval
func (d *Dev) ReadAirQuality() (uint16, uint16, error) {
	// Send a 0x2008
	// Receive 2 words with + 8 bit CRC on each
	if err := d.i2c.Tx([]byte{0x20, 0x08}, nil); err != nil {
		return 0, 0, fmt.Errorf("sgp30: Error while requesting air quality: %w", err)
	}

	// Requires a short delay before reading results
	time.Sleep(10 * time.Millisecond)
	var data [6]byte
	if err := d.i2c.Tx(nil, data[:]); err != nil {
		return 0, 0, fmt.Errorf("sgp30: Error while reading air quality: %w", err)
	}

	if !checkCRC8(data[0:3]) {
		return 0, 0, fmt.Errorf("sgp30: read air quality word 1 CRC8 failed on: %v", data[0:3])
	}
	if !checkCRC8(data[3:6]) {
		return 0, 0, fmt.Errorf("sgp30: read air quality word 2 CRC8 failed on: %v", data[3:6])
	}

	if len(d.baselineFile) > 0 && time.Since(d.lastSave) >= d.baselineInterval {
		d.lastSave = time.Now()
		baseline, err := d.ReadBaseline()
		if err != nil {
			return 0, 0, fmt.Errorf("sgp30: Error while reading baseline: %w", err)
		}
		ioutil.WriteFile(d.baselineFile, baseline[:], 0644)
	}

	return word(data[:], 0), word(data[:], 3), nil
}

// ReadBaseline returns the 6 data bytes for the measurement baseline
// These values should be saved to disk and restore using SetBaseline when the program
// restarts.
func (d *Dev) ReadBaseline() ([6]byte, error) {
	// Send a 0x2015
	// Receive 2 words + 8 bit CRC on each
	var data [6]byte
	if err := d.i2c.Tx([]byte{0x20, 0x15}, data[:]); err != nil {
		return [6]byte{}, fmt.Errorf("sgp30: Error while reading baseline: %w", err)
	}

	if !checkCRC8(data[0:3]) {
		return [6]byte{}, fmt.Errorf("sgp30: baseline word 1 CRC8 failed on: %v", data[0:3])
	}
	if !checkCRC8(data[3:6]) {
		return [6]byte{}, fmt.Errorf("sgp30: baseline word 2 CRC8 failed on: %v", data[3:6])
	}

	return data, nil
}

// SetBaseline sets the measurement baseline data bytes
// The values should have been previously read from the device using ReadBaseline
//
// NOTE: The data order for setting it is TVOC, CO2 even though the order when
// reading is CO2, TVOC. This assumes that the baseline data passed in is CO2, TVOC
func (d *Dev) SetBaseline(baseline []byte) error {
	if !checkCRC8(baseline[0:3]) {
		return fmt.Errorf("sgp30: set baseline word 1 CRC8 failed on: %v", baseline[0:3])
	}
	if !checkCRC8(baseline[3:6]) {
		return fmt.Errorf("sgp30: set baseline word 2 CRC8 failed on: %v", baseline[3:6])
	}

	// Send InitAirQuality
	d.StartMeasurements()

	// Send a 0x201e + TVOC, CO2 baseline data (2 words + CRCs)
	data := append(append([]byte{0x20, 0x1e}, baseline[3:6]...), baseline[0:3]...)
	if err := d.i2c.Tx(data, nil); err != nil {
		return fmt.Errorf("sgp30: Error while setting baseline: %w", err)
	}
	return nil
}

// word returns 16 bits from the byte stream, starting at index i
func word(data []byte, i int) uint16 {
	return uint16(data[i])<<8 + uint16(data[i+1])
}
