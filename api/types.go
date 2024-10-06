package api

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Thread struct {
	Quest map[string]interface{} `json:"quest"`
	Extra map[string]interface{} `json:"-"`
}

type Message struct {
	ID      string                 `json:"id"`
	Content string                 `json:"content"`
	Extra   map[string]interface{} `json:"-"`
}

type ThreadsResponse struct {
	Quests []Thread               `json:"quests"`
	Extra  map[string]interface{} `json:"-"`
}

type MessagesResponse struct {
	Answers []Message              `json:"answers"`
	Extra   map[string]interface{} `json:"-"`
}

// UnmarshalJSON implements the json.Unmarshaler interface for Thread
func (t *Thread) UnmarshalJSON(data []byte) error {
	var temp map[string]interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	t.Quest = temp["quest"].(map[string]interface{})
	delete(temp, "quest")
	t.Extra = temp

	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for Message
func (m *Message) UnmarshalJSON(data []byte) error {
	var temp map[string]interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	m.ID = temp["id"].(string)
	m.Content = temp["content"].(string)
	delete(temp, "id")
	delete(temp, "content")
	m.Extra = temp

	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for ThreadsResponse
func (tr *ThreadsResponse) UnmarshalJSON(data []byte) error {
	var temp map[string]interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	quests, ok := temp["quests"].([]interface{})
	if !ok {
		return fmt.Errorf("quests field is not an array")
	}

	tr.Quests = make([]Thread, len(quests))
	for i, q := range quests {
		questData, err := json.Marshal(q)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(questData, &tr.Quests[i]); err != nil {
			return err
		}
	}

	delete(temp, "quests")
	tr.Extra = temp

	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for MessagesResponse
func (mr *MessagesResponse) UnmarshalJSON(data []byte) error {
	var temp map[string]interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	answers, ok := temp["answers"].([]interface{})
	if !ok {
		return fmt.Errorf("answers field is not an array")
	}

	mr.Answers = make([]Message, len(answers))
	for i, a := range answers {
		answerData, err := json.Marshal(a)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(answerData, &mr.Answers[i]); err != nil {
			return err
		}
	}

	delete(temp, "answers")
	mr.Extra = temp

	return nil
}

// ToASCII returns a human-readable representation of a Message
func (m *Message) ToASCII() string {
	userName := "Unknown User"
	if user, ok := m.Extra["user"].(map[string]interface{}); ok {
		if name, ok := user["name"].(string); ok {
			userName = name
		}
	}
	return fmt.Sprintf("%s: %s\n", userName, m.Content)
}

// ToASCII returns a human-readable representation of a Thread
func (t *Thread) ToASCII() string {
	var output strings.Builder

	if parent, ok := t.Quest["parent"].(map[string]interface{}); ok {
		parentMessage := Message{
			ID:      parent["id"].(string),
			Content: parent["content"].(string),
			Extra:   parent,
		}
		output.WriteString(parentMessage.ToASCII())
	}

	if sortedAnswers, ok := t.Quest["sorted_answers"].([]interface{}); ok {
		for _, answer := range sortedAnswers {
			if answerMap, ok := answer.(map[string]interface{}); ok {
				message := Message{
					ID:      answerMap["id"].(string),
					Content: answerMap["content"].(string),
					Extra:   answerMap,
				}
				output.WriteString(message.ToASCII())
			}
		}
	}

	return output.String()
}
