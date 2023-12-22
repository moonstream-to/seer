package starknet

import (
	"encoding/json"
)

type EnumVariant struct {
	Name  string `json:"name"`
	Index int    `json:"index"`
}

type Enum struct {
	Type     string         `json:"type"`
	Name     string         `json:"name"`
	Variants []*EnumVariant `json:"variants"`
}

type StructMember struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Kind string `json:"kind,omitempty"`
}

type Struct struct {
	Type    string          `json:"type"`
	Name    string          `json:"name"`
	Members []*StructMember `json:"members"`
}

type EventStruct struct {
	Type    string          `json:"type"`
	Name    string          `json:"name"`
	Kind    string          `json:"kind"`
	Members []*StructMember `json:"members"`
}

type ABIItemType struct {
	Type string `json:"type,omitempty"`
	Kind string `json:"kind,omitempty"`
}

type ParsedABI struct {
	Enums   []*Enum        `json:"enums"`
	Structs []*Struct      `json:"structs"`
	Events  []*EventStruct `json:"events"`
}

type IntermediateABI struct {
	Types    []ABIItemType
	Messages []json.RawMessage
}

func ParseABI(rawABI []byte) (*ParsedABI, error) {
	parsedABI := &ParsedABI{}

	var itemTypes []ABIItemType
	var rawMessages []json.RawMessage
	intermediateUnmarshalErr := json.Unmarshal(rawABI, &itemTypes)
	if intermediateUnmarshalErr != nil {
		return parsedABI, intermediateUnmarshalErr
	}

	messagesUnmarshalErr := json.Unmarshal(rawABI, &rawMessages)
	if messagesUnmarshalErr != nil {
		return parsedABI, messagesUnmarshalErr
	}

	numEnums := 0
	numStructs := 0
	numEvents := 0
	for _, item := range itemTypes {
		switch item.Type {
		case "enum":
			numEnums++
		case "struct":
			numStructs++
		case "event":
			if item.Kind == "struct" {
				numEvents++
			}
		}
	}

	parsedABI.Enums = make([]*Enum, numEnums)
	currentEnum := 0
	for i, item := range itemTypes {
		if item.Type == "enum" {
			var enum *Enum
			enumUnmarshalErr := json.Unmarshal(rawMessages[i], &enum)
			if enumUnmarshalErr != nil {
				return parsedABI, enumUnmarshalErr
			}

			parsedABI.Enums[currentEnum] = enum
			currentEnum++
		}
	}

	for _, item := range parsedABI.Enums {
		for i, variant := range item.Variants {
			variant.Index = i
		}
	}

	parsedABI.Structs = make([]*Struct, numStructs)
	currentStruct := 0
	for i, item := range itemTypes {
		if item.Type == "struct" {
			var structItem *Struct
			structUnmarshalErr := json.Unmarshal(rawMessages[i], &structItem)
			if structUnmarshalErr != nil {
				return parsedABI, structUnmarshalErr
			}

			parsedABI.Structs[currentStruct] = structItem
			currentStruct++
		}
	}

	parsedABI.Events = make([]*EventStruct, numEvents)
	currentEvent := 0
	for i, item := range itemTypes {
		if item.Type == "event" && item.Kind == "struct" {
			var event *EventStruct
			eventUnmarshalErr := json.Unmarshal(rawMessages[i], &event)
			if eventUnmarshalErr != nil {
				return parsedABI, eventUnmarshalErr
			}

			parsedABI.Events[currentEvent] = event
			currentEvent++
		}
	}

	return parsedABI, nil
}
