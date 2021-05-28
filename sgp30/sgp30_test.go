// Copyright 2020 by Brian C. Lane <bcl@brianlane.com>. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package sgp30

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"periph.io/x/periph/conn/i2c/i2ctest"
)

var (
	BadSerialNumber    = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0}
	GoodSerialNumber   = []byte{0x00, 0x00, 0x81, 0x01, 0x57, 0x9C, 0xAC, 0xA2, 0x54}
	BadBaselineData    = []byte{0, 0, 0, 0, 0, 0}
	GoodBaselineData   = []byte{0x88, 0xa1, 0x58, 0x8d, 0xc4, 0x61}
	BadFeaturesData    = []byte{0, 0, 0}
	GoodFeaturesData   = []byte{0x00, 0x22, 0x65}
	BadAirQualityData  = []byte{0, 0, 0, 0, 0, 0}
	GoodAirQualityData = []byte{0x01, 0x9e, 0x53, 0x00, 0x0d, 0xcd}
)

func TestWord(t *testing.T) {
	data := []byte{0x00, 0x01, 0x80, 0x0A, 0x55, 0xAA, 0xFF, 0x7F}
	result := []uint16{0x0001, 0x800A, 0x55AA, 0xFF7F}
	for i := 0; i < len(result); i += 1 {
		if word(data, i*2) != result[i] {
			t.Errorf("word error: i == %d", i)
		}
	}
}

func TestChecksum(t *testing.T) {
	if !checkCRC8(GoodSerialNumber[0:3]) {
		t.Fatal("serial number word 1 CRC8 error")
	}
	if !checkCRC8(GoodSerialNumber[3:6]) {
		t.Fatal("serial number word 2 CRC8 error")
	}
	if !checkCRC8(GoodSerialNumber[6:9]) {
		t.Fatal("serial number word 3 CRC8 error")
	}
}

func TestFailReadChipID(t *testing.T) {
	bus := i2ctest.Playback{
		// Chip ID detection read fail.
		Ops:       []i2ctest.IO{},
		DontPanic: true,
	}
	if _, err := New(&bus, "", time.Second); err == nil {
		t.Fatal("can't read chip ID")
	}
}

func TestBadSerialNumber(t *testing.T) {
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Bad serial number
			{Addr: 0x58, W: []byte{0x36, 0x82}, R: BadSerialNumber},
		},
	}
	if _, err := New(&bus, "", time.Second); err == nil {
		t.Fatal("Bad serial number Error")
	}
}

func TestGoodSerialNumber(t *testing.T) {
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Good serial number
			{Addr: 0x58, W: []byte{0x36, 0x82}, R: GoodSerialNumber},
		},
	}
	if _, err := New(&bus, "", time.Second); err != nil {
		t.Fatalf("Good serial number Error: %s", err)
	}
}

func TestBadBaselineData(t *testing.T) {
	// Temporary baseline file, defer removal
	bf, err := ioutil.TempFile("", "sgp30.")
	if err != nil {
		t.Fatalf("TempFile Error: %s", err)
	}
	defer os.Remove(bf.Name())

	_, err = bf.Write(BadBaselineData)
	if err != nil {
		t.Fatalf("TempFile Write Error: %s", err)
	}

	// Calling New with a baseline reads the serial number, starts measurements,
	// and then writes the baseline data
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Good serial number
			{Addr: 0x58, W: []byte{0x36, 0x82}, R: GoodSerialNumber},
			{Addr: 0x58, W: []byte{0x20, 0x03}, R: []byte{}},
			{Addr: 0x58, W: []byte{0x20, 0x15}, R: BadBaselineData},
		},
	}
	if _, err := New(&bus, bf.Name(), time.Second); err == nil {
		t.Fatal("Bad baseline data")
	}
}

func TestGoodBaselineData(t *testing.T) {
	// Temporary baseline file, defer removal
	bf, err := ioutil.TempFile("", "sgp30.")
	if err != nil {
		t.Fatalf("TempFile Error: %s", err)
	}
	defer os.Remove(bf.Name())

	_, err = bf.Write(GoodBaselineData)
	if err != nil {
		t.Fatalf("TempFile Write Error: %s", err)
	}

	// The CO2 and TVOC data is swapped when writing it back to the SGP30
	BaselineWrite := append(append([]byte{0x20, 0x1e}, GoodBaselineData[3:6]...), GoodBaselineData[0:3]...)

	// Calling New with a baseline reads the serial number, starts measurements,
	// and then writes the baseline data
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Good serial number
			{Addr: 0x58, W: []byte{0x36, 0x82}, R: GoodSerialNumber},
			{Addr: 0x58, W: []byte{0x20, 0x03}, R: []byte{}},
			{Addr: 0x58, W: BaselineWrite, R: []byte{}},
		},
	}
	if _, err := New(&bus, bf.Name(), time.Second); err != nil {
		t.Fatalf("Good Baseline Error: %s", err)
	}
}

