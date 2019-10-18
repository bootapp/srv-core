package oauth

import (
	"bytes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/hex"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)
func padding(src []byte,blocksize int) []byte {
	padnum:=blocksize-len(src)%blocksize
	pad:=bytes.Repeat([]byte{byte(padnum)},padnum)
	return append(src,pad...)
}

func unpadding(src []byte) []byte {
	n:=len(src)
	unpadnum:=int(src[n-1])
	return src[:n-unpadnum]
}
func (s *UserPassOAuthServer) AESEncrypt(src string) string {
	code := []byte(src)
	code = padding(code, s.aesCipher.BlockSize())
	aesEncrypt := cipher.NewCBCEncrypter(s.aesCipher, s.aesKey)
	aesEncrypt.CryptBlocks(code, code)
	return hex.EncodeToString(code)
}

func (s *UserPassOAuthServer) AESDecrypt(src string) string {
	code, err := hex.DecodeString(src)
	if err != nil {
		return ""
	}
	aesDecrypt := cipher.NewCBCDecrypter(s.aesCipher, s.aesKey)
	aesDecrypt.CryptBlocks(code, code)
	res := unpadding(code)
	return string(res)
}


func (s *UserPassOAuthServer) RS256Sign(content string) (string, error) {
	if !s.Hash.Available() {
		return "", status.Error(codes.Internal, "hash unavailable")
	}
	hasher := s.Hash.New()
	hasher.Write([]byte(content))
	if sigBytes, err := rsa.SignPSS(rand.Reader, s.privKey, s.Hash, hasher.Sum(nil), nil); err == nil {
		return EncodeSegment(sigBytes), nil
	} else {
		return "", err
	}
}
func (s *UserPassOAuthServer) RS256Verify(content, sig string) bool {
	hasher := s.Hash.New()
	s.Hash.New()
	hasher.Write([]byte(content))
	rawSig, err := DecodeSegment(sig)
	if err != nil {
		return false
	}
	if err := rsa.VerifyPSS(s.pubKey, s.Hash, hasher.Sum(nil), rawSig, nil); err == nil {
		return true
	} else {
		return false
	}
}
func EncodeSegment(seg []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(seg), "=")
}
func DecodeSegment(seg string) ([]byte, error) {
	if l := len(seg) % 4; l > 0 {
		seg += strings.Repeat("=", 4-l)
	}

	return base64.URLEncoding.DecodeString(seg)
}