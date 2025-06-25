package llminterface

// SystemInputMessage 创建系统输入消息
func SystemInputMessage(text string) InputMessage {
	return InputMessage{
		Role: RoleSystem,
		Content: []ContentPart{
			{Type: PartTypeText, Text: text},
		},
	}
}

// UserInputMessageText 创建用户输入消息
func UserInputMessageText(text string) InputMessage {
	return InputMessage{
		Role: RoleUser,
		Content: []ContentPart{
			{Type: PartTypeText, Text: text},
		},
	}
}
