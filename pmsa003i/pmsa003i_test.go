// Copyright 2020 by Brian C. Lane <bcl@brianlane.com>. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package pmsa003i

import (
	"fmt"
	"strings"
	"testing"

	"periph.io/x/periph/conn/i2c/i2ctest"
)

var (
	GoodSensorData = []byte{
		0x42, 0x4d, 0x00, 0x1c, 0x00, 0x00, 0x00, 0x01,
		0x00, 0x05, 0x00, 0x00, 0x00, 0x01, 0x00, 0x05,
		0x00, 0x7e, 0x00, 0x2a, 0x00, 0x0f, 0x00, 0x09,
		0x00, 0x03, 0x00, 0x03, 0x97, 0x00, 0x02, 0x14}
	BadStartSensorData = []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	BadChecksumSensorData = []byte{
		0x42, 0x4d, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
)

func TestWord(t *testing.T) {
	data := []byte{0x00, 0x01, 0x80, 0x0A, 0x55, 0xAA, 0xFF, 0x7F}
	result := []uint16{0x0001, 0x800A, 0x55AA, 0xFF7F}
	for i := 0; i < len(result); i++ {
		if word(data, i*2) != result[i] {
			t.Errorf("word error: i == %d", i)
		}
	}
}

func TestChecksum(t *testing.T) {
	if !checksum(GoodSensorData) {
		t.Fatal("Checksum Error")
	}
}

func TestFailReadChipID(t *testing.T) {
	bus := i2ctest.Playback{
		// Chip ID detection read fail.
		Ops:       []i2ctest.IO{},
		DontPanic: true,
	}
	if _, err := New(&bus); err == nil {
		t.Fatal("can't read chip ID")
	}
}

func TestBadSensorData(t *testing.T) {
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Bad sensor data
			{Addr: 0x12, W: []byte{}, R: BadStartSensorData},
		},
	}
	if _, err := New(&bus); err == nil {
		t.Fatal("Bad sensor data Error")
	}
}

func TestGoodSensorData(t *testing.T) {
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Bad sensor data
			{Addr: 0x12, W: []byte{}, R: GoodSensorData},
		},
	}
	if _, err := New(&bus); err != nil {
		t.Fatalf("Good sensor data Error: %s", err)
	}
}

func TestReadSensorBadStart(t *testing.T) {
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Bad start word sensor data
			{Addr: 0x12, W: []byte{}, R: GoodSensorData},
			{Addr: 0x12, W: []byte{}, R: BadStartSensorData},
		},
	}
	d, err := New(&bus)
	if err != nil {
		t.Fatalf("Good sensor data Error: %s", err)
	}
	_, err = d.ReadSensor()
	if err == nil {
		t.Fatal("Read Sensor bad start Error")
	}
	if !strings.Contains(fmt.Sprintf("%s", err), "Bad start word") {
		t.Fatalf("Not bad start Error: %s", err)
	}
}

func TestReadSensorBadChecksum(t *testing.T) {
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Bad checksum sensor data
			{Addr: 0x12, W: []byte{}, R: GoodSensorData},
			{Addr: 0x12, W: []byte{}, R: BadChecksumSensorData},
		},
	}
	d, err := New(&bus)
	if err != nil {
		t.Fatalf("Good sensor data Error: %s", err)
	}
	_, err = d.ReadSensor()
	if err == nil {
		t.Fatal("Read Sensor bad checksum Error")
	}
	if !strings.Contains(fmt.Sprintf("%s", err), "Bad checksum") {
		t.Fatalf("Not bad checksum Error: %s", err)
	}
}

func TestReadSensor(t *testing.T) {
	bus := i2ctest.Playback{
		Ops: []i2ctest.IO{
			// Bad checksum sensor data
			{Addr: 0x12, W: []byte{}, R: GoodSensorData},
			{Addr: 0x12, W: []byte{}, R: GoodSensorData},
		},
	}
	d, err := New(&bus)
	if err != nil {
		t.Fatalf("Good sensor data Error: %s", err)
	}
	r, err := d.ReadSensor()
	if err != nil {
		t.Fatalf("Read Sensor Error: %s", err)
	}
	expected := Results{
		CfPm1:    0,
		CfPm2_5:  1,
		CfPm10:   5,
		EnvPm1:   0,
		EnvPm2_5: 1,
		EnvPm10:  5,
		Cnt0_3:   126,
		Cnt0_5:   42,
		Cnt1:     15,
		Cnt2_5:   9,
		Cnt5:     3,
		Cnt10:    3,
		Version:  151,
	}
	if r != expected {
		t.Fatalf("Read Sensor Data Error: %v", r)
	}
}
