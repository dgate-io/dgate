package spec

import (
	"encoding/json"
	"fmt"
	"time"
)

type Duration time.Duration

func (duration *Duration) UnmarshalJSON(b []byte) (err error) {
	var jsonData interface{}
	err = json.Unmarshal(b, &jsonData)
	if err != nil {
		return err
	}
	switch value := jsonData.(type) {
	case float64:
		*duration = Duration(time.Duration(value))
	case string:
		dur, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*duration = Duration(dur)
	default:
		return fmt.Errorf("invalid duration: %#v", jsonData)
	}
	return nil
}

func (duration Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(duration.String())
}

func (duration Duration) TimeDuration() time.Duration {
	return time.Duration(duration)
}

func (duration Duration) String() string {
	return time.Duration(duration).String()
}
