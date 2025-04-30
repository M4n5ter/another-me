package trigger

type TriggerType uint8

const (
	TriggerTypeTimer     TriggerType = iota // 定时触发器
	TriggerTypeUserEvent                    // 用户事件触发器
	TriggerTypeAPICall                      // API 调用触发器
)

func (t TriggerType) String() string {
	switch t {
	case TriggerTypeTimer:
		return "Timer"
	case TriggerTypeUserEvent:
		return "UserEvent"
	case TriggerTypeAPICall:
		return "APICall"
	default:
		return "Unknown"
	}
}