func TestBadFeatures(t *testing.T) {
	// Calling New with a baseline reads the serial number
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Good serial number
			{Addr: 0x58, W: []byte{0x36, 0x82}, R: GoodSerialNumber},
			{Addr: 0x58, W: []byte{0x20, 0x2f}, R: BadFeaturesData},
		},
	}
	d, err := New(&bus, "", time.Second)
	if err != nil {
		t.Fatalf("Bad Features: %s", err)
	}
	if _, _, err := d.GetFeatures(); err == nil {
		t.Fatal("Bad Features Error")
	}
}

func TestGoodFeatures(t *testing.T) {
	// Calling New with a baseline reads the serial number
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Good serial number
			{Addr: 0x58, W: []byte{0x36, 0x82}, R: GoodSerialNumber},
			{Addr: 0x58, W: []byte{0x20, 0x2f}, R: GoodFeaturesData},
		},
	}
	d, err := New(&bus, "", time.Second)
	if err != nil {
		t.Fatalf("Good Features: %s", err)
	}
	ProdType, ProdVersion, err := d.GetFeatures()
	if err != nil {
		t.Fatalf("Good Features Error: %s", err)
	}
	if ProdType != 0x00 {
		t.Error("Wrong Product type")
	}
	if ProdVersion != 0x22 {
		t.Error("Wrong Product version")
	}
}

func TestReadBadBaseline(t *testing.T) {
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Good serial number
			{Addr: 0x58, W: []byte{0x36, 0x82}, R: GoodSerialNumber},
			{Addr: 0x58, W: []byte{0x20, 0x15}, R: BadBaselineData},
		},
	}
	d, err := New(&bus, "", time.Second)
	if err != nil {
		t.Fatalf("Good serial number Error: %s", err)
	}
	if _, err := d.ReadBaseline(); err == nil {
		t.Fatal("Read Bad Baseline Error")
	}
}

func TestReadGoodBaseline(t *testing.T) {
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Good serial number
			{Addr: 0x58, W: []byte{0x36, 0x82}, R: GoodSerialNumber},
			{Addr: 0x58, W: []byte{0x20, 0x15}, R: GoodBaselineData},
		},
	}
	d, err := New(&bus, "", time.Second)
	if err != nil {
		t.Fatalf("Good serial number Error: %s", err)
	}
	if _, err := d.ReadBaseline(); err != nil {
		t.Fatalf("Read Good Baseline Error: %s", err)
	}
}

func TestBadAirQuality(t *testing.T) {
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Good serial number
			{Addr: 0x58, W: []byte{0x36, 0x82}, R: GoodSerialNumber},
			{Addr: 0x58, W: []byte{0x20, 0x08}, R: []byte{}},
			{Addr: 0x58, W: []byte{}, R: BadAirQualityData},
		},
	}
	d, err := New(&bus, "", time.Second)
	if err != nil {
		t.Fatalf("Good serial number Error: %s", err)
	}
	if _, _, err := d.ReadAirQuality(); err == nil {
		t.Fatalf("Read Bad AirQuality Error")
	}
}

func TestGoodAirQuality(t *testing.T) {
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Good serial number
			{Addr: 0x58, W: []byte{0x36, 0x82}, R: GoodSerialNumber},
			{Addr: 0x58, W: []byte{0x20, 0x08}, R: []byte{}},
			{Addr: 0x58, W: []byte{}, R: GoodAirQualityData},
		},
	}
	d, err := New(&bus, "", time.Second)
	if err != nil {
		t.Fatalf("Good serial number Error: %s", err)
	}
	co2, tvoc, err := d.ReadAirQuality()
	if err != nil {
		t.Fatalf("Read Good AirQuality Error: %s", err)
	}
	if co2 != 414 {
		t.Error("CO2 reading is wrong")
	}
	if tvoc != 13 {
		t.Error("TVOC reading is wrong")
	}
}
