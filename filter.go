package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"filter/cli"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func main() {
	var params = cli.Params{}
	params.Load()
	params.Print()

	Run(params)
}

func Run(params cli.Params) {
	//prepare dest folder
	prepareDst(params)

	//get file list by mask
	files, err := filepath.Glob(filepath.Join(params.InputDir, params.Mask))
	if err != nil {
		log.Fatal(err)
	}

	gwg := new(sync.WaitGroup)

	for _, filePath := range files {
		//get FileInfo
		file, err := os.Stat(filePath)
		if err != nil {
			log.Fatal(err)
		}

		if !file.IsDir() {
			if strings.HasSuffix(file.Name(), "zip") {
				log.Printf("zip file: %v", file.Name())
				gwg.Add(1)
				go introspectZip(file.Name(), params, gwg)
			} else {
				log.Printf("log file: %v", file.Name())
				gwg.Add(1)
				go processLog(file.Name(), params, gwg)
			}
		}
	}

	//wait for goroutines
	gwg.Wait()

	//delete empty files
	cleanup(params)
}

func prepareDst(params cli.Params) {
	path := filepath.Join(params.InputDir, params.OutputDir)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		if params.Force {
			if err := os.RemoveAll(path); err != nil {
				log.Fatalf("can't clear directory %v due to %v", path, err)
			}
		} else {
			if params.X {
				log.Printf("WARN: directory %v already exist, but -f option is missed", path)
			} else {
				log.Fatalf("directory %v already exist", path)
			}
		}
	}
	if params.X {
		return
	}
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		log.Fatal(err)
	}
}

func introspectZip(name string, params cli.Params, gwg *sync.WaitGroup) {
	defer gwg.Done()

	log.Printf("start introspecting %v", name)
	defer log.Printf("finish introspecting %v", name)

	//open zip
	zf, err := zip.OpenReader(filepath.Join(params.InputDir, name))
	if err != nil {
		log.Printf("can't open input file %v due to error %v", name, err)
		return
	}
	defer func() {
		if err := zf.Close(); err != nil {
			log.Fatalf("can't close file %v due to %v", name, err)
		}
	}()

	zwg := new(sync.WaitGroup)

	for _, file := range zf.File {
		zwg.Add(1)
		go processArchived(file, params, zwg)
	}

	zwg.Wait()
}

func processArchived(file *zip.File, params cli.Params, zwg *sync.WaitGroup) {
	defer zwg.Done()

	log.Printf("start processing %v", file.Name)
	defer log.Printf("finish processing %v", file.Name)

	//open in file
	inFile, err := file.Open()
	if err != nil {
		log.Printf("can't open input file %v due to error %v", file.Name, err)
		return
	}
	defer func() {
		if err := inFile.Close(); err != nil {
			log.Fatalf("can't close file %v due to %v", file.Name, err)
		}
	}()

	inReader := bufio.NewReader(inFile)

	process(inReader, params, file.Name)
}

func processLog(fileName string, params cli.Params, gwg *sync.WaitGroup) {
	defer gwg.Done()

	log.Printf("start processing %v", fileName)
	defer log.Printf("finish processing %v", fileName)

	//open in file
	inFile, err := os.Open(filepath.Join(params.InputDir, fileName))
	if err != nil {
		log.Printf("can't open input file %v due to error %v", fileName, err)
		return
	}
	defer func() {
		if err := inFile.Close(); err != nil {
			log.Fatalf("can't close file %v due to %v", fileName, err)
		}
	}()

	inReader := bufio.NewReader(inFile)

	process(inReader, params, fileName)
}

func process(in *bufio.Reader, params cli.Params, name string) {
	if params.X {
		return
	}

	//open out file
	outName := filepath.Join(params.InputDir, params.OutputDir, name)
	outFile, err := os.Create(outName)
	if err != nil {
		log.Printf("can't open output file %v due to error %v", name, err)
		return
	}
	defer func() {
		if err := outFile.Close(); err != nil {
			log.Fatalf("can't close file %v due to %v", name, err)
		}
	}()
	outWriter := bufio.NewWriter(outFile)

	//prepare scanner
	buffer := make([]byte, params.BufferSize)
	scan := bufio.NewScanner(in)
	scan.Buffer(buffer, params.BufferSize)
	scan.Split(getSplitFunc(params))

	//process file
	for scan.Scan() {
		token := scan.Bytes()
		for _, search := range params.SearchStrings {
			if bytes.Contains(token, search) == !params.Inverse {
				writeChunk(outWriter, params.Delimiter, name)
				writeChunk(outWriter, token, name)
			}
		}
		for _, search := range params.RegexpStrings {
			if search.Match(token) == !params.Inverse {
				writeChunk(outWriter, params.Delimiter, name)
				writeChunk(outWriter, token, name)
			}
		}
	}

	//process read errors
	if err := scan.Err(); err != nil {
		log.Printf("can't read file %v due to %v", name, err)
	}
}

func writeChunk(out *bufio.Writer, data []byte, name string) {
	if _, err := (*out).Write(data); err != nil {
		log.Printf("can't write file %v due to %v", name, err)
	}
}

func getSplitFunc(params cli.Params) bufio.SplitFunc {
	if params.Line {
		return bufio.ScanLines
	} else {
		return splitAt(params.Delimiter)
	}
}

func splitAt(substring []byte) func(data []byte, atEOF bool) (advance int, token []byte, err error) {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		// Return nothing if at end of file and no data passed
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		if i := bytes.Index(data, substring); i >= 0 {
			return i + len(substring), data[0:i], nil
		}

		// If we're at EOF, we have a final, non-terminated line. Return it.
		if atEOF {
			return len(data), data, nil
		}

		// Request more data.
		return 0, nil, nil
	}
}

func cleanup(params cli.Params) {
	path := filepath.Join(params.InputDir, params.OutputDir)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Printf("can't list directory %v due to %v", path, err)
		return
	}
	for _, v := range files {
		if v.Size() == 0 {
			fn := filepath.Join(path, v.Name())
			err := os.Remove(fn)
			if err != nil {
				log.Printf("can't delete empty file %v due to %v", fn, err)
			}
		}
	}
}
