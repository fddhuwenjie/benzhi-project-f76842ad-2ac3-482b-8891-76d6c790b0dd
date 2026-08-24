package httpapi

import (
	"encoding/json"
	"fmt"
	"time"
)

type domainTime struct{ time.Time }

func (v *domainTime) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("时间必须是 RFC3339 字符串")
	}
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return fmt.Errorf("时间必须使用 RFC3339 格式")
	}
	v.Time = parsed
	return nil
}
