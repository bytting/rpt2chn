package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ParseAquisitionDate(line string) ([]byte, []byte, error) {

	items := strings.Split(line, " ")
	dt := items[2]
	tm := items[3]

	monthNames := [...]string{"JAN", "FEB", "MAR", "APR", "MAY", "JUN", "JUL", "AUG", "SEP", "OCT", "NOV", "DEC"}

	var month int
	fmt.Sscanf(dt[3:5], "%2d", &month)

	b := make([]byte, 0, 12)
	b = append(b, []byte(dt[:2])...)
	b = append(b, []byte(monthNames[month-1])...)
	b = append(b, []byte(dt[8:10])...)

	b = append(b, []byte("1")...)

	b = append(b, []byte(tm[:2])...)
	b = append(b, []byte(tm[3:5])...)

	s := make([]byte, 0, 2)
	s = append(s, []byte(tm[6:8])...)

	return b, s, nil
}

func ParseTrailingFloat(line string) (float64, error) {

	items := strings.Split(line, " ")
	return strconv.ParseFloat(items[len(items)-1], 32)
}

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
		return
	}

	if _, err := os.Stat(inFile); err != nil {
		fmt.Fprintln(os.Stderr, "input file does not exist")
		return
	}

	fin, err := os.Open(inFile)
	if err != nil {
		panic(err)
	}
	defer fin.Close()

	scanner := bufio.NewScanner(fin)

	scanner.Scan()
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	dateTimeInfo, seconds, err := ParseAquisitionDate(scanner.Text())
	if err != nil {
		panic(err)
	}

	scanner.Scan()
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	livetime, err := ParseTrailingFloat(scanner.Text())
	if err != nil {
		panic(err)
	}

	scanner.Scan()
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	realtime, err := ParseTrailingFloat(scanner.Text())
	if err != nil {
		panic(err)
	}

	var fileBuffer bytes.Buffer
	binary.Write(&fileBuffer, binary.LittleEndian, int16(-1))
	binary.Write(&fileBuffer, binary.LittleEndian, int16(1))
	binary.Write(&fileBuffer, binary.LittleEndian, int16(1))
	fileBuffer.Write(seconds)
	binary.Write(&fileBuffer, binary.LittleEndian, int32(realtime*50.0))
	binary.Write(&fileBuffer, binary.LittleEndian, int32(livetime*50.0))
	fileBuffer.Write(dateTimeInfo)
	binary.Write(&fileBuffer, binary.LittleEndian, int16(0))

	var channelBuffer bytes.Buffer
	nchans := 0
	for scanner.Scan() {
		items := strings.Fields(scanner.Text())
		for _, v := range items[1:] {
			ch, err := strconv.Atoi(v)
			if err != nil {
				panic(err)
			}
			binary.Write(&channelBuffer, binary.LittleEndian, int32(ch))
			nchans += 1
		}
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	binary.Write(&fileBuffer, binary.LittleEndian, int16(nchans))
	fileBuffer.Write(channelBuffer.Bytes())

	fout, err := os.Create(outFile)
	if err != nil {
		panic(err)
	}
	defer fout.Close()

	fout.Write(fileBuffer.Bytes())
}
