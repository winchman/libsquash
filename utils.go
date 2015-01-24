package libsquash

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

func humanDuration(d time.Duration) string {
	if seconds := int(d.Seconds()); seconds < 1 {
		return "Less than a second"
	} else if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	} else if minutes := int(d.Minutes()); minutes == 1 {
		return "About a minute"
	} else if minutes < 60 {
		return fmt.Sprintf("%d minutes", minutes)
	} else if hours := int(d.Hours()); hours == 1 {
		return "About an hour"
	} else if hours < 48 {
		return fmt.Sprintf("%d hours", hours)
	} else if hours < 24*7*2 {
		return fmt.Sprintf("%d days", hours/24)
	} else if hours < 24*30*3 {
		return fmt.Sprintf("%d weeks", hours/24/7)
	} else if hours < 24*365*2 {
		return fmt.Sprintf("%d months", hours/24/30)
	}
	return fmt.Sprintf("%f years", d.Hours()/24/365)
}

func truncateID(id string) string {
	shortLen := 12
	if len(id) < shortLen {
		shortLen = len(id)
	}
	return id[:shortLen]
}

func newID() (string, error) {
	for {
		id := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, id); err != nil {
			return "", err
		}
		value := hex.EncodeToString(id)
		if _, err := strconv.ParseInt(truncateID(value), 10, 64); err == nil {
			continue
		}
		return value, nil
	}
}

func isWhiteout(filepath string) bool {
	nameParts := strings.Split(filepath, string(os.PathSeparator))
	fileName := nameParts[len(nameParts)-1]
	return strings.HasPrefix(fileName, ".wh.")
}

func nameWithoutWhiteoutPrefix(filepath string) string {
	return strings.Replace(filepath, ".wh.", "", -1)
}

func matchesWhiteout(filename string, whiteouts []whiteoutFile) (uuidContainingWhiteout string, matches bool) {
	for _, whiteout := range whiteouts {
		if strings.HasPrefix(filename, whiteout.prefix) {
			return whiteout.uuid, true
		}
	}
	return "", false
}
