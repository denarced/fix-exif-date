// Package fixexif .
package fixexif

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/denarced/fix-exif-date/shared"
	"github.com/dsoprea/go-exif/v3"
	ji "github.com/dsoprea/go-jpeg-image-structure/v2"
	riimage "github.com/dsoprea/go-utility/v2/image"
)

type updatePairContext struct {
	dateTag       uint16
	dateIfdPath   string
	offsetTag     uint16
	offsetIfdPath string
}

// Output for CLI output.
type Output interface {
	Done(bool)
	PrintFile(string)
	PrintDates(uint16, string, string)
	PrintOffsets(uint16, string, string)
	SkipFile()
}

// FixDate fixes JPG file in path.
func FixDate(filepath string, location *time.Location, output Output) (err error) {
	output.PrintFile(filepath)
	jmp := ji.NewJpegMediaParser()
	intfc, err := jmp.ParseFile(filepath)
	if err != nil {
		return
	}

	updatePair := func(mediaContext riimage.MediaContext, ctx updatePairContext) (err error) {
		dateTimeString, err := extractStringValue(mediaContext, ctx.dateTag)
		if err != nil {
			return
		}
		offsetString, err := extractStringValue(mediaContext, ctx.offsetTag)
		if err != nil {
			return
		}
		targetDate, targetOffset := toZone(dateTimeString, offsetString, location)
		if offsetString == targetOffset {
			output.SkipFile()
			shared.Logger.Info().Msg("Timezone offset is correct, nothing to do.")
			return
		}

		shared.Logger.Info().
			Str("date", dateTimeString).
			Str("offset", offsetString).
			Msg("Old values.")
		shared.Logger.Info().Str("date", targetDate).Str("offset", targetOffset).Msg("New values.")
		output.PrintDates(ctx.dateTag, dateTimeString, targetDate)
		output.PrintOffsets(ctx.offsetTag, offsetString, targetOffset)

		sl := mediaContext.(*ji.SegmentList)
		rootIb, err := sl.ConstructExifBuilder()
		if err != nil {
			return
		}

		ifdIb, err := exif.GetOrCreateIbFromRootIb(rootIb, ctx.dateIfdPath)
		if err != nil {
			return
		}

		err = ifdIb.SetStandard(ctx.dateTag, targetDate)
		if err != nil {
			return
		}
		first, err := exif.GetOrCreateIbFromRootIb(rootIb, ctx.offsetIfdPath)
		if err != nil {
			return
		}
		err = first.SetStandard(ctx.offsetTag, targetOffset)
		if err != nil {
			return
		}

		return sl.SetExif(rootIb)
	}

	pairs := []updatePairContext{
		{0x0132, "IFD0", 0x9010, "IFD0/Exif0"},
		{0x9003, "IFD0/Exif0", 0x9011, "IFD0/Exif0"},
		{0x9004, "IFD0/Exif0", 0x9012, "IFD0/Exif0"},
	}
	for _, each := range pairs {
		err = updatePair(intfc, each)
		if err != nil {
			return
		}
	}

	out, err := os.Create(filepath)
	if err != nil {
		return
	}
	defer out.Close()

	err = intfc.(*ji.SegmentList).Write(out)
	output.Done(err == nil)
	return err
}

func extractStringValue(mediaContext riimage.MediaContext, tag uint16) (string, error) {
	parsedExif, _, err := mediaContext.Exif()
	if err != nil {
		shared.Logger.Error().Err(err).Msg("Failed to extract exif.")
		return "", err
	}
	var foundValue string
	visitor := func(i *exif.Ifd, e *exif.IfdTagEntry) error {
		if foundValue != "" {
			return nil
		}
		if e.TagId() == tag {
			value, err := e.Value()
			shared.Logger.Info().Uint16("tag", tag).Any("Value", value).Msg("Found value.")
			if err != nil {
				return err
			}
			foundValue = value.(string)
		}
		return nil
	}
	err = parsedExif.EnumerateTagsRecursively(visitor)
	if err != nil {
		return "", err
	}
	if foundValue == "" {
		return "", fmt.Errorf("failed to find value")
	}
	return foundValue, nil
}

func deriveOffsetSeconds(date string, timezone string) int {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		shared.Logger.Error().Err(err).Msg("Failed to load timezone location.")
		panic(fmt.Sprintf("Invalid timezone: %s", timezone))
	}
	// 2006-01-02T15:04:05.999999999Z07:00
	parsed, err := time.ParseInLocation("2006:01:02 15:04:05", date, location)
	if err != nil {
		shared.Logger.Error().Err(err).Msg("Failed to parse date.")
		panic(fmt.Sprintf("Failed to parse date"))
	}
	_, seconds := parsed.Zone()
	return seconds
}

func convertToOffsetSeconds(offset string) int {
	multiplier := convertPrefixToMultiplier(offset[0:1])
	rest := offset[1:]
	pieces := strings.Split(rest, ":")
	if len(pieces) != 2 {
		panic(fmt.Sprintf("Invalid offset: %s", offset))
	}
	hours, err := strconv.Atoi(pieces[0])
	if err != nil {
		panic(fmt.Sprintf("Invalid offset: %s", offset))
	}
	minutes, err := strconv.Atoi(pieces[1])
	if err != nil {
		panic(fmt.Sprintf("Invalid offset: %s", offset))
	}
	return multiplier * (hours*3600 + minutes*60)
}

func convertPrefixToMultiplier(prefix string) int {
	switch prefix {
	case "+":
		return 1
	case "-":
		return -1
	default:
		panic(fmt.Sprintf("Invalid prefix: %s", prefix))
	}
}

func toZone(date, offset string, location *time.Location) (string, string) {
	parsedDate, err := time.Parse("2006:01:02 15:04:05-07:00", date+offset)
	soPanic(err, "Failed to parse time")
	inLocation := parsedDate.In(location)
	formatted := inLocation.Format("2006:01:02 15:04:05#-07:00")
	pieces := strings.Split(formatted, "#")
	return pieces[0], pieces[1]
}

func soPanic(err error, message string) {
	if err == nil {
		return
	}
	shared.Logger.Error().Err(err).Msg(message)
	panic(message)
}
