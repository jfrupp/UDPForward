package controlpacker

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"
	"timehelper"
)

func TestCheckSHA512Implementation(t *testing.T) {
	testVector := []byte("abc")
	hashValue, _ := hex.DecodeString("ddaf35a193617abacc417349ae20413112e6fa4e89a97ea20a9eeee64b55d39a2" +
		"192992a274fc1a836ba3c23a3feebbd454d4423643ce80e2a9ac94fa54ca49f")
	// The rest of the hash value is omitted for brevity
	hash := sha512.New()
	hash.Write(testVector)
	sha512sum := hash.Sum(nil)
	if hex.EncodeToString(sha512sum) != hex.EncodeToString(hashValue) {
		t.Errorf("SHA512 implementation failed: \ngot %v,\nexp %v",
			hex.EncodeToString(sha512sum), hex.EncodeToString(hashValue))
	}
}

func TestAES(t *testing.T) {
	key := []byte{0x2b, 0x7e, 0x15, 0x16, 0x28, 0xae, 0xd2, 0xa6, 0xab, 0xf7, 0x15, 0x88, 0x09, 0xcf, 0x4f, 0x3c}
	iv := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f}
	plaintext := []byte{0x6b, 0xc1, 0xbe, 0xe2, 0x2e, 0x40, 0x9f, 0x96, 0xe9, 0x3d, 0x7e, 0x11, 0x73, 0x93, 0x17, 0x2a}
	ciphertext := []byte{0x76, 0x49, 0xab, 0xac, 0x81, 0x19, 0xb2, 0x46, 0xce, 0xe9, 0x8e, 0x9b, 0x12, 0xe9, 0x19, 0x7d}

	cipher1 := make([]byte, 16)
	cipher2 := make([]byte, 16)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Errorf("AES NewCipher error: %v", err)
	}
	enc := cipher.NewCBCEncrypter(block, iv)
	enc.CryptBlocks(cipher1, plaintext)
	if string(cipher1) != string(ciphertext) {
		t.Errorf("AES encryption failed: \ngot %v,\nexp %v",
			cipher1, ciphertext)
	}
	enc = cipher.NewCBCEncrypter(block, iv)
	enc.CryptBlocks(cipher2, plaintext)
	if string(cipher2) != string(ciphertext) {
		t.Errorf("AES encryption failed: \ngot %v,\nexp %v",
			cipher2, ciphertext)
	}
	decr := cipher.NewCBCDecrypter(block, iv)
	plain1 := make([]byte, 16)
	plain2 := make([]byte, 16)
	decr.CryptBlocks(plain1, ciphertext)
	if string(plain1) != string(plaintext) {
		t.Errorf("AES decryption failed: \ngot %v,\nexp %v",
			plain1, plaintext)
	}
	decr = cipher.NewCBCDecrypter(block, iv)
	decr.CryptBlocks(plain2, ciphertext)
	if string(plain2) != string(plaintext) {
		t.Errorf("AES decryption failed: \ngot %v,\nexp %v",
			plain2, plaintext)
	}
}

func getNewControlPackerForTest() *ControlPacker {
	err := os.WriteFile("hello~", []byte("Hello, Gophers01234567890!"), 0666)
	if err != nil {
		return nil
	}
	return NewControlPacker("hello~", 1, 2)
}

func TestNewControlPacker(t *testing.T) {
	cp := getNewControlPackerForTest()
	if cp == nil {
		t.Errorf("NewControlPacker returned nil")
	}
}

func TestControlPacker_Encode1(t *testing.T) {
	cp := getNewControlPackerForTest()
	if cp == nil {
		t.Errorf("NewControlPacker returned nil")
		return
	}
	var d DecodedControlPacket
	d.Id = 0
	d.Flow = 5
	e1a, _ := cp.EncodeControlPacket(&d, true, 1625079600, false)
	e2a, _ := cp.EncodeControlPacket(&d, false, 1625079600, false)
	e1b, _ := cp.EncodeControlPacket(&d, true, 1625079600, false)
	e2b, _ := cp.EncodeControlPacket(&d, false, 1625079600, false)
	if e1a.Len != 10 {
		t.Error("Wrong length of encoded data, must be 10")
	}
	if string(e1a.Data[:e1a.Len]) != string(e1b.Data[:e1b.Len]) {
		t.Errorf("EncodeControlPacket 1a and 1b differ:\n%v\n%v",
			e1a.Data[:e1a.Len], e1b.Data[:e1b.Len])
	}
	if string(e2a.Data[:e2a.Len]) != string(e2b.Data[:e2b.Len]) {
		t.Errorf("EncodeControlPacket 2a and 2b differ:\n%v\n%v",
			e2a.Data[:e2a.Len], e2b.Data[:e2b.Len])
	}
	if string(e1a.Data[:e1a.Len]) == string(e2a.Data[:e2a.Len]) {
		t.Errorf("EncodeControlPacket 1a and 1b are the same:\n%v\n%v",
			e1a.Data[:e1a.Len], e2a.Data[:e2a.Len])
	}

}

