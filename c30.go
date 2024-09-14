package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

var (
	decodeFlag = flag.Bool("d", false, "Decode mode")
	helpFlag   = flag.Bool("h", false, "Show help")
	widthFlag  = flag.Int("w", 0, "Number of encoded characters per line (0 for no wrapping)")
)

const bufferSize = 1024 * 1024 // 1MB buffer

func usage() {
	fmt.Fprintf(os.Stderr, "Encode binary data to German uppercase letters and back.\n\n")
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] < infile > outfile\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	encodeMap, decodeMap := createMaps()

	reader := bufio.NewReaderSize(os.Stdin, bufferSize)
	writer := bufio.NewWriterSize(os.Stdout, bufferSize)

	start := time.Now()
	var err error
	if *decodeFlag {
		err = decode(reader, writer, decodeMap)
	} else {
		err = encode(reader, writer, encodeMap, *widthFlag)
	}
	duration := time.Since(start)

	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}

	if err := writer.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "\nError flushing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "\nOperation completed in %v\n", duration)
}

func createMaps() (map[byte]rune, map[rune]byte) {
	encodeMap := make(map[byte]rune)
	decodeMap := make(map[rune]byte)

	// Code30: A-Z, ÄÖÜẞ (30 characters)
	code30 := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÜẞ")

	for i := 0; i < 30; i++ {
		encodeMap[byte(i)] = code30[i]
		decodeMap[code30[i]] = byte(i)
	}

	return encodeMap, decodeMap
}

func encode(reader *bufio.Reader, writer *bufio.Writer, encodeMap map[byte]rune, width int) error {
	totalBytes := 0
	lineBuffer := make([]rune, 0, width)
	base := 30

	for {
		b, err := reader.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}

		// Split the byte into two parts: division and remainder for mapping
		div := b / byte(base) // First part
		rem := b % byte(base) // Second part

		// Encode both parts using the map
		lineBuffer = append(lineBuffer, encodeMap[rem], encodeMap[div])

		if width > 0 && len(lineBuffer) >= width {
			if _, err := writer.WriteString(string(lineBuffer) + "\r\n"); err != nil {
				return fmt.Errorf("error writing output: %w", err)
			}
			lineBuffer = lineBuffer[:0]
		}

		totalBytes++
		if totalBytes%bufferSize == 0 {
			fmt.Fprintf(os.Stderr, "\rProcessed: %d MB", totalBytes/1024/1024)
		}
	}

	// Write any remaining data
	if len(lineBuffer) > 0 {
		if _, err := writer.WriteString(string(lineBuffer)); err != nil {
			return fmt.Errorf("error writing final output: %w", err)
		}
	}

	fmt.Fprint(os.Stderr, "\n")
	return nil
}

func decode(reader *bufio.Reader, writer *bufio.Writer, decodeMap map[rune]byte) error {
	totalBytes := 0

	for {
		// Read two runes at a time to decode the original byte
		remRune, _, err := reader.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}

		// Ignore line breaks or other unexpected characters
		if remRune == '\r' || remRune == '\n' {
			continue // Skip line breaks
		}

		divRune, _, err := reader.ReadRune()
		if err == io.EOF {
			return fmt.Errorf("unexpected EOF: input length is not even")
		}
		if err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}

		// Ignore line breaks or other unexpected characters
		if divRune == '\r' || divRune == '\n' {
			continue // Skip line breaks
		}

		// Map the runes back to bytes
		rem, remOk := decodeMap[remRune]
		div, divOk := decodeMap[divRune]

		if !remOk || !divOk {
			return fmt.Errorf("invalid character in input")
		}

		// Reconstruct the original byte
		base := byte(30)
		originalByte := div*base + rem

		if err := writer.WriteByte(originalByte); err != nil {
			return fmt.Errorf("error writing output: %w", err)
		}

		totalBytes++
		if totalBytes%bufferSize == 0 {
			fmt.Fprintf(os.Stderr, "\rProcessed: %d MB", totalBytes/1024/1024)
		}
	}

	fmt.Fprint(os.Stderr, "\n")
	return nil
}
