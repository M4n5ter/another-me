package llminterface

func SystemInputMessage(text string) InputMessage {
	return InputMessage{
		Role: RoleSystem,
		Content: []ContentPart{
			{Type: PartTypeText, Text: text},
		},
	}
}

func UserInputMessageText(text string) InputMessage {
	return InputMessage{
		Role: RoleUser,
		Content: []ContentPart{
			{Type: PartTypeText, Text: text},
		},
	}
}
