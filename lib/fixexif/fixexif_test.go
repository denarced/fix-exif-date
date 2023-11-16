package fixexif

import (
	"testing"
	"time"

	"github.com/denarced/fix-exif-date/shared"
	ji "github.com/dsoprea/go-jpeg-image-structure/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveOffsetSeconds(t *testing.T) {
	run := func(name, date string, dst bool) {
		t.Run(name, func(t *testing.T) {
			shared.InitTestLogging(t)
			// EXERCISE
			seconds := deriveOffsetSeconds(date, "Europe/Helsinki")

			expected := ternary(dst, 3, 2) * 60 * 60
			// VERIFY
			require.Equal(t, expected, seconds)
		})
	}

	run("new year", "2023:01:01 00:00:00", false)
	run("end of standard time 2023", "2023:03:26 02:59:59", false)
	run("start of daylight savings time 2023", "2023:03:26 04:00:00", true)
	run("end of daylight savings time 2023", "2023:10:29 02:59:59", true)
	run("start of standard time 2023", "2023:10:29 04:00:00", false)
}

func TestConvertPrefixToMultiplier(t *testing.T) {
	shared.InitTestLogging(t)
	ass := assert.New(t)
	ass.Equal(1, convertPrefixToMultiplier("+"))
	ass.Equal(-1, convertPrefixToMultiplier("-"))
}

func TestToZone(t *testing.T) {
	shared.InitTestLogging(t)
	req := require.New(t)

	location, err := time.LoadLocation("Europe/Helsinki")
	req.Nil(err, "LoadLocation err should be nil.")
	// EXERCISE
	date, offset := toZone("2023:11:05 17:42:51", "+03:00", location)

	// VERIFY
	req.Equal("2023:11:05 16:42:51", date)
	req.Equal("+02:00", offset)
}

func ternary[T any](which bool, truthy, falsy T) T {
	if which {
		return truthy
	}
	return falsy
}

func TestConvertToOffsetSeconds(t *testing.T) {
	run := func(offsetString string, expectedSeconds int, expectPanic bool) {
		t.Run(offsetString, func(t *testing.T) {
			shared.InitTestLogging(t)
			// EXERCISE & VERIFY
			if expectPanic {
				require.Panics(t, func() { convertToOffsetSeconds(offsetString) })
			} else {
				require.Equal(t, expectedSeconds, convertToOffsetSeconds(offsetString))
			}
		})
	}

	run("+02:00", 2*60*60, false)
	run("-02:00", -2*60*60, false)
	run("+00:00", 0, false)
	run("-00:00", 0, false)
	run("-07:31", -(7*3600 + 31*60), false)
	run("", 0, true)
	run("00:00", 0, true)
	run("0:00:00", 0, true)
	run("+00:00:00", 0, true)
	run("+0x:00", 0, true)
	run("+00:0x", 0, true)
}

func TestExtractStringValue(t *testing.T) {
	shared.InitTestLogging(t)

	jmp := ji.NewJpegMediaParser()
	intfc, err := jmp.ParseFile("testdata/image.jpg")
	// Make = 0x010f
	// EXERCISE
	value, err := extractStringValue(intfc, 0x010f)

	// VERIFY
	ass := assert.New(t)
	ass.Equal("Canon", value)
	ass.Nil(err)
}
