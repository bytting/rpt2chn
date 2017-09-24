//  rpt2chn - Convert RPT spectrum reports to CHN spectrum format
//  Copyright (C) 2017  NRPA
//
//  This program is free software: you can redistribute it and/or modify
//  it under the terms of the GNU General Public License as published by
//  the Free Software Foundation, either version 3 of the License, or
//  (at your option) any later version.
//
//  This program is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU General Public License for more details.
//
//  You should have received a copy of the GNU General Public License
//  along with this program.  If not, see <http://www.gnu.org/licenses/>.
//
//  Authors: Dag Robole,

package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

var (
	inFile  string
	outFile string
)

func init() {

	flag.StringVar(&inFile, "if", "", "RPT file to read from")
	flag.StringVar(&outFile, "of", "", "CHN file to write to")
}

func main() {

	flag.Parse()

	if flag.NFlag() != 2 || len(inFile) == 0 || len(outFile) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	_, err := os.Stat(inFile)
	dieIf(err)

	fin, err := os.Open(inFile)
	dieIf(err)
	defer fin.Close()

	scanner := bufio.NewScanner(fin)

	scanner.Scan()
	dieIf(scanner.Err())
	hoursMinutes, seconds, err := parseAquisitionDate(scanner.Text())
	dieIf(err)

	scanner.Scan()
	dieIf(scanner.Err())
	livetime, err := parseTrailingFloat(scanner.Text())
	dieIf(err)

	scanner.Scan()
	dieIf(scanner.Err())
	realtime, err := parseTrailingFloat(scanner.Text())
	dieIf(err)

	channelBuffer := new(bytes.Buffer)
	numChannels := int16(0)
	for scanner.Scan() {
		err := absorbChannels(scanner.Text(), channelBuffer, &numChannels)
		dieIf(err)
	}
	dieIf(scanner.Err())

	if !isPowerOfTwo(numChannels) {
		dieIf(errors.New("number of channels is not a power of two"))
	}

	fout, err := os.Create(outFile)
	dieIf(err)
	defer fout.Close()

	binary.Write(fout, binary.LittleEndian, int16(-1))
	binary.Write(fout, binary.LittleEndian, int16(1))
	binary.Write(fout, binary.LittleEndian, int16(1))
	fout.Write(seconds)
	binary.Write(fout, binary.LittleEndian, int32(realtime*50.0))
	binary.Write(fout, binary.LittleEndian, int32(livetime*50.0))
	fout.Write(hoursMinutes)
	binary.Write(fout, binary.LittleEndian, int16(0))
	binary.Write(fout, binary.LittleEndian, numChannels)
	fout.Write(channelBuffer.Bytes())
}

func dieIf(err error) {

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func isPowerOfTwo(n int16) bool {

	return (n != 0) && ((n & (n - 1)) == 0)
}

func parseAquisitionDate(line string) ([]byte, []byte, error) {

	items := strings.Split(strings.Trim(line, " \t\n"), " ")
	if len(items) < 4 {
		return nil, nil, errors.New("ParseAquisitionDate: missing items")
	}

	dt, tm := items[2], items[3]
	if len(dt) != 10 || len(tm) != 8 {
		return nil, nil, errors.New("ParseAquisitionDate: date/time format invalid")
	}

	var month int
	fmt.Sscanf(dt[3:5], "%2d", &month)
	if month < 1 || month > 12 {
		return nil, nil, errors.New("ParseAquisitionDate: month out of range")
	}

	monthNames := [...]string{"JAN", "FEB", "MAR", "APR", "MAY", "JUN", "JUL", "AUG", "SEP", "OCT", "NOV", "DEC"}

	dateParts := [][]byte{[]byte(dt[:2]), []byte(monthNames[month-1]), []byte(dt[8:10]), []byte("1"), []byte(tm[:2]), []byte(tm[3:5])}

	return bytes.Join(dateParts, []byte("")), []byte(tm[6:8]), nil
}

func parseTrailingFloat(line string) (float64, error) {

	items := strings.Split(strings.Trim(line, " \t\n"), " ")
	if len(items) == 0 {
		return 0.0, errors.New("ParseTrailingFloat: no valid decimal found")
	}
	return strconv.ParseFloat(items[len(items)-1], 32)
}

func absorbChannels(line string, w io.Writer, nchans *int16) error {

	items := strings.Fields(strings.Trim(line, " \t\n"))
	if len(items) < 2 {
		return nil
	}

	for _, v := range items[1:] {
		ch, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		binary.Write(w, binary.LittleEndian, uint32(ch))
		*nchans += 1
	}

	return nil
}
