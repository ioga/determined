package internal

import (
	"encoding/csv"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/device"
)

func getRocmVersion() (string, error) {
	cmd := exec.Command("rocm-smi", "--showdriverversion", "--csv")
	out, err := cmd.Output()

	if execError, ok := err.(*exec.Error); ok && execError.Err == exec.ErrNotFound {
		return "", nil
	} else if err != nil {
		log.WithError(err).WithField("output", string(out)).Warnf("error while executing rocm-smi")
		return "", nil
	}

	r := csv.NewReader(strings.NewReader(string(out)))

	var record []string

	// First line is the header, second line is data.
	// Example input to be parsed:
	//
	// device,Driver version
	// cardsystem,5.11.32.21.40
	//
	for i := 0; i < 2; i++ {
		record, err = r.Read()
		switch {
		case err == io.EOF:
			return unknownGPUDriverVersion, nil
		case err != nil:
			return "", errors.Wrap(err, "error parsing output of rocm-smi as csv")
		case len(record) != 2:
			return "", errors.New(
				"error parsing output of rocm-smi; GPU record should have exactly 1 field")
		case i == 0:
			continue
		}
	}

	return record[1], nil
}

/*
Example output for rocm-smi card detection:

$ rocm-smi  --showid --showuniqueid --showproductname --csv
device,GPU ID,Unique ID,Card series,Card model,Card vendor,Card SKU
card0,0x738c,0xa0ef0d3db6f12111,0x738c,0x0c34,Advanced Micro Devices Inc. [AMD/ATI],D34314
card1,0x738c,0xa38132cf2362b8c3,0x738c,0x0c34,Advanced Micro Devices Inc. [AMD/ATI],D34314
card2,0x738c,0x1a0c44eae3f2753e,0x738c,0x0c34,Advanced Micro Devices Inc. [AMD/ATI],D34314
card3,0x738c,0xfe1b937ecaa13ccc,0x738c,0x0c34,Advanced Micro Devices Inc. [AMD/ATI],D34314
card4,0x738c,0x11c62d148b1506b1,0x738c,0x0c34,Advanced Micro Devices Inc. [AMD/ATI],D34314
card5,0x738c,0x7a85070458971be6,0x738c,0x0c34,Advanced Micro Devices Inc. [AMD/ATI],D34314
card6,0x738c,0xeb6bad8d73890a57,0x738c,0x0c34,Advanced Micro Devices Inc. [AMD/ATI],D34314
card7,0x738c,0x9bcde85fbd3d889b,0x738c,0x0c34,Advanced Micro Devices Inc. [AMD/ATI],D34303
*/

func detectRocmGPUs(visibleGPUs string) ([]device.Device, error) {
	args := []string{"--showid", "--showuniqueid", "--showproductname", "--csv"}

	if visibleGPUs != "" {
		gpuIds := strings.Split(visibleGPUs, ",")
		args = append(args, "-d")
		args = append(args, gpuIds...)
	}

	cmd := exec.Command("rocm-smi", args...)
	out, err := cmd.Output()
	if execError, ok := err.(*exec.Error); ok && execError.Err == exec.ErrNotFound {
		return nil, nil
	} else if err != nil {
		log.WithError(err).WithField("output", string(out)).Warnf(
			"error while executing rocm-smi to detect GPUs")
		return nil, nil
	}

	devices := []device.Device{}
	reader := csv.NewReader(strings.NewReader(string(out)))

	header, err := reader.Read()
	if err != nil {
		return nil, errors.Wrap(err, "error parsing output of rocm-smi")
	}

	expectedHeaders := []string{
		"device", "GPU ID", "Unique ID", "Card series",
		"Card model", "Card vendor", "Card SKU",
	}
	for i, h := range expectedHeaders {
		if header[i] != h {
			return nil, errors.New("bad rocm-smi csv header")
		}
	}

	for {
		record, err := reader.Read()
		switch {
		case err == io.EOF:
			return devices, nil
		case err != nil:
			return nil, errors.Wrap(err, "error parsing output of rocm-smi")
		}

		index, err := strconv.Atoi(record[0][len("card"):])
		if err != nil {
			return nil, errors.Wrap(
				err, "error parsing output of nvidia-smi; index of GPU cannot be converted to int")
		}

		brand := strings.TrimSpace(record[5])
		uuid := strings.TrimSpace(record[2])

		devices = append(devices, device.Device{ID: index, Brand: brand, UUID: uuid, Type: device.GPU})
	}
}
