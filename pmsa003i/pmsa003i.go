// Copyright 2020 by Brian C. Lane <bcl@brianlane.com>. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package pmsa003i

import (
	"fmt"

	"periph.io/x/periph/conn"
	"periph.io/x/periph/conn/i2c"
)

const (
	PMSA003I_ADDR = 0x12
)

func checksum(data []byte) bool {
	var cksum uint16
	for i := 0; i < len(data)-2; i++ {
		cksum = cksum + uint16(data[i])
	}
	return word(data, 0x1e) == cksum
}

type Results struct {
	CfPm1    uint16 // PM1.0 in μg/m3 standard particle
	CfPm2_5  uint16 // PM2.5 in μg/m3 standard particle
	CfPm10   uint16 // PM10 in μg/m3 standard particle
	EnvPm1   uint16 // PM1.0 in μg/m3 atmospheric environment
	EnvPm2_5 uint16 // PM2.5 in μg/m3 atmospheric environment
	EnvPm10  uint16 // PM10 in μg/m3 atmospheric environment
	Cnt0_3   uint16 // Count of particles > 0.3μm in 0.1L of air
	Cnt0_5   uint16 // Count of particles > 0.5μm in 0.1L of air
	Cnt1     uint16 // Count of particles > 1.0μm in 0.1L of air
	Cnt2_5   uint16 // Count of particles > 2.5μm in 0.1L of air
	Cnt5     uint16 // Count of particles > 5.0μm in 0.1L of air
	Cnt10    uint16 // Count of particles > 10.0μm in 0.1L of air
	Version  uint8
}

// New returns a PMSA003I device struct for communicating with the device
//
func New(i i2c.Bus) (*Dev, error) {
	d := &Dev{i2c: &i2c.Dev{Bus: i, Addr: PMSA003I_ADDR}}

	_, err := d.ReadSensor()
	if err != nil {
		return nil, err
	}
	return d, nil
}

type Dev struct {
	i2c conn.Conn // i2c device handle for the sgp30
	err error
}

// Halt implements conn.Resource.
func (d *Dev) Halt() error {
	return nil
}

// ReadSensor returns particle measurement results
func (d *Dev) ReadSensor() (Results, error) {
	// Receive 32 bytes
	var data [32]byte
	if err := d.i2c.Tx(nil, data[:]); err != nil {
		return Results{}, fmt.Errorf("pmsa003i: Error while reading the sensor: %w", err)
	}

	if word(data[:], 0) != 0x424d {
		return Results{}, fmt.Errorf("pmsa003i: Bad start word")
	}
	if !checksum(data[:]) {
		return Results{}, fmt.Errorf("pmsa003i: Bad checksum")
	}
	if data[0x1d] != 0x00 {
		return Results{}, fmt.Errorf("pmsa0031: Error code %x", data[0x1d])
	}

	return Results{
		CfPm1:    word(data[:], 0x04),
		CfPm2_5:  word(data[:], 0x06),
		CfPm10:   word(data[:], 0x08),
		EnvPm1:   word(data[:], 0x0a),
		EnvPm2_5: word(data[:], 0x0c),
		EnvPm10:  word(data[:], 0x0e),
		Cnt0_3:   word(data[:], 0x10),
		Cnt0_5:   word(data[:], 0x12),
		Cnt1:     word(data[:], 0x14),
		Cnt2_5:   word(data[:], 0x16),
		Cnt5:     word(data[:], 0x18),
		Cnt10:    word(data[:], 0x1a),
		Version:  data[0x1c],
	}, nil
}

// word returns 16 bits from the byte stream, starting at index i
func word(data []byte, i int) uint16 {
	return uint16(data[i])<<8 + uint16(data[i+1])
}
