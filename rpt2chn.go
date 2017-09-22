package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
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
	flag.StringVar(&outFile, "of", "", "New CHN file to write to")
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

	dateTimeInfo, seconds, err := ParseAquisitionDate(scanner.Text())
	dieIf(err)

	scanner.Scan()
	dieIf(scanner.Err())

	livetime, err := ParseTrailingFloat(scanner.Text())
	dieIf(err)

	scanner.Scan()
	dieIf(scanner.Err())

	realtime, err := ParseTrailingFloat(scanner.Text())
	dieIf(err)

	fileBuffer := new(bytes.Buffer)
	binary.Write(fileBuffer, binary.LittleEndian, int16(-1))
	binary.Write(fileBuffer, binary.LittleEndian, int16(1))
	binary.Write(fileBuffer, binary.LittleEndian, int16(1))
	fileBuffer.Write(seconds)
	binary.Write(fileBuffer, binary.LittleEndian, int32(realtime*50.0))
	binary.Write(fileBuffer, binary.LittleEndian, int32(livetime*50.0))
	fileBuffer.Write(dateTimeInfo)
	binary.Write(fileBuffer, binary.LittleEndian, int16(0))

	channelBuffer := new(bytes.Buffer)
	nchans := uint16(0)
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " \t\n")
		if len(line) == 0 {
			continue
		}

		items := strings.Fields(line)
		for _, v := range items[1:] {
			ch, err := strconv.Atoi(v)
			dieIf(err)
			binary.Write(channelBuffer, binary.LittleEndian, int32(ch))
			nchans += 1
		}
	}

	dieIf(scanner.Err())

	if !isPowerOfTwo(nchans) {
		dieIf(errors.New("number of channels is not a power of two"))
	}

	binary.Write(fileBuffer, binary.LittleEndian, int16(nchans))
	fileBuffer.Write(channelBuffer.Bytes())

	fout, err := os.Create(outFile)
	dieIf(err)
	defer fout.Close()

	fout.Write(fileBuffer.Bytes())
}

func dieIf(err error) {

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func isPowerOfTwo(n uint16) bool {

	return (n != 0) && ((n & (n - 1)) == 0)
}

func ParseAquisitionDate(line string) ([]byte, []byte, error) {

	line = strings.Trim(line, " \t\n")
	items := strings.Split(line, " ")
	if len(items) < 4 {
		return nil, nil, errors.New("ParseAquisitionDate: missing items")
	}

	dt := items[2]
	tm := items[3]

	monthNames := [...]string{"JAN", "FEB", "MAR", "APR", "MAY", "JUN", "JUL", "AUG", "SEP", "OCT", "NOV", "DEC"}

	var month int
	fmt.Sscanf(dt[3:5], "%2d", &month)
	if month < 1 || month > 12 {
		return nil, nil, errors.New("ParseAquisitionDate: month out of range")
	}

	parts := [][]byte{
		[]byte(dt[:2]), []byte(monthNames[month-1]), []byte(dt[8:10]), []byte("1"), []byte(tm[:2]), []byte(tm[3:5]),
	}

	b := make([]byte, 0, 12)
	for _, r := range parts {
		b = append(b, r...)
	}

	s := make([]byte, 0, 2)
	s = append(s, []byte(tm[6:8])...)

	return b, s, nil
}

func ParseTrailingFloat(line string) (float64, error) {

	line = strings.Trim(line, " \t\n")
	items := strings.Split(line, " ")
	if len(items) == 0 {
		return 0.0, errors.New("ParseTrailingFloat: no valid decimal found")
	}
	return strconv.ParseFloat(items[len(items)-1], 32)
}
