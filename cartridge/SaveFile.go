package cartridge

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"
)

type SaveFile struct {
	Game       string
	Path       string
	NoOfBanks  int
	Banks      []string
	BankHashes []uint32
	LastSaved  string
}

func NewSaveFile(savesDir string, game string) *SaveFile {
	var s *SaveFile = new(SaveFile)
	s.Game = game
	s.Path = filepath.Join(savesDir, game) + ".sav"
	return s
}

func (s *SaveFile) Validate() error {
	if s.NoOfBanks != len(s.Banks) {
		return errors.New(fmt.Sprintf("No. of banks does (%d) NOT match number of actual banks (%d)", s.NoOfBanks, s.Banks))
	}

	return nil
}

//Takes a byte array, converts it to a base64 string and compresses it using ZLIB.
func (s *SaveFile) DeflateBank(bank []byte) (string, error) {
	var outBuffer bytes.Buffer
	zl := zlib.NewWriter(&outBuffer)
	_, err := zl.Write(bank)
	if err != nil {
		return "", err
	}
	zl.Close()

	compressedBankStr := base64.StdEncoding.EncodeToString(outBuffer.Bytes())
	return compressedBankStr, nil
}

//Takes a base64 string and decompresses it using ZLIB into a byte array
func (s *SaveFile) InflateBank(bankStr string) ([]byte, error) {
	compressedBank, err := base64.StdEncoding.DecodeString(bankStr)
	if err != nil {
		return nil, err
	}

	var inBuffer *bytes.Buffer = bytes.NewBuffer(compressedBank)
	zl, err := zlib.NewReader(inBuffer)
	if err != nil {
		return nil, err
	}

	var outBuffer *bytes.Buffer = bytes.NewBuffer(make([]byte, 0))
	io.Copy(outBuffer, zl)
	zl.Close()

	return outBuffer.Bytes(), nil
}

func (s *SaveFile) Load(noOfBanks int) ([][]byte, error) {
	log.Println("Loading RAM from", s.Path)
	file, err := ioutil.ReadFile(s.Path)
	if err != nil {
		return nil, err
	}

	//Now parse into struct
	var saveFile SaveFile
	err = json.Unmarshal(file, &saveFile)
	if err != nil {
		return nil, err
	}

	//ensure save file is valid
	if err := saveFile.Validate(); err != nil {
		return nil, err
	}

	s = &saveFile
	log.Println("Game was last saved:", s.LastSaved)

	if len(s.Banks) != noOfBanks {
		return nil, errors.New(fmt.Sprintln("Error: Expected", noOfBanks, "banks but found", len(s.Banks)))
	}

	s.NoOfBanks = noOfBanks

	var result [][]byte = make([][]byte, s.NoOfBanks)
	for i, bank := range s.Banks {
		log.Println("--> Loading bank", i)

		//decompress into byte array
		inflatedBank, err := s.InflateBank(bank)
		if err != nil {
			return nil, errors.New(fmt.Sprintln("Error attempting to parse and decompress bank %d (%v), save file could be corrupted!", i, err))
		}

		//check to ensure checksum is valid against what we decompressed
		hash := crc32.ChecksumIEEE(inflatedBank)
		if hash != s.BankHashes[i] {
			return nil, errors.New(fmt.Sprintln("Hash error occured, ram save file is corrupted! (inflated bank", i, " does not match hash on disk!)"))
		}

		result[i] = inflatedBank
	}

	return result, nil
}

//compresses ram banks and stores as base64 strings.
//hashes are taken each bank
//information is stored on disk in JSON format
func (s *SaveFile) Save(data [][]byte) error {
	s.NoOfBanks = len(data)
	s.Banks = make([]string, s.NoOfBanks)
	s.BankHashes = make([]uint32, s.NoOfBanks)
	s.LastSaved = fmt.Sprint(time.Now().Format(time.UnixDate))

	log.Println("Saving RAM to", s.Path)
	for i, bank := range data {
		//take crc32 hash of bank
		s.BankHashes[i] = crc32.ChecksumIEEE(bank)

		//compress
		bankStr, err := s.DeflateBank(bank)
		if err != nil {
			return errors.New(fmt.Sprintln("Error attempting to compress bank %d (%v)", i, err))
		}

		log.Printf("--> Storing bank %d (Compression ratio: %.1f%%)", i, 100.00-((float32(len(bankStr))/float32(len(bank)))*100))
		s.Banks[i] = bankStr
	}

	//serialize to JSON
	js, err := json.Marshal(&s)
	if err != nil {
		return errors.New(fmt.Sprintln("Error attempting to parse into JSON", err))
	}

	log.Println("Save file", s.Path, "size is", len(js), "bytes")

	//write to disk
	return ioutil.WriteFile(s.Path, js, 0755)
}
