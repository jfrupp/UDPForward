package controlpacker

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha512"
	"errors"
	"os"
	"time"
	"timehelper"
)

var ErrorWrongParameter = errors.New("wrong parameter")
var ErrorTimeExpired = errors.New("time expired")
var WrongMAC = errors.New("wrong MAC")

type EncodedControlPacket struct {
	Data [16]byte
	// Len may be between 10 and 16 bytes depending on the content.
	Len int
}

func NewEncodedControlPacket() *EncodedControlPacket {
	return &EncodedControlPacket{}
}

type DecodedControlPacket struct {
	Id       uint8
	Flow     uint8
	UnixTime int64
	// MAC [4]byte is already stripped
	OptExtension [6]byte
}

func NewDecodedControlPacket() *DecodedControlPacket {
	return &DecodedControlPacket{}
}

type ControlPacker struct {
	cipherInsideToOutside *cipher.Block
	cipherOutsideToInside *cipher.Block
	iVInsideToOutside     []byte
	iVOutsideToInside     []byte
	max_unix_time_diff    int64
	ConfigID              uint32
}

func (e *EncodedControlPacket) AsSlice() []byte {
	s := e.Data[:]
	return s[0:e.Len]
}

// nil error is returned only if config file cannot be read, this is the only fatal error
func NewControlPacker(config_file_name string, version uint8, max_unix_time_diff int64) *ControlPacker {

	var cp ControlPacker
	cp.max_unix_time_diff = max_unix_time_diff
	fileContentForHash, err := os.ReadFile(config_file_name)
	if err != nil {
		return nil
	}
	hash := sha512.New()
	hash.Write(fileContentForHash)
	hash.Write([]byte{byte(version)})
	hash.Write([]byte("ControlPackerConfigFileForHash3.14159265359"))
	sha512sum := hash.Sum(nil)
	cipherInsideToOutside, _ := aes.NewCipher(sha512sum[0:16])
	cipherOutsideToInside, _ := aes.NewCipher(sha512sum[16:32])
	cp.cipherInsideToOutside = &cipherInsideToOutside
	cp.cipherOutsideToInside = &cipherOutsideToInside
	cp.iVInsideToOutside = make([]byte, 16)
	cp.iVOutsideToInside = make([]byte, 16)
	copy(cp.iVInsideToOutside, sha512sum[32:48])
	copy(cp.iVOutsideToInside, sha512sum[48:64])
	hash = sha512.New()
	b := make([]byte, 16)
	copy(b, cp.iVOutsideToInside)
	// Remove some information from the block
	for i := 0; i < 16; i++ {
		b[i] = b[i] | byte(i)
	}
	hash.Write(b)
	//Rehash the block
	sha512sum = hash.Sum(nil)
	// Give out only limited information
	cp.ConfigID = uint32(sha512sum[0]) + uint32(sha512sum[1])<<8 + uint32(sha512sum[2])<<16
	return &cp
}

// plainblock: 16 bytes input block (required by AES CBC mode)
// bIsInsideToOutside: true for inside to outside direction, false for outside to inside direction
func (cp *ControlPacker) GetMAC(plainblock []byte, bIsInsideToOutside bool) [4]byte {
	var mac [4]byte
	var enc cipher.BlockMode
	cipherblock := make([]byte, 16)

	if bIsInsideToOutside {
		enc = cipher.NewCBCEncrypter(*cp.cipherInsideToOutside, cp.iVInsideToOutside)
	} else {
		enc = cipher.NewCBCEncrypter(*cp.cipherOutsideToInside, cp.iVOutsideToInside)
	}
	enc.CryptBlocks(cipherblock, plainblock)
	copy(mac[:], cipherblock[:4])
	return mac
}

func (cp *ControlPacker) EncodeControlPacket(d *DecodedControlPacket,
	bIsInsideToOutside bool, unixTime int64, bKeepPacketTime bool) (EncodedControlPacket, error) {
	var e EncodedControlPacket

	e.Len = 0
	e.Data[0] = d.Id
	e.Data[1] = d.Flow
	e.Data[2] = 0 // Placeholder for MAC
	e.Data[3] = 0 // .
	e.Data[4] = 0 // .
	e.Data[5] = 0 // .
	var ut uint32
	if !bKeepPacketTime {
		if unixTime <= 0 {
			unixTime = time.Now().Unix()
		}
		ut = timehelper.UnixTimeToU32(uint64(unixTime))
	} else {
		ut = timehelper.UnixTimeToU32(uint64(d.UnixTime))
	}
	e.Data[6] = byte(ut & 0xFF) // Timestamp as little endian u32
	e.Data[7] = byte((ut >> 8) & 0xFF)
	e.Data[8] = byte((ut >> 16) & 0xFF)
	e.Data[9] = byte((ut >> 24) & 0xFF)
	e.Len = 10
	// Optional extension
	for i := 0; i < 6; i++ {
		e.Data[10+i] = d.OptExtension[i]
	}
	// For the optional extension, determine location of last value !=0
	count := 6
	for ; count > 0; count-- {
		if d.OptExtension[count-1] != 0 {
			break
		}
	}
	e.Len += count
	// Now compute and fill in the MAC
	mac := cp.GetMAC(e.Data[:], bIsInsideToOutside)
	e.Data[2] = mac[0]
	e.Data[3] = mac[1]
	e.Data[4] = mac[2]
	e.Data[5] = mac[3]

	return e, nil
}

func (cp *ControlPacker) DecodeControlPacket(e *EncodedControlPacket,
	bIsInsideToOutside bool, unixTime int64) (DecodedControlPacket, error) {
	var d DecodedControlPacket
	var ut uint32

	if e.Len < 10 || e.Len > 16 {
		return d, ErrorWrongParameter
	}
	d.Id = e.Data[0]
	d.Flow = e.Data[1]
	ut = uint32(e.Data[6]) | (uint32(e.Data[7]) << 8) |
		(uint32(e.Data[8]) << 16) | (uint32(e.Data[9]) << 24)
	if unixTime <= 0 {
		unixTime = time.Now().Unix()
	}
	d.UnixTime = timehelper.U32ToUnixTime(ut, unixTime)
	if timehelper.AbsDiffUnixTime(unixTime, d.UnixTime) > cp.max_unix_time_diff {
		return d, ErrorTimeExpired
	}
	for i := 0; i < e.Len-10; i++ {
		d.OptExtension[i] = e.Data[10+i]
	}
	// sanitze optional data junk
	for i := e.Len; i < 16; i++ {
		e.Data[i] = 0
	}
	// extract MAC
	var macEncodedPacket [4]byte
	copy(macEncodedPacket[:], e.Data[2:6])
	// zero MAC in packet
	e.Data[2] = 0
	e.Data[3] = 0
	e.Data[4] = 0
	e.Data[5] = 0
	calculatedMAC := cp.GetMAC(e.Data[:], bIsInsideToOutside)
	if macEncodedPacket != calculatedMAC {
		return d, WrongMAC
	}
	return d, nil
}

// Get the MAC from an Encoded Control Packet
func (e *EncodedControlPacket) GetMACFromEncodeControlPacket() []byte {
	s := e.Data[:]
	return s[2:6]
}