func TestControlPacker_EncodeDecode(t *testing.T) {
	cp := getNewControlPackerForTest()
	if cp == nil {
		t.Errorf("NewControlPacker returned nil")
		return
	}
	var d DecodedControlPacket
	d.Id = 0
	d.Flow = 8
	e1, _ := cp.EncodeControlPacket(&d, true, 0, false)
	_, err1 := cp.DecodeControlPacket(&e1, true, 0)
	e2, _ := cp.EncodeControlPacket(&d, true, 0, false)
	d2, err2 := cp.DecodeControlPacket(&e2, true, 0)
	_, err3 := cp.DecodeControlPacket(&e2, false, 0)

	if err1 != nil {
		t.Error(err1)
	}
	if err2 != nil {
		t.Error(err2)
	}
	if err3 == nil {
		t.Error("Decoding with wrong key successful")
	}
	if timehelper.AbsDiffUnixTime(time.Now().Unix(), d2.UnixTime) > 2 {
		t.Error("Time not properly encoded")
	}
	if d2.Id != 0 || d2.Flow != 8 {
		t.Error("Bad flow labels")
	}

}

func TestControlPacker_EncodeDecodeExtension1(t *testing.T) {
	cp := getNewControlPackerForTest()
	if cp == nil {
		t.Errorf("NewControlPacker returned nil")
		return
	}

	var d DecodedControlPacket
	d.Id = 0
	d.Flow = 8
	ext := [...]byte{3, 0, 1, 0, 0, 0}
	d.OptExtension = ext
	e, err1 := cp.EncodeControlPacket(&d, true, 0, false)
	d, err2 := cp.DecodeControlPacket(&e, true, 0)
	if err1 != nil || err2 != nil {
		t.Errorf("Encoding or Decoding has failed")
	}
	if e.Len != 13 {
		t.Errorf("Wrong length of encoded packet, should be 13")
	}
	if e.Data[10] != 3 || e.Data[12] != 1 {
		t.Errorf("Wrong content of OptExtension in encoded packet")
	}
	if d.OptExtension[0] != 3 || d.OptExtension[2] != 1 {
		t.Errorf("Wrong content of OptExtension in decoded packet")
	}
}

func TestControlPacker_EncodeDecodeExtension2(t *testing.T) {
	cp := getNewControlPackerForTest()
	if cp == nil {
		t.Errorf("NewControlPacker returned nil")
		return
	}

	var d DecodedControlPacket
	d.Id = 0
	d.Flow = 8
	ext := [...]byte{0, 0, 0, 0, 0, 20}
	d.OptExtension = ext
	e, err1 := cp.EncodeControlPacket(&d, true, 0, false)
	d, err2 := cp.DecodeControlPacket(&e, true, 0)
	if err1 != nil || err2 != nil {
		t.Errorf("Encoding or Decoding has failed")
	}
	if e.Len != 16 {
		t.Errorf("Wrong length of encoded packet, should be 13")
	}
	if e.Data[15] != 20 {
		t.Errorf("Wrong content of OptExtension in encoded packet")
	}
	if d.OptExtension[5] != 20 {
		t.Errorf("Wrong content of OptExtension in decoded packet")
	}
}

func TestControlPacker_EncodeDecodeBadTime(t *testing.T) {
	cp := getNewControlPackerForTest()
	if cp == nil {
		t.Errorf("NewControlPacker returned nil")
		return
	}

	var d DecodedControlPacket
	d.Id = 0
	d.Flow = 8
	e, err1 := cp.EncodeControlPacket(&d, true, 1000, false)
	_, err2 := cp.DecodeControlPacket(&e, true, 1004)
	if err1 != nil {
		t.Errorf("Encoding has failed")
	}
	if err2 == nil {
		t.Errorf("Decoding has NOT failed, but it should have failed!")
	}
}

func TestControlPacker_EncodeDecodeBadMAC(t *testing.T) {
	cp := getNewControlPackerForTest()
	if cp == nil {
		t.Errorf("NewControlPacker returned nil")
		return
	}

	var d DecodedControlPacket
	d.Id = 0
	d.Flow = 8
	e, err1 := cp.EncodeControlPacket(&d, true, 1000, false)
	fmt.Println(e.GetMACFromEncodeControlPacket())
	e.Data[4] = ^e.Data[4]
	_, err2 := cp.DecodeControlPacket(&e, true, 1000)
	if err1 != nil {
		t.Errorf("Encoding has failed")
	}
	if err2 == nil {
		t.Errorf("Decoding has NOT failed, but it should have failed!")
	}
}

func TestEncode10000ControlPackets(t *testing.T) {
	cp := getNewControlPackerForTest()
	var d DecodedControlPacket
	d.Id = 0
	d.Flow = 8
	for i := 0; i < 1000000; i++ {
		cp.EncodeControlPacket(&d, true, 1000, false)
	}
}

func TestConfigID(t *testing.T) {
	id := getNewControlPackerForTest().ConfigID
	fmt.Println(id)
	if id != 16370355 {
		t.Error("Config ID does not match file hello~")
	}
}
