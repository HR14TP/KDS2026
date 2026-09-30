package helper

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"
	"unicode/utf8"
)

func StringToInt(str string) int {
	i, err := strconv.Atoi(str)
	if err != nil {
		i = -1
	}
	return i
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func StringArrayToIntArray(strarray string) []int {
	var ints []int
	err := json.Unmarshal([]byte(strarray), &ints)
	if err != nil {
		log.Fatal(err)
	}
	return ints
}

func IntToString(str int) string {
	return strconv.Itoa(str)
}

func IntToDoubleString(str int) string {
	if str < 10 {
		return "0" + strconv.Itoa(str)
	}
	return strconv.Itoa(str)
}

func FloatToString(fl float64) string {
	return fmt.Sprintf("%.2f", fl)
}

func StringToFloat(str string) float64 {
	var def float64
	def = 0.00
	if len(str) > 0 {
		i, err := strconv.ParseFloat(str, 32)
		if err != nil {
			log.Print("EXCEPTION IN StringToFloat")
			log.Print(str)
			i = 0.00
		}
		def = i
	}
	return def
}

func StringToIntBetter(str string) int {
	i, err := strconv.Atoi(str)
	if err != nil {
		i = -1
	}
	return i
}
func StringToIntKDS(str string) int {
	i, err := strconv.Atoi(str)
	if err != nil {
		i = 1
	}
	return i
}
func ToNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func ToNullInt64(s string) sql.NullInt64 {
	i, err := strconv.Atoi(s)
	return sql.NullInt64{Int64: int64(i), Valid: err == nil}
}

func ErrorHandler(err error) {
	if err != nil {
		fmt.Println(err.Error())
	}
}

func UtilgetTimeSQL() string {
	t := time.Now()
	ts := t.Format("2006-01-02 15:04:05")
	return ts
}

func GetMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func UtilStringToFloat64(s string) float64 {
	if converted, err := strconv.ParseFloat(s, 64); err == nil {
		return converted
	} else {
		return 1
	}
}

// RemoveNonUTF ... RemoveNonUTF
func RemoveNonUTF(s string) string {
	if !utf8.ValidString(s) {
		v := make([]rune, 0, len(s))
		for i, r := range s {
			if r == utf8.RuneError {
				_, size := utf8.DecodeRuneInString(s[i:])
				if size == 1 {
					continue
				}
			}
			v = append(v, r)
		}
		s = string(v)
	}
	return s
}
